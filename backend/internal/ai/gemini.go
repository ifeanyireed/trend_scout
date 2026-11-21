package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"trendscout-backend/internal/db"
)

type AIConfig struct {
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	GeminiAPIKey  string `json:"geminiApiKey"`
	YouTubeAPIKey string `json:"youtubeApiKey"`
}

// GetAIConfig resolves Gemini & YouTube keys from user_settings (PostgreSQL) or environment variables (.env).
func GetAIConfig(ctx context.Context) AIConfig {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API"))
	}

	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = "gemini-2.5-flash"
	}

	cfg := AIConfig{
		Provider:      "Google Gemini",
		Model:         model,
		GeminiAPIKey:  apiKey,
		YouTubeAPIKey: strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY")),
	}

	if db.DB != nil {
		var aiJSON string
		if err := db.DB.QueryRowContext(ctx, `SELECT ai_prefs_json FROM user_settings LIMIT 1`).Scan(&aiJSON); err == nil && aiJSON != "" {
			var m map[string]any
			if json.Unmarshal([]byte(aiJSON), &m) == nil {
				if v, ok := m["geminiApiKey"].(string); ok && strings.TrimSpace(v) != "" {
					cfg.GeminiAPIKey = strings.TrimSpace(v)
				}
				if v, ok := m["youtubeApiKey"].(string); ok && strings.TrimSpace(v) != "" {
					cfg.YouTubeAPIKey = strings.TrimSpace(v)
				}
				if v, ok := m["model"].(string); ok && strings.TrimSpace(v) != "" {
					modelVal := strings.TrimSpace(v)
					if strings.HasPrefix(modelVal, "gemini-") && !strings.Contains(modelVal, "/") {
						cfg.Model = modelVal
					}
				}
			}
		}
	}
	return cfg
}

func HasGeminiKey(ctx context.Context) bool {
	return GetAIConfig(ctx).GeminiAPIKey != ""
}

