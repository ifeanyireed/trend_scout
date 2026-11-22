package agents

import (
	"strings"
	"testing"

	"trendscout-backend/internal/connectors"
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

func TestDocumentaryScriptVolumeStyleAndSevenSecondRule(t *testing.T) {
	subjects := []struct {
		name     string
		duration int
		input    BiographyInput
		override string
		minShots int
	}{
		{
			name:     "20-Minute Tech Biography",
			duration: 20,
			minShots: 165,
			input: BiographyInput{
				PersonName:        "Elena Vance",
				Category:          "Technology & Innovation",
				Country:           "United States",
				ProjectTitle:      "The Protocol: Inside Elena Vance's Network",
				Summary:           "Elena Vance built an open messaging protocol in a university basement that scaled to 900 million users before triggering international privacy inquiries.",
				Biography:         "Elena Vance was born in 1986 in Seattle. Her parents were public school teachers. In 2005, while studying computer science, she wrote a distributed encryption router over a single weekend. In 2008, two former collaborators sued over intellectual property ownership; the case settled out of court in 2010 under sealed terms. By 2016, her platform surpassed 500 million daily users and she pledged 80 percent of her shares to an independent digital literacy foundation. In 2024, she testified before parliamentary committees over algorithmic safety and youth privacy allegations, which the company denied.",
				Headline:          "Elena Vance faces parliamentary testimony after two decades of platform growth",
				EventTitle:        "2024 Parliamentary Digital Safety Inquiry",
				Origins:           "Born in Seattle to two teachers, she learned systems programming on surplus university terminals.",
				EarlyAmbitions:    "She set out to build resilient peer-to-peer communication software independent of telecom carriers.",
				Disadvantages:     "Operating with zero institutional venture backing during the first three years.",
				Rejections:        "Early security audits and major carriers dismissed the protocol as unscalable.",
				Obstacles:         "Competing lawsuits, regulatory subpoenas across three jurisdictions, and internal governance disputes.",
				TurningPoints:     "A 2011 open-source release triggered global adoption across journalists and students.",
				CareerProgression: "Expanded from a three-person apartment team to a multinational software platform.",
				Breakthrough:      "Reached 900 million active users while navigating landmark privacy litigation.",
				Claims: []ScriptClaimRef{
					{ClaimID: "c1", SourceID: "s1", Category: "Origins", Text: "Born in 1986 in Seattle and studied computer science.", URL: "https://en.wikipedia.org/wiki/Example", Status: "Verified"},
					{ClaimID: "c2", SourceID: "s1", Category: "Dispute", Text: "Former collaborators sued in 2008 and settled under sealed terms in 2010.", URL: "https://en.wikipedia.org/wiki/Example", Status: "Reported"},
				},
			},
		},
		{
			name:     "15-Minute Science Biography",
			duration: 15,
			minShots: 120,
			input: BiographyInput{
				PersonName:   "Dr. Katalin Solis",
				Category:     "Science & Medicine",
				Country:      "Hungary",
				ProjectTitle: "The Sequence: Dr. Katalin Solis",
				Summary:      "Biochemist whose decades of rejected grant proposals culminated in a transformative therapeutic breakthrough.",
				Biography:    "Born in a small town in 1958, Dr. Katalin Solis studied biochemistry at university and emigrated with her family in 1985. For fourteen years, major funding committees rejected her RNA stability proposals and she was demoted from a tenure-track post. In 2005, she co-authored a landmark paper modifying synthetic nucleosides that initially received muted attention. Fifteen years later, global clinical trials validated the mechanism at unprecedented speed, earning international prizes alongside intense patent disputes.",
				Breakthrough: "Nucleoside-modified RNA breakthrough deployed globally across clinical medicine.",
			},
		},
		{
			name:     "25-Minute Politics & Law Biography",
			duration: 25,
			minShots: 205,
			input: BiographyInput{
				PersonName:   "Marcus Okafor",
				Category:     "Politics, Law & Governance",
				Country:      "Nigeria",
				ProjectTitle: "The Ledger of Office: Marcus Okafor",
				Summary:      "Former municipal prosecutor who challenged entrenched procurement networks before facing a national inquiry.",
				Biography:    "Marcus Okafor grew up in Enugu and studied constitutional law. As a young public prosecutor in 2004, he filed landmark anti-corruption indictments against regional contractors. Elected governor in 2015, he overhauled state digital audits and public schools. In 2023, opposition lawmakers initiated an oversight tribunal examining emergency infrastructure contracts, allegations Okafor publicly disputed in sworn testimony.",
				Breakthrough: "Led sweeping public procurement transparency reforms across state ministries.",
			},
		},
	}

	for _, tc := range subjects {
		t.Run(tc.name, func(t *testing.T) {
			dom := ResolveBiographicalDomain(tc.input.Category, tc.input.Biography, tc.input.Summary, tc.input.Headline, tc.override)
			hook, sections := BuildDeterministicDocumentaryScript(tc.input, dom, tc.duration)

			if strings.TrimSpace(hook) == "" {
				t.Fatalf("expected non-empty opening hook")
			}
			if len(sections) != 7 {
				t.Fatalf("expected exactly 7 chapters (0..6), got %d", len(sections))
			}

			totalShots := 0
			var prevStart float64 = -1
			for chIdx, sec := range sections {
				if sec.ChapterNumber != chIdx {
					t.Fatalf("chapter %d has wrong chapterNumber %d", chIdx, sec.ChapterNumber)
				}
				if len(sec.Shots) == 0 {
					t.Fatalf("chapter %d has zero shots", chIdx)
				}
				for _, sh := range sec.Shots {
					totalShots++
					if sh.DurationSeconds > MaxShotDurationSeconds {
						t.Fatalf("shot %d (%s) violated <= 7s rule: duration=%.1fs (narration=%q)", sh.ShotNumber, sh.Timestamp, sh.DurationSeconds, sh.Narration)
					}
					if sh.DurationSeconds < MinShotDurationSeconds {
						t.Fatalf("shot %d (%s) below min duration 3s: duration=%.1fs", sh.ShotNumber, sh.Timestamp, sh.DurationSeconds)
					}
					if prevStart >= 0 && (sh.StartSeconds-prevStart) > MaxShotDurationSeconds+0.01 {
						t.Fatalf("timestamp gap exceeded 7s between %.1fs and %.1fs at shot %d", prevStart, sh.StartSeconds, sh.ShotNumber)
					}
					prevStart = sh.StartSeconds
					if sh.Narration != "" {
						wCount := len(strings.Fields(sh.Narration))
						if wCount > MaxWordsPerShot+1 {
							t.Fatalf("shot %d narration too long for <=7s speech (%d words): %q", sh.ShotNumber, wCount, sh.Narration)
						}
					}
					if strings.TrimSpace(sh.Visual) == "" {
						t.Fatalf("shot %d missing visual description", sh.ShotNumber)
					}
				}
			}

			if totalShots < tc.minShots {
				t.Fatalf("expected >= %d shots for %d-min documentary, got %d", tc.minShots, tc.duration, totalShots)
			}

			rendered := FormatRealignedScript(sections)
			if !strings.Contains(rendered, "0 · Cold Open (0:00 – ") {
				t.Fatalf("formatted script missing Cold Open chapter header")
			}
			if strings.Contains(rendered, "IN-BETWEEN") {
				t.Fatalf("formatted script must NOT contain [NEW IN-BETWEEN] editorial markers")
			}
			if !strings.Contains(rendered, "VISUAL:") {
				t.Fatalf("formatted script missing VISUAL: cues")
			}
			if !strings.Contains(rendered, "NARRATOR (V.O.):") {
				t.Fatalf("formatted script missing NARRATOR (V.O.): lines")
			}

			spoken := ExtractSpokenNarration(rendered)
			if strings.Contains(spoken, "VISUAL:") {
				t.Fatalf("ExtractSpokenNarration leaked visual stage directions")
			}
			if len(strings.Fields(spoken)) < 600 {
				t.Fatalf("expected rich documentary monologue volume, got only %d spoken words", len(strings.Fields(spoken)))
			}

			// Round-trip parse verification: parsed shots also obey <= 7.0s strictly
			reparsed := ParseRealignedScriptShots(rendered, dom)
			if len(reparsed) < tc.minShots {
				t.Fatalf("expected >= %d reparsed shots, got %d", tc.minShots, len(reparsed))
			}
			for _, rs := range reparsed {
				if rs.DurationSeconds > MaxShotDurationSeconds {
					t.Fatalf("reparsed shot %d exceeded 7s: %.1fs", rs.ShotNumber, rs.DurationSeconds)
				}
			}
		})
	}
}

func TestSplitLongNarrationEnforcesSevenSecondCutaways(t *testing.T) {
	dom := domainScienceMedicine()
	input := BiographyInput{
		PersonName: "Dr. Ada Sterling",
		Category:   "Science & Medicine",
		Summary:    "Physicist.",
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
	for _, sh := range enforced[0].Shots {
		if sh.DurationSeconds > MaxShotDurationSeconds {
			t.Fatalf("enforcer left shot exceeding 7s: %.1fs", sh.DurationSeconds)
		}
		distinctVisuals[sh.Visual] = true
		if sh.Narration != "" && len(strings.Fields(sh.Narration)) > MaxWordsPerShot+1 {
			t.Fatalf("enforcer left narration too long for 7s: %d words (%q)", len(strings.Fields(sh.Narration)), sh.Narration)
		}
	}
	if len(distinctVisuals) < 3 {
		t.Fatalf("expected distinct fresh visuals when splitting long speech, got only %d", len(distinctVisuals))
	}
}

func TestVideoPackagingDescriptionTagsAndHashtags(t *testing.T) {
	dom := domainTechInnovation()
	seo := BuildDeterministicVideoPackaging(
		"Elena Vance",
		dom,
		"United States",
		"Elena Vance faces parliamentary testimony after two decades of platform growth",
		"Built an encrypted network used by 900 million people before international privacy hearings.",
		"Traces Elena Vance from a university basement to multinational platform governance.",
		"",
		20,
	)

	if !strings.Contains(seo.VideoDescription, "Chapters:") || !strings.Contains(seo.VideoDescription, "0:00 Cold Open") {
		t.Fatalf("expected Chapters timestamps in video description, got:\n%s", seo.VideoDescription)
	}
	if len(seo.Tags) < 12 {
		t.Fatalf("expected >= 12 high-performing SEO tags, got %d", len(seo.Tags))
	}
	if seo.TagsCharCount > 500 {
		t.Fatalf("expected total tag characters <= 500 for YouTube limit, got %d", seo.TagsCharCount)
	}
	for _, tag := range seo.Tags {
		if strings.HasPrefix(tag, "#") {
			t.Fatalf("SEO tag should not start with #: %q", tag)
		}
	}
	if len(seo.Hashtags) < 6 {
		t.Fatalf("expected >= 6 hashtags, got %d", len(seo.Hashtags))
	}
	for _, h := range seo.Hashtags {
		if !strings.HasPrefix(h, "#") || strings.Contains(h, " ") {
			t.Fatalf("invalid hashtag format: %q", h)
		}
	}
}

func TestCompetitiveBriefSynthesisAndPromptFormatting(t *testing.T) {
	dom := domainSportsCompetition()
	studies := []connectors.CompetitorVideoStudy{
		{
			VideoID:             "41vUn1McRjE",
			Title:               "59 Seconds With Neeraj Chopra #shorts",
			ChannelName:         "Curly Tales",
			URL:                 "https://www.youtube.com/watch?v=41vUn1McRjE",
			Views:               13_806_175,
			ViewsPerHour:        435,
			ChannelMultiplier:   41.5,
			EngagementRate:      6.21,
			VideoFormat:         "Short (<60s)",
			DurationSeconds:     59,
			TranscriptExcerpt:   "Growing up in Panipat nobody thought javelin would change everything.",
			TranscriptAvailable: true,
			TopComments:         []string{"[1400 likes] Someone needs to make a full documentary on his injury comeback!"},
		},
	}

	brief := BuildDeterministicCompetitiveBrief("Neeraj Chopra", dom, studies, 20)
	if brief.AnalyzedVideoCount != 1 || brief.TranscribedVideoCount != 1 || brief.CommentsAnalyzedCount != 1 {
		t.Fatalf("unexpected study counters: %+v", brief)
	}
	if !strings.Contains(brief.FormatAdvantage, "Short") || brief.OutperformanceStrategy == "" {
		t.Fatalf("expected format gap advantage & strategy, got %+v", brief)
	}
	prompt := FormatCompetitiveBriefForPrompt(&brief)
	if !strings.Contains(prompt, "OUTPERFORMANCE STRATEGY:") || !strings.Contains(prompt, "injury comeback") {
		t.Fatalf("formatted prompt missing strategy or viewer comment signal:\n%s", prompt)
	}
}

