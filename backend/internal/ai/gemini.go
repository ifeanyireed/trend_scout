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
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
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
	cfg := GetAIConfig(ctx)
	if cfg.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY not configured")
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
				Temperature:      0.25,
				ResponseMimeType: "application/json",
			},
		}

		payloadBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: 15 * time.Second}
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

		return json.Unmarshal([]byte(text), dst)
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

type ScriptActAI struct {
	ActNumber  int      `json:"actNumber"`
	Title      string   `json:"title"`
	Paragraphs []string `json:"paragraphs"`
}

func GenerateScriptWithGemini(
	ctx context.Context,
	personName string,
	projectTitle string,
	targetDurationMinutes int,
	style string,
	biography string,
	claimsText string,
) (string, []ScriptActAI, error) {
	if !HasGeminiKey(ctx) {
		return "", nil, fmt.Errorf("gemini api key not set")
	}

	sysPrompt := `You are TrendScout Studio's Lead Documentary Screenwriter.
Write a deeply compelling, source-grounded 6-act continuous documentary narration script about the subject.
CRITICAL RULES:
- Use ONLY verified biographical facts from the provided biography and research claims.
- Structure into exactly 6 acts (Act I through Act VI), each with 3 rich, cinematic narration paragraphs.
- Style: ` + style

	userPrompt := fmt.Sprintf(`Subject: %s
Documentary Title: %s
Target Duration: %d minutes
Verified Biography & Claims:
%s

%s

Return JSON with this exact structure:
{
  "openingHook": "Powerful 2-3 sentence cold open hook",
  "acts": [
    {
      "actNumber": 1,
      "title": "Act I — Title",
      "paragraphs": ["Paragraph 1...", "Paragraph 2...", "Paragraph 3..."]
    }
  ]
}`, personName, projectTitle, targetDurationMinutes, biography, claimsText)

	var resp struct {
		OpeningHook string        `json:"openingHook"`
		Acts        []ScriptActAI `json:"acts"`
	}
	if err := CallGeminiJSON(ctx, sysPrompt, userPrompt, &resp); err != nil {
		return "", nil, err
	}
	if len(resp.Acts) == 0 {
		return "", nil, fmt.Errorf("empty script acts returned")
	}
	return resp.OpeningHook, resp.Acts, nil
}