type geminiRequest struct {
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
	Contents          []geminiContent      `json:"contents"`
	GenerationConfig  *geminiGenConfig     `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature      float64 `json:"temperature,omitempty"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type GeminiCallOptions struct {
	Temperature     float64
	MaxOutputTokens int
	Timeout         time.Duration
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// CallGeminiJSON invokes the official Google Gemini generateContent REST API and unmarshals the JSON response into dst.
func CallGeminiJSON(ctx context.Context, systemPrompt string, userPrompt string, dst any) error {
	return CallGeminiJSONWithOptions(ctx, systemPrompt, userPrompt, dst, GeminiCallOptions{
		Temperature: 0.25,
		Timeout:     15 * time.Second,
	})
}

// CallGeminiJSONWithOptions invokes Gemini with configurable timeout, temperature, and output token ceiling.
func CallGeminiJSONWithOptions(ctx context.Context, systemPrompt string, userPrompt string, dst any, opts GeminiCallOptions) error {
	cfg := GetAIConfig(ctx)
	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY not configured")
	}

	temp := opts.Temperature
	if temp <= 0 {
		temp = 0.25
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	modelsToTry := []string{cfg.Model}
	if cfg.Model != "gemini-2.5-flash" {
		modelsToTry = append(modelsToTry, "gemini-2.5-flash")
	}
	if cfg.Model != "gemini-2.0-flash" {
		modelsToTry = append(modelsToTry, "gemini-2.0-flash")
	}

	var lastErr error
	for _, model := range modelsToTry {
		if model == "" || strings.Contains(model, " ") {
			model = "gemini-2.5-flash"
		}

		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, cfg.GeminiAPIKey)
		reqBody := geminiRequest{
			SystemInstruction: &geminiContent{
				Parts: []geminiPart{{Text: systemPrompt}},
			},
			Contents: []geminiContent{
				{
					Role:  "user",
					Parts: []geminiPart{{Text: userPrompt}},
				},
			},
			GenerationConfig: &geminiGenConfig{
				Temperature:      temp,
				MaxOutputTokens:  opts.MaxOutputTokens,
				ResponseMimeType: "application/json",
			},
		}

		payloadBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", cfg.GeminiAPIKey)
		if strings.HasPrefix(cfg.GeminiAPIKey, "AQ.") || strings.HasPrefix(cfg.GeminiAPIKey, "ya29.") {
			req.Header.Set("Authorization", "Bearer "+cfg.GeminiAPIKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		rawResp, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var gResp geminiResponse
		if err := json.Unmarshal(rawResp, &gResp); err != nil {
			lastErr = fmt.Errorf("decode gemini response: %w", err)
			continue
		}
		if gResp.Error != nil {
			lastErr = fmt.Errorf("gemini api error (%d): %s", gResp.Error.Code, gResp.Error.Message)
			// Only retry another model if the model name itself was not found (404)
			if gResp.Error.Code == 404 {
				continue
			}
			return lastErr
		}
		if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("gemini returned empty candidates")
			continue
		}

		text := strings.TrimSpace(gResp.Candidates[0].Content.Parts[0].Text)
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)

		if err := json.Unmarshal([]byte(text), dst); err != nil {
			lastErr = fmt.Errorf("unmarshal gemini json: %w", err)
			continue
		}
		return nil
	}

	return lastErr
}

type RawSignalInput struct {
	Index     int    `json:"index"`
	Headline  string `json:"headline"`
	Summary   string `json:"summary"`
	Platform  string `json:"platform"`
	Publisher string `json:"publisher"`
}

type ExtractedCandidateAI struct {
	Index           int      `json:"index"`
	IsRealPerson    bool     `json:"isRealPerson"`
	PersonName      string   `json:"personName"`
	Aliases         []string `json:"aliases"`
	Category        string   `json:"category"`
	Country         string   `json:"country"`
	EventTitle      string   `json:"eventTitle"`
	Achievement     string   `json:"achievement"`
	IsUnderdogStory bool     `json:"isUnderdogStory"`
}

// ExtractCandidatesWithGemini uses Gemini to extract genuine human subjects and achievements from raw headlines.
func ExtractCandidatesWithGemini(ctx context.Context, items []RawSignalInput) (map[int]ExtractedCandidateAI, error) {
	if !HasGeminiKey(ctx) || len(items) == 0 {
		return nil, fmt.Errorf("gemini unavailable or empty items")
	}

	itemsJSON, _ := json.Marshal(items)
	sysPrompt := `You are TrendScout Studio's Lead Documentary Discovery Intelligence AI.
Given a list of live news, Reddit, and Google Trends headlines, identify ONLY real individual human beings (athletes, prodigies, innovators, competitors, creators, or everyday heroes) who achieved a notable breakthrough, record, comeback, or underdog milestone.
CRITICAL RULES:
- Never extract team names, country names, cities, companies, or generic phrases as a personName.
- If an item does not mention a specific real individual person by first and last name, set "isRealPerson": false.
- Return a JSON object with a "candidates" array matching the schema.`

	userPrompt := fmt.Sprintf(`Analyze these live items and return JSON matching:
{
  "candidates": [
    {
      "index": 0,
      "isRealPerson": true,
      "personName": "Full Canonical Name",
      "aliases": ["Short Name"],
      "category": "Sports & Competition | Mind Sports | Science & Innovation | Arts & Culture",
      "country": "Country Name",
      "eventTitle": "Specific Event or Championship Name",
      "achievement": "1-sentence factual description of what the person achieved",
      "isUnderdogStory": true
    }
  ]
}

Items:
%s`, string(itemsJSON))

	var out struct {
		Candidates []ExtractedCandidateAI `json:"candidates"`
	}
	if err := CallGeminiJSON(ctx, sysPrompt, userPrompt, &out); err != nil {
		return nil, err
	}

	result := make(map[int]ExtractedCandidateAI)
	for _, c := range out.Candidates {
		if c.IsRealPerson && strings.TrimSpace(c.PersonName) != "" {
			result[c.Index] = c
		}
	}
	return result, nil
}

type UnderdogEvaluationAI struct {
	Classification          string            `json:"classification"`
	OriginsSummary          string            `json:"originsSummary"`
	EarlyAmbitions          string            `json:"earlyAmbitions"`
	Disadvantages           string            `json:"disadvantages"`
	RejectionsAndFailures   string            `json:"rejectionsAndFailures"`
	Obstacles               string            `json:"obstacles"`
	TurningPoints           string            `json:"turningPoints"`
	CareerProgression       string            `json:"careerProgression"`
	RecentBreakthrough      string            `json:"recentBreakthrough"`
	ContradictionsReport    string            `json:"contradictionsReport"`
	ResearchGapsReport      string            `json:"researchGapsReport"`
	AudienceInterestReport  string            `json:"audienceInterestReport"`
	OverallScore            float64           `json:"overallScore"`
	UnderdogStrength        float64           `json:"underdogStrength"`
	TransformationScore     float64           `json:"transformationScore"`
	AchievementSignificance float64           `json:"achievementSignificance"`
	AudienceMomentum        float64           `json:"audienceMomentum"`
	DocumentaryPotential    float64           `json:"documentaryPotential"`
	TimelinessScore         float64           `json:"timelinessScore"`
	EvidenceConfidence      float64           `json:"evidenceConfidence"`
	SaturationRisk          float64           `json:"saturationRisk"`
	ScoreRationale          map[string]string `json:"scoreRationale"`
}

func EvaluateUnderdogWithGemini(
	ctx context.Context,
	personName string,
	headline string,
	achievement string,
	wikiExtract string,
	youtubeViews int64,
	youtubeVPH float64,
	redditUpvotes int64,
	redditComments int64,
) (*UnderdogEvaluationAI, error) {
	if !HasGeminiKey(ctx) {
		return nil, fmt.Errorf("gemini api key not set")
	}

	sysPrompt := `You are TrendScout Studio's Underdog Assessment & Documentary Scoring Agent.
Evaluate the provided real-world biographical evidence, news headline, and live platform metrics (YouTube views/velocity, Reddit upvotes/comments, Wikipedia biography) for a documentary subject.
CRITICAL RULES:
- Base every field strictly on documented facts from the provided biography and headline. Do not invent fake facts.
- Score each of the 8 dimensions realistically between 40.0 and 99.0 based on the strength of the underdog narrative, real momentum, and biographical depth.
- Classification must be one of: "Strong underdog", "Emerging underdog", "Comeback story", "Breakthrough story", "Unexpected champion", "Extraordinary achievement".`

	userPrompt := fmt.Sprintf(`Subject: %s
Latest Headline: %s
Reported Achievement: %s
Live Metrics: YouTube Views=%d (%.0f views/hr), Reddit Upvotes=%d (%d comments)
Wikipedia Biographical Extract:
%s

Return a JSON object with this exact structure:
{
  "classification": "Strong underdog | Comeback story | Breakthrough story | Unexpected champion | Emerging underdog",
  "originsSummary": "Factual summary of origins and early background",
  "earlyAmbitions": "Early career ambitions and formative path",
  "disadvantages": "Specific structural, financial, physical, or competitive disadvantages faced",
  "rejectionsAndFailures": "Documented setbacks, losses, injuries, or doubts overcome",
  "obstacles": "Key obstacles on the path to the top",
  "turningPoints": "Decisive career turning points",
  "careerProgression": "Chronological progression summary",
  "recentBreakthrough": "Specific recent milestone or victory",
  "contradictionsReport": "Fact-check consistency note across sources",
  "researchGapsReport": "Specific archival interviews or footage to acquire next",
  "audienceInterestReport": "Analysis of current YouTube, Reddit, and search momentum",
  "overallScore": 88.5,
  "underdogStrength": 90.0,
  "transformationScore": 87.0,
  "achievementSignificance": 91.0,
  "audienceMomentum": 86.0,
  "documentaryPotential": 89.0,
  "timelinessScore": 88.0,
  "evidenceConfidence": 92.0,
  "saturationRisk": 22.0,
  "scoreRationale": {
    "underdog_strength": "1-sentence specific reason",
    "transformation_score": "1-sentence specific reason",
    "achievement_significance": "1-sentence specific reason",
    "audience_momentum": "1-sentence specific reason",
    "documentary_potential": "1-sentence specific reason",
    "timeliness_score": "1-sentence specific reason",
    "evidence_confidence": "1-sentence specific reason",
    "saturation_risk": "1-sentence specific reason"
  }
}`, personName, headline, achievement, youtubeViews, youtubeVPH, redditUpvotes, redditComments, wikiExtract)

	var eval UnderdogEvaluationAI
	if err := CallGeminiJSON(ctx, sysPrompt, userPrompt, &eval); err != nil {
		return nil, err
	}
	return &eval, nil
}

type DocumentaryShotAI struct {
	Visual          string  `json:"visual"`
	Narration       string  `json:"narration"`
	AudioCue        string  `json:"audioCue,omitempty"`
	DurationSeconds float64 `json:"durationSeconds"`
	LocationRef     string  `json:"locationRef,omitempty"`
}

type DocumentaryChapterOutlineAI struct {
	ChapterNumber   int      `json:"chapterNumber"`
	Title           string   `json:"title"`
	StartSeconds    int      `json:"startSeconds"`
	EndSeconds      int      `json:"endSeconds"`
	NarrativeArc    string   `json:"narrativeArc"`
	SpatialMotif    string   `json:"spatialMotif"`
	KeyFacts        []string `json:"keyFacts"`
	TargetShotCount int      `json:"targetShotCount"`
}

type DocumentaryOutlineAI struct {
	DetectedField   string                        `json:"detectedField"`
	CentralQuestion string                        `json:"centralQuestion"`
	OpeningHook     string                        `json:"openingHook"`
	RecurringMotif  string                        `json:"recurringMotif"`
	LedgerPositive  []string                      `json:"ledgerPositive"`
	LedgerCritical  []string                      `json:"ledgerCritical"`
	Chapters        []DocumentaryChapterOutlineAI `json:"chapters"`
}

type DocumentaryChapterAI struct {
	ChapterNumber int                 `json:"chapterNumber"`
	Title         string              `json:"title"`
	StartSeconds  int                 `json:"startSeconds"`
	EndSeconds    int                 `json:"endSeconds"`
	Shots         []DocumentaryShotAI `json:"shots"`
}

type DocumentaryScriptAI struct {
	DetectedField   string                 `json:"detectedField"`
	CentralQuestion string                 `json:"centralQuestion"`
	OpeningHook     string                 `json:"openingHook"`
	RecurringMotif  string                 `json:"recurringMotif"`
	Chapters        []DocumentaryChapterAI `json:"chapters"`
}

type ScriptActAI struct {
	ActNumber  int      `json:"actNumber"`
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}

// ComputeChapterTimeWindows returns the 7 prestige documentary chapter time windows scaled to totalSeconds.
// Proportions match the 20:00 benchmark: 0:00-1:30 (7.5%), 1:30-3:30 (10%), 3:30-6:30 (15%),
// 6:30-9:30 (15%), 9:30-12:30 (15%), 12:30-18:00 (27.5%), 18:00-20:00 (10%).
func ComputeChapterTimeWindows(totalSeconds int) [][2]int {
	if totalSeconds < 600 {
		totalSeconds = 1200
	}
	weights := []float64{0.075, 0.10, 0.15, 0.15, 0.15, 0.275, 0.10}
	out := make([][2]int, len(weights))
	cur := 0
	for i, w := range weights {
		end := cur + int(float64(totalSeconds)*w+0.5)
		if i == len(weights)-1 {
			end = totalSeconds
		}
		out[i] = [2]int{cur, end}
		cur = end
	}
	return out
}

func formatClockMMSS(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// GenerateDocumentaryScriptWithGemini executes a 2-stage prestige investigative documentary architecture:
// Stage 1: Dynamic biographical domain analysis + 7-chapter blueprint scaled to targetDurationMinutes.
// Stage 2: Concurrent chapter-by-chapter shot writer enforcing <= 7 seconds per visual cut.
func GenerateDocumentaryScriptWithGemini(
	ctx context.Context,
	personName string,
	category string,
	country string,
	projectTitle string,
	targetDurationMinutes int,
	style string,
	biographicalField string,
	biography string,
	claimsText string,
	assessmentSummary string,
) (*DocumentaryScriptAI, error) {
	if !HasGeminiKey(ctx) {
		return nil, fmt.Errorf("gemini api key not set")
	}

	if targetDurationMinutes < 15 || targetDurationMinutes > 25 {
		targetDurationMinutes = 20
	}
	totalSeconds := targetDurationMinutes * 60
	windows := ComputeChapterTimeWindows(totalSeconds)

	var windowSpec strings.Builder
	for i, w := range windows {
		dur := w[1] - w[0]
		minShots := (dur + 6) / 7
		if minShots < 10 {
			minShots = 10
		}
		windowSpec.WriteString(fmt.Sprintf("- Chapter %d: %s – %s (%d seconds, minimum %d shots at <=7s/cut)\n",
			i, formatClockMMSS(w[0]), formatClockMMSS(w[1]), dur, minShots))
	}

	fieldDirective := "Auto-detect the subject's primary biographical domain/field directly from the biography (e.g., Technology & Founders, Science & Medicine, Politics & Law, Sports & Competition, Arts & Culture, Business & Finance, Humanitarian & Civic Leadership, etc.) and tailor every room, institutional artifact, motif, and visual grammar specifically to that domain."
	if strings.TrimSpace(biographicalField) != "" && !strings.EqualFold(biographicalField, "auto") {
		fieldDirective = fmt.Sprintf("Treat this biography explicitly within the domain/field of '%s', tailoring every chapter title, institutional artifact, spatial motif, and visual shot to that field.", biographicalField)
	}

	// STAGE 1: Master Documentary Architecture & 7-Chapter Outline
	outlineSys := `You are the Lead Investigative Documentary Showrunner & Screenwriter for a prestige biographical film series (in the measured, austere, rigorously attributed style of Frontline / The New Yorker Films / prestige archival cinema).
Design a 7-chapter biographical documentary blueprint (Chapters 0 through 6) dynamically adapted to the subject's actual biographical domain.
EDITORIAL RULES:
1. ZERO melodrama, ZERO hype adjectives, ZERO clickbait clichés. Use restrained, forensic, declarative documentary journalism.
2. STRICT ATTRIBUTION: Distinguish verified public records, court/institutional filings, reported accounts, allegations, and subject denials. Never invent dates, verdicts, quotes, or facts not present in the dossier.
3. ARCHITECTURAL ARC:
   - Chapter 0 · Cold Open: Begin in media res inside a decisive public room/moment of reckoning or turning point, pull back to humble origins, pose one central investigative thesis question, and pivot ("We begin small. Very small.").
   - Chapter 1 · Early Origins & Craft: Birth, family/community context, formative obsession/skill, early education or apprenticeship, and the first small room where their capability emerged.
   - Chapter 2 · The First Breakthrough: The early creation, debut, invention, campaign, or breakthrough—how it worked, how rapidly it spread, and the first friction or controversy.
   - Chapter 3 · Contested Ground / The Record: Disputes, rivals, institutional pushback, critics, competing versions of events, and how public myth differs from the documented record ("This film uses only what can be attributed.").
   - Chapter 4 · Scale & Expansion: Rapid ascent, institutional growth, global reach, major milestones, honors or philanthropic/civic initiatives—and the structural tensions of scale.
   - Chapter 5 · Reckoning & Accountability: The longest, deepest chronological investigative chapter examining trials, controversies, setbacks, regulatory/institutional scrutiny, shifting alliances, and measurable consequences.
   - Chapter 6 · The Open Question: Return visually to the origin space and opening room. Lay out both sides of the factual ledger side-by-side. Close on an unresolved, resonant question ("The record is not closed.").
4. ` + fieldDirective + `
5. Style focus: ` + style

	outlineUser := fmt.Sprintf(`Subject: %s
Known Category: %s
Country / Origin: %s
Documentary Title: %s
Target Runtime: %d minutes (%d seconds)

Exact Chapter Time Windows:
%s

Subject Assessment & Context:
%s

Verified Biography Dossier:
%s

Verified Claims & Sources:
%s

Return strictly valid JSON matching:
{
  "detectedField": "Specific Biographical Domain (e.g., Technology & Platform Governance, Molecular Biology & Medicine, Olympic Athletics, etc.)",
  "centralQuestion": "One sharp, unresolved documentary thesis question",
  "openingHook": "2-3 austere cold-open narration sentences setting the defining public room",
  "recurringMotif": "The recurring spatial/room motif tracking scale from origin room to public arena",
  "ledgerPositive": ["Concrete documented achievement/contribution 1", "Concrete documented achievement 2"],
  "ledgerCritical": ["Concrete documented controversy/cost/unresolved tension 1", "Concrete documented challenge 2"],
  "chapters": [
    {
      "chapterNumber": 0,
      "title": "Cold Open",
      "startSeconds": 0,
      "endSeconds": 90,
      "narrativeArc": "Specific factual progression for this chapter",
      "spatialMotif": "The physical room/setting defining this chapter",
      "keyFacts": ["Specific verified fact/date/name 1", "Specific verified fact 2"],
      "targetShotCount": 14
    }
  ]
}`, personName, category, country, projectTitle, targetDurationMinutes, totalSeconds, windowSpec.String(), assessmentSummary, biography, claimsText)

	var outline DocumentaryOutlineAI
	if err := CallGeminiJSONWithOptions(ctx, outlineSys, outlineUser, &outline, GeminiCallOptions{
		Temperature:     0.3,
		MaxOutputTokens: 4096,
		Timeout:         30 * time.Second,
	}); err != nil {
		return nil, fmt.Errorf("stage 1 outline failed: %w", err)
	}
	if len(outline.Chapters) == 0 {
		return nil, fmt.Errorf("stage 1 returned zero chapters")
	}

	// Normalize chapter windows so all 7 chapters strictly match the target duration budget
	chaptersOut := make([]DocumentaryChapterAI, len(outline.Chapters))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	chapterSys := `You are the Lead Investigative Documentary Screenwriter & Visual Pacing Editor.
You write realigned shot-by-shot documentary scripts where EVERY visual cut is STRICTLY <= 7 seconds (4 to 7 seconds per shot) so the viewer sees a fresh, purposeful image at least every 7 seconds of speech or silence.

MANDATORY STYLE, COMPLEXITY & PACING RULES:
1. VOICEOVER STYLE: Sparse, declarative, measured, deeply researched journalism. Short, rhythmic sentences (often 8–16 words). Cite specific dates, places, institutions, numbers, and named people from the provided dossier. Explicitly attribute contested claims ("According to reporting...", "Filings allege...", "The subject denied..."). Never editorialize with hype or clichés.
2. <= 7 SECOND VISUAL CUT DISCIPLINE:
   - Every single shot MUST have durationSeconds between 4 and 7 (NEVER > 7).
   - A narrator speaks ~2.4 words per second (~15–17 words max in 7 seconds).
   - Whenever a sentence or thought runs longer than ~15 words, SPLIT it cleanly across two shots using ellipsis bridges ("...toward the audience..." -> "...and apologized to the families...") with a fresh cutaway shot for the continuation.
   - Interleave tactile cutaway shots (Macro details, Close-ups of hands/documents/tools/doors/screens, High-angle or Low-angle architectural angles) between primary establishing shots so no image ever holds longer than 7 seconds.
   - Include 1 to 3 deliberate breathing/atmosphere shots per chapter where narration is "" and audioCue is e.g. "Silence" or "Ambient room sound" (4–7 seconds each) at emotional transitions or chapter endings.
3. CONCRETE, FILMABLE VISUALS: Every visual description MUST begin with a concrete camera scale ("Wide shot of...", "Medium shot of...", "Close-up of...", "Macro shot of...", "Over-the-shoulder shot of...", "Low-angle shot of...", "High-angle wide shot of...", "Aerial shot of...") describing real physical spaces, lighting, archival documents, instruments, or environments authentic to the subject's biographical field. NEVER write vague abstractions like "visual metaphor of success".
4. VOLUME: Generate enough shots to completely fill the chapter's exact duration in seconds (sum of durationSeconds ~= endSeconds - startSeconds).`

	for i, ch := range outline.Chapters {
		chIdx := i
		chPlan := ch
		if chIdx < len(windows) {
			chPlan.StartSeconds = windows[chIdx][0]
			chPlan.EndSeconds = windows[chIdx][1]
		}
		chDur := chPlan.EndSeconds - chPlan.StartSeconds
		if chDur <= 0 {
			chDur = 120
		}
		targetShots := (chDur + 6) / 7
		if targetShots < 10 {
			targetShots = 10
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			chCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()

			factsJSON, _ := json.Marshal(chPlan.KeyFacts)
			chUser := fmt.Sprintf(`Subject: %s
Biographical Field: %s
Central Documentary Question: %s
Recurring Spatial Motif: %s
Style: %s

WRITE CHAPTER %d: "%s"
Time Window: %s – %s (Total Duration: %d seconds)
REQUIRED SHOT COUNT: Exactly %d to %d shots (every shot MUST be 4 to 7 seconds; NEVER exceed 7 seconds!)
Chapter Narrative Arc: %s
Chapter Spatial Motif: %s
Allocated Verified Facts for This Chapter: %s

Full Subject Biography & Claims Reference (use ONLY real details grounded here):
%s
%s

Return strictly valid JSON matching:
{
  "chapterNumber": %d,
  "title": "%s",
  "startSeconds": %d,
  "endSeconds": %d,
  "shots": [
    {
      "visual": "Wide shot of ...",
      "narration": "10 to 16 words of crisp, source-grounded documentary narration.",
      "audioCue": "",
      "durationSeconds": 7,
      "locationRef": "Specific physical setting"
    },
    {
      "visual": "Macro shot of ...",
      "narration": "...continuation clause under 16 words.",
      "audioCue": "",
      "durationSeconds": 7,
      "locationRef": "Specific physical setting"
    }
  ]
}`,
				personName, outline.DetectedField, outline.CentralQuestion, outline.RecurringMotif, style,
				chPlan.ChapterNumber, chPlan.Title, formatClockMMSS(chPlan.StartSeconds), formatClockMMSS(chPlan.EndSeconds), chDur,
				targetShots, targetShots+4,
				chPlan.NarrativeArc, chPlan.SpatialMotif, string(factsJSON),
				biography, claimsText,
				chPlan.ChapterNumber, chPlan.Title, chPlan.StartSeconds, chPlan.EndSeconds)

			var built DocumentaryChapterAI
			if err := CallGeminiJSONWithOptions(chCtx, chapterSys, chUser, &built, GeminiCallOptions{
				Temperature:     0.35,
				MaxOutputTokens: 8192,
				Timeout:         42 * time.Second,
			}); err == nil && len(built.Shots) > 0 {
				built.ChapterNumber = chPlan.ChapterNumber
				if strings.TrimSpace(built.Title) == "" {
					built.Title = chPlan.Title
				}
				built.StartSeconds = chPlan.StartSeconds
				built.EndSeconds = chPlan.EndSeconds
				chaptersOut[chIdx] = built
			}
		}()
	}
	wg.Wait()

	populated := 0
	for _, c := range chaptersOut {
		if len(c.Shots) > 0 {
			populated++
		}
	}
	if populated == 0 {
		return nil, fmt.Errorf("stage 2 chapter generation returned zero shots")
	}

	return &DocumentaryScriptAI{
		DetectedField:   outline.DetectedField,
		CentralQuestion: outline.CentralQuestion,
		OpeningHook:     outline.OpeningHook,
		RecurringMotif:  outline.RecurringMotif,
		Chapters:        chaptersOut,
	}, nil
}

// RegenerateShotWithGemini refines a single documentary shot block while keeping <= 7s pacing and journalistic style.
func RegenerateShotWithGemini(
	ctx context.Context,
	personName string,
	biographicalField string,
	existingBlock string,
	biography string,
	tone string,
) (string, string, error) {
	if !HasGeminiKey(ctx) {
		return "", "", fmt.Errorf("gemini api key not set")
	}
	sysPrompt := `You are a prestige investigative documentary script editor.
Refine a single <= 7-second documentary shot block (visual direction + narrator voiceover line).
RULES:
- Narration MUST be <= 16 words (strictly <= 7 seconds spoken at 145 WPM).
- Restrained, forensic, journalistic tone — zero hype or melodrama.
- Visual MUST start with a concrete camera scale (Wide shot, Medium shot, Close-up, Macro shot, Low-angle shot, Over-the-shoulder shot) grounded in the subject's biographical domain.
- Ground every factual reference strictly in the provided biography.`

	userPrompt := fmt.Sprintf(`Subject: %s
Biographical Field: %s
Requested Tone / Emphasis: %s
Existing Shot Block:
%s

Biography Reference:
%s

Return JSON:
{
  "visual": "Concrete camera shot description matching the biographical field",
  "narration": "Refined narration line strictly <= 16 words"
}`, personName, biographicalField, tone, existingBlock, biography)

	var out struct {
		Visual    string `json:"visual"`
		Narration string `json:"narration"`
	}
	if err := CallGeminiJSONWithOptions(ctx, sysPrompt, userPrompt, &out, GeminiCallOptions{
		Temperature:     0.35,
		MaxOutputTokens: 512,
		Timeout:         15 * time.Second,
	}); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(out.Visual), strings.TrimSpace(out.Narration), nil
}

// GenerateScriptWithGemini is preserved for legacy callers, wrapping the 2-stage prestige script writer.
func GenerateScriptWithGemini(
	ctx context.Context,
	personName string,
	projectTitle string,
	targetDurationMinutes int,
	style string,
	biography string,
	claimsText string,
) (string, []ScriptActAI, error) {
	doc, err := GenerateDocumentaryScriptWithGemini(ctx, personName, "", "", projectTitle, targetDurationMinutes, style, "auto", biography, claimsText, "")
	if err != nil {
		return "", nil, err
	}
	var acts []ScriptActAI
	for i, ch := range doc.Chapters {
		var paras []string
		for _, sh := range ch.Shots {
			if strings.TrimSpace(sh.Narration) != "" {
				paras = append(paras, sh.Narration)
			}
		}
		acts = append(acts, ScriptActAI{
			ActNumber:  i + 1,
			Title:      ch.Title,
			Paragraphs: paras,
		})
	}
	return doc.OpeningHook, acts, nil
}

