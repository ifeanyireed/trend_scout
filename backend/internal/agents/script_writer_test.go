package agents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"trendscout-backend/internal/db"
)

func TestDynamicBiographicalDomainDetection(t *testing.T) {
	cases := []struct {
		name      string
		category  string
		bio       string
		override  string
		wantKey   string
		wantLabel string
	}{
		{
			name:      "Tech Founder",
			category:  "Technology & Innovation",
			bio:       "Built a social media platform in a Harvard dorm room, scaled to billions, testified before the Senate Judiciary Committee.",
			wantKey:   "tech_innovation",
			wantLabel: "Technology & Innovation",
		},
		{
			name:      "Scientist / Medicine",
			category:  "Science & Medicine",
			bio:       "Nobel laureate molecular biologist who worked in a university laboratory developing mRNA vaccine technology and faced peer-reviewed grant rejections.",
			wantKey:   "science_medicine",
			wantLabel: "Science & Medicine",
		},
		{
			name:      "Politics & Law",
			category:  "Politics",
			bio:       "Served as prosecutor, elected senator, and argued constitutional legislation before the Supreme Court.",
			wantKey:   "politics_law",
			wantLabel: "Politics, Law & Governance",
		},
		{
			name:      "Sports Athlete",
			category:  "Sports & Competition",
			bio:       "Unseeded tennis champion who won an Olympic gold medal after overcoming injury and regional tournament defeats.",
			wantKey:   "sports_competition",
			wantLabel: "Sports & Competition",
		},
		{
			name:      "Arts & Film",
			category:  "Arts & Culture",
			bio:       "Independent film director and novelist whose screenplay premiered at international festivals and won an Oscar.",
			wantKey:   "arts_culture",
			wantLabel: "Arts, Film & Music",
		},
		{
			name:      "Explicit Field Override",
			category:  "Sports & Competition",
			bio:       "Olympic athlete turned startup founder.",
			override:  "Business & Founders",
			wantKey:   "business_founders",
			wantLabel: "Business & Founders",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dom := ResolveBiographicalDomain(tc.category, tc.bio, "", "", tc.override)
			if dom.Key != tc.wantKey || dom.Label != tc.wantLabel {
				t.Fatalf("expected (%s, %s), got (%s, %s)", tc.wantKey, tc.wantLabel, dom.Key, dom.Label)
			}
		})
	}
}

func TestSplitLongNarrationEnforcesSevenSecondCutaways(t *testing.T) {
	dom := domainScienceMedicine()
	input := BiographyInput{
		PersonName: "Dr. Ada Sterling",
		Category:   "Science & Medicine",
		Summary:    "Physicist whose laboratory breakthrough survived fourteen years of skepticism.",
		Biography:  "In October 2020, after fourteen years of clinical skepticism, three independent review committees confirmed her diagnostic method reduced errors by 84 percent across 42 hospitals.",
	}
	longSentence := "In October 2020, after fourteen years of clinical skepticism, three independent review committees confirmed that Dr. Ada Sterling's laboratory method reduced diagnostic error rates by eighty-four percent across forty-two regional hospitals while triggering an immediate international patent dispute."

	sections := []ScriptSection{
		{
			ActNumber:     1,
			ChapterNumber: 0,
			Title:         "0 · Cold Open",
			Shots: []ScriptShot{
				{
					Visual:          "Wide shot of a scientific symposium auditorium.",
					Narration:       longSentence,
					DurationSeconds: 18, // intentionally violating 7s rule
				},
			},
		},
	}

	enforced := EnforceMaxSevenSecondPacing(sections, dom, input, 15)
	if len(enforced[0].Shots) < 10 {
		t.Fatalf("expected pacing enforcer to split and fill cold open chapter, got %d shots", len(enforced[0].Shots))
	}
	distinctVisuals := map[string]bool{}
	var prevStart float64 = -1
	for _, sh := range enforced[0].Shots {
		if sh.DurationSeconds > MaxShotDurationSeconds {
			t.Fatalf("enforcer left shot exceeding 7s: %.1fs", sh.DurationSeconds)
		}
		if sh.DurationSeconds < MinShotDurationSeconds {
			t.Fatalf("enforcer left shot under 3s: %.1fs", sh.DurationSeconds)
		}
		if prevStart >= 0 && (sh.StartSeconds-prevStart) > MaxShotDurationSeconds+0.01 {
			t.Fatalf("timestamp gap exceeded 7s between %.1fs and %.1fs", prevStart, sh.StartSeconds)
		}
		prevStart = sh.StartSeconds
		distinctVisuals[sh.Visual] = true
		if sh.Narration != "" && len(strings.Fields(sh.Narration)) > MaxWordsPerShot+1 {
			t.Fatalf("enforcer left narration too long for 7s: %d words (%q)", len(strings.Fields(sh.Narration)), sh.Narration)
		}
	}
	if len(distinctVisuals) < 3 {
		t.Fatalf("expected distinct fresh visuals when splitting long speech, got only %d", len(distinctVisuals))
	}

	rendered := FormatRealignedScript(enforced)
	if strings.Contains(rendered, "IN-BETWEEN") {
		t.Fatalf("formatted script must NOT contain [NEW IN-BETWEEN] editorial markers")
	}
	spoken := ExtractSpokenNarration(rendered)
	if strings.Contains(spoken, "VISUAL:") {
		t.Fatalf("ExtractSpokenNarration leaked visual stage directions")
	}
	reparsed := ParseRealignedScriptShots(rendered, dom)
	if len(reparsed) < 10 {
		t.Fatalf("expected >= 10 reparsed shots, got %d", len(reparsed))
	}
	for _, rs := range reparsed {
		if rs.DurationSeconds > MaxShotDurationSeconds {
			t.Fatalf("reparsed shot %d exceeded 7s: %.1fs", rs.ShotNumber, rs.DurationSeconds)
		}
	}
}

func TestVideoPackagingTagAndHashtagNormalization(t *testing.T) {
	rawTags := []string{
		"#Elena Vance", "elena vance", "Elena Vance documentary", "underdog documentary",
		"encrypted messaging history", "parliamentary testimony", "tech founder biography",
		"true story documentary", "real life underdog", "platform governance",
		"silicon valley investigation", "full biography film", "investigative journalism",
	}
	tags := NormalizeTags(rawTags, 480)
	if len(tags) < 10 {
		t.Fatalf("expected deduplicated tags, got %d", len(tags))
	}
	if TagsTotalChars(tags) > 480 {
		t.Fatalf("expected total tag characters <= 480, got %d", TagsTotalChars(tags))
	}
	for _, tag := range tags {
		if strings.HasPrefix(tag, "#") {
			t.Fatalf("SEO tag should not start with #: %q", tag)
		}
	}

	hashtags := NormalizeHashtags([]string{"elena vance", "#ElenaVance", "Underdog Story!", "documentary", "True Story"}, 12)
	if len(hashtags) != 4 {
		t.Fatalf("expected 4 unique normalized hashtags, got %v", hashtags)
	}
	for _, h := range hashtags {
		if !strings.HasPrefix(h, "#") || strings.Contains(h, " ") {
			t.Fatalf("invalid hashtag format: %q", h)
		}
	}
}

func TestCompetitiveBriefPromptFormatting(t *testing.T) {
	brief := &CompetitivePositioningBrief{
		AnalyzedVideoCount:       3,
		TranscribedVideoCount:    2,
		CommentsAnalyzedCount:    24,
		OutperformanceStrategy:   "Beat Curly Tales' 59-second clip by opening on the 2019 elbow surgery.",
		RecommendedColdOpenEdge:  "Open inside the 2019 rehabilitation clinic before cutting to Tokyo.",
		UnderdogOpeningArchetype: "The Hidden Low Point",
		FormatAdvantage:          "All 3 top clips are under 2 minutes; a 20-minute film owns search.",
		WinningHooks:             []string{"Curly Tales 41.5x Short on Panipat roots"},
		CompetitorBlindSpots:     []string{"Zero coverage of the 2019 surgery"},
		AudienceQuestions:        []string{"Someone needs to make a full documentary on his injury comeback!"},
		ScriptExecutionPlan:      []string{"Ch 0: Open on 2019 surgery", "Ch 3: Detail rehab"},
		UsedGemini:               true,
	}
	prompt := FormatCompetitiveBriefForPrompt(brief)
	for _, want := range []string{
		"OUTPERFORMANCE STRATEGY:",
		"COLD OPEN EDGE:",
		"UNDERDOG OPENING ARCHETYPE: The Hidden Low Point",
		"CHAPTER EXECUTION PLAN:",
		"injury comeback",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("formatted prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestStrictAIRequiredNoFallbacks(t *testing.T) {
	// Ensure zero Gemini key in env during this test
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_API", "")

	ctx := context.Background()
	dom := domainSportsCompetition()

	// 1. Stage 0 Competitor Study MUST return error when no AI key is configured
	if _, _, err := BuildProjectCompetitiveBrief(ctx, "", "", "", "Neeraj Chopra", dom, 20); err == nil {
		t.Fatalf("expected BuildProjectCompetitiveBrief to return error without Gemini AI key (no fallback)")
	}

	// 2. YouTube SEO Packaging MUST return error when no AI key is configured
	if _, err := generateVideoSEOWithAI(ctx, "Neeraj Chopra", dom, "India", "Headline", "Achievement", "Bio", "0:00 Cold Open", "", 20); err == nil {
		t.Fatalf("expected generateVideoSEOWithAI to return error without Gemini AI key (no fallback)")
	}

	// 3. Shot/Block refinement MUST return error when Gemini refiner fails (no fallback)
	_, err := RegenerateScriptBlock(
		"Neeraj Chopra", "Sports & Competition", "Bio", "Summary",
		"[0:00] VISUAL: Wide shot.\nNARRATOR (V.O.): Test line.",
		0, "Investigative",
		func(personName, field, existingBlock, biography, tone string) (string, string, error) {
			return "", "", fmt.Errorf("gemini api error (401): unauthorized")
		},
	)
	if err == nil {
		t.Fatalf("expected RegenerateScriptBlock to return error when AI fails (no fallback)")
	}
}

func TestLiveAIDocumentaryPipeline(t *testing.T) {
	if os.Getenv("RUN_LIVE_AI_TEST") != "1" {
		t.Skip("set RUN_LIVE_AI_TEST=1 to run live end-to-end AI Stage 0 + Stage 1 + Stage 2 documentary generation")
	}
	db.LoadEnvFile()
	if _, err := db.ConnectAndMigrate(); err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	ctx := context.Background()
	projectID := os.Getenv("LIVE_TEST_PROJECT_ID")
	if projectID == "" {
		projectID = "proj-neeraj-chopra-8ac3"
	}
	scrID, err := GenerateDocumentaryScriptWithOptions(ctx, projectID, 20, "Investigative Archival Documentary", "auto")
	if err != nil {
		t.Fatalf("live AI documentary script generation failed: %v", err)
	}
	var hook, metaJSON, fullText string
	var words int
	if err := db.DB.QueryRowContext(ctx, `SELECT opening_hook, generation_meta_json, word_count, full_narration_text FROM documentary_scripts WHERE id = $1`, scrID).Scan(&hook, &metaJSON, &words, &fullText); err != nil {
		t.Fatalf("failed reading saved script: %v", err)
	}
	t.Logf("LIVE AI SCRIPT SUCCEEDED: id=%s words=%d meta=%s", scrID, words, metaJSON)
	t.Logf("OPENING HOOK: %s", hook)
	if len(fullText) > 500 {
		t.Logf("SCRIPT PREVIEW:\n%s", fullText[:500])
	}
}
