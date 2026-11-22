package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
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

// GetAIConfig resolves AI provider, model, API key & YouTube key from user_settings (PostgreSQL) or environment variables (.env).
// Supports Groq (gsk_...), OpenRouter (sk-or-...), OpenAI (sk-...), Google AI Studio (AIza...), and Vertex AI Express (AQ...).
func GetAIConfig(ctx context.Context) AIConfig {
	provider := strings.TrimSpace(os.Getenv("AI_PROVIDER"))
	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	model := strings.TrimSpace(os.Getenv("AI_MODEL"))
	if apiKey != "" {
		if provider == "" {
			provider = "Groq"
		}
		if model == "" {
			model = "llama-3.3-70b-versatile"
		}
	} else if orKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); orKey != "" {
		apiKey = orKey
		if provider == "" {
			provider = "OpenRouter"
		}
		if model == "" {
			model = "meta-llama/llama-3.3-70b-instruct:free"
		}
	} else {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("GEMINI_API"))
		}
		if provider == "" {
			provider = "Google Gemini"
		}
		if model == "" {
			model = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
		}
		if model == "" {
			model = "gemini-2.5-flash"
		}
	}

	cfg := AIConfig{
		Provider:      provider,
		Model:         model,
		GeminiAPIKey:  apiKey,
		YouTubeAPIKey: strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY")),
	}

	if db.DB != nil {
		var aiJSON string
		if err := db.DB.QueryRowContext(ctx, `SELECT ai_prefs_json FROM user_settings LIMIT 1`).Scan(&aiJSON); err == nil && aiJSON != "" {
			var m map[string]any
			if json.Unmarshal([]byte(aiJSON), &m) == nil {
				if v, ok := m["provider"].(string); ok && strings.TrimSpace(v) != "" {
					cfg.Provider = strings.TrimSpace(v)
				}
				if v, ok := m["geminiApiKey"].(string); ok && strings.TrimSpace(v) != "" {
					cfg.GeminiAPIKey = strings.TrimSpace(v)
				}
				if v, ok := m["groqApiKey"].(string); ok && strings.TrimSpace(v) != "" && cfg.GeminiAPIKey == "" {
					cfg.GeminiAPIKey = strings.TrimSpace(v)
				}
				if v, ok := m["youtubeApiKey"].(string); ok && strings.TrimSpace(v) != "" {
					cfg.YouTubeAPIKey = strings.TrimSpace(v)
				}
				if v, ok := m["model"].(string); ok && strings.TrimSpace(v) != "" {
					modelVal := strings.TrimSpace(v)
					if !strings.Contains(modelVal, " ") {
						cfg.Model = modelVal
					}
				}
			}
		}
	}
	return cfg
}

func detectProviderKind(cfg AIConfig) (kind string, label string, baseURL string, defaultModels []string) {
	key := strings.TrimSpace(cfg.GeminiAPIKey)
	provLow := strings.ToLower(strings.TrimSpace(cfg.Provider))

	switch {
	case strings.HasPrefix(key, "gsk_") || strings.Contains(provLow, "groq"):
		return "groq", "Groq", "https://api.groq.com/openai/v1/chat/completions", []string{
			"llama-3.3-70b-versatile",
			"openai/gpt-oss-120b",
			"openai/gpt-oss-20b",
			"llama-3.1-8b-instant",
		}
	case strings.HasPrefix(key, "sk-or-") || strings.Contains(provLow, "openrouter"):
		return "openrouter", "OpenRouter", "https://openrouter.ai/api/v1/chat/completions", []string{
			"meta-llama/llama-3.3-70b-instruct:free",
			"google/gemma-3-27b-it:free",
		}
	case strings.HasPrefix(key, "AQ."):
		return "vertex", "Vertex AI Express", "", []string{
			"gemini-2.5-flash",
			"gemini-2.0-flash",
		}
	case strings.HasPrefix(key, "AIza") || strings.HasPrefix(key, "ya29."):
		return "gemini", "Google AI Studio", "", []string{
			"gemini-2.5-flash",
			"gemini-2.0-flash",
		}
	case strings.HasPrefix(key, "sk-") || strings.Contains(provLow, "openai"):
		return "openai", "OpenAI", "https://api.openai.com/v1/chat/completions", []string{
			"gpt-4o-mini",
			"gpt-4o",
		}
	default:
		return "gemini", "Google AI Studio", "", []string{
			"gemini-2.5-flash",
			"gemini-2.0-flash",
		}
	}
}

func isModelCompatibleWithProvider(kind, model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" || strings.Contains(m, " ") {
		return false
	}
	switch kind {
	case "groq":
		return !strings.HasPrefix(m, "gemini-") && !strings.HasPrefix(m, "gpt-") && !strings.HasSuffix(m, ":free")
	case "openrouter":
		return strings.Contains(m, "/")
	case "openai":
		return strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4")
	default:
		return strings.HasPrefix(m, "gemini-") && !strings.Contains(m, "/")
	}
}

var thinkBlockRe = regexp.MustCompile(`(?s)<think>.*?</think>`)

func cleanAIJSONResponse(raw string) string {
	text := strings.TrimSpace(raw)
	text = thinkBlockRe.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```JSON")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	// Extract outermost JSON object if wrapped in stray prose
	if !strings.HasPrefix(text, "{") {
		if idx := strings.Index(text, "{"); idx >= 0 {
			text = text[idx:]
		}
	}
	if !strings.HasSuffix(text, "}") {
		if idx := strings.LastIndex(text, "}"); idx >= 0 {
			text = text[:idx+1]
		}
	}
	return strings.TrimSpace(text)
}

func ActiveProviderLabel(ctx context.Context) string {
	_, label, _, _ := detectProviderKind(GetAIConfig(ctx))
	return label
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

type GeminiTestResult struct {
	Connected       bool     `json:"connected"`
	ModelUsed       string   `json:"modelUsed"`
	EndpointType    string   `json:"endpointType"`
	LatencyMs       int64    `json:"latencyMs"`
	AvailableModels []string `json:"availableModels,omitempty"`
	Error           string   `json:"error,omitempty"`
}

var (
	groqModelsMu      sync.Mutex
	groqModelsCache   []string
	groqModelsFetched time.Time
)

func isGroqChatModel(id string) bool {
	low := strings.ToLower(strings.TrimSpace(id))
	if low == "" {
		return false
	}
	exclude := []string{"whisper", "tts", "orpheus", "playai", "guard", "safeguard", "embed", "rerank", "moderation", "vision-preview", "distil-whisper"}
	for _, ex := range exclude {
		if strings.Contains(low, ex) {
			return false
		}
	}
	return true
}

func rankGroqModel(id string) int {
	low := strings.ToLower(id)
	switch {
	case low == "llama-3.3-70b-versatile":
		return 100
	case low == "openai/gpt-oss-120b":
		return 95
	case strings.Contains(low, "qwen") && !strings.Contains(low, "guard"):
		return 90
	case strings.Contains(low, "70b") || strings.Contains(low, "120b"):
		return 85
	case low == "openai/gpt-oss-20b":
		return 80
	case strings.Contains(low, "llama-4"):
		return 78
	case low == "llama-3.1-8b-instant":
		return 70
	case strings.Contains(low, "compound"):
		return 60
	default:
		return 50
	}
}

// fetchLiveGroqModels queries GET https://api.groq.com/openai/v1/models to discover the exact active
// chat models enabled on the user's Groq account (cached for 15 minutes).
func fetchLiveGroqModels(ctx context.Context, apiKey string, forceRefresh bool) []string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}
	groqModelsMu.Lock()
	if !forceRefresh && len(groqModelsCache) > 0 && time.Since(groqModelsFetched) < 15*time.Minute {
		out := append([]string(nil), groqModelsCache...)
		groqModelsMu.Unlock()
		return out
	}
	groqModelsMu.Unlock()

	mCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(mCtx, http.MethodGet, "https://api.groq.com/openai/v1/models", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var payload struct {
		Data []struct {
			ID     string `json:"id"`
			Active *bool  `json:"active"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	var models []string
	for _, it := range payload.Data {
		if it.Active != nil && !*it.Active {
			continue
		}
		if isGroqChatModel(it.ID) {
			models = append(models, it.ID)
		}
	}
	// Sort highest-capability chat models first
	for i := 0; i < len(models); i++ {
		best := i
		for j := i + 1; j < len(models); j++ {
			if rankGroqModel(models[j]) > rankGroqModel(models[best]) {
				best = j
			}
		}
		models[i], models[best] = models[best], models[i]
	}

	if len(models) > 0 {
		groqModelsMu.Lock()
		groqModelsCache = append([]string(nil), models...)
		groqModelsFetched = time.Now()
		groqModelsMu.Unlock()
	}
	return models
}

// TestGeminiConnection runs a real live JSON probe against the configured AI provider
// (Groq, OpenRouter, OpenAI, Google AI Studio, or Vertex AI Express) so producers can immediately verify live AI.
func TestGeminiConnection(ctx context.Context, overrideKey string, overrideModel string) GeminiTestResult {
	return TestAIConnection(ctx, "", overrideKey, overrideModel)
}

func TestAIConnection(ctx context.Context, overrideProvider string, overrideKey string, overrideModel string) GeminiTestResult {
	cfg := GetAIConfig(ctx)
	if strings.TrimSpace(overrideProvider) != "" {
		cfg.Provider = strings.TrimSpace(overrideProvider)
	}
	if strings.TrimSpace(overrideKey) != "" {
		cfg.GeminiAPIKey = strings.TrimSpace(overrideKey)
	}
	if strings.TrimSpace(overrideModel) != "" {
		cfg.Model = strings.TrimSpace(overrideModel)
	}

	kind, epType, _, _ := detectProviderKind(cfg)

	if cfg.GeminiAPIKey == "" {
		return GeminiTestResult{
			Connected:    false,
			EndpointType: epType,
			Error:        "No AI API key configured. Paste a Groq (gsk_...), OpenRouter (sk-or-...), or Google Gemini key.",
		}
	}

	var liveModels []string
	if kind == "groq" {
		liveModels = fetchLiveGroqModels(ctx, cfg.GeminiAPIKey, true)
	}

	start := time.Now()
	var probe struct {
		Status string `json:"status"`
	}
	usedModel, err := callGeminiWithConfig(ctx, cfg,
		`You are a connection health probe. Return strictly valid JSON: {"status":"ok"}`,
		`Respond with JSON: {"status":"ok"}`,
		&probe,
		GeminiCallOptions{Temperature: 0.1, MaxOutputTokens: 64, Timeout: 14 * time.Second},
	)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return GeminiTestResult{
			Connected:       false,
			ModelUsed:       cfg.Model,
			EndpointType:    epType,
			LatencyMs:       latency,
			AvailableModels: liveModels,
			Error:           err.Error(),
		}
	}
	return GeminiTestResult{
		Connected:       true,
		ModelUsed:       usedModel,
		EndpointType:    epType,
		LatencyMs:       latency,
		AvailableModels: liveModels,
	}
}

// CallGeminiJSONWithOptions invokes the active AI provider (Groq, OpenRouter, OpenAI, or Gemini) with JSON mode.
func CallGeminiJSONWithOptions(ctx context.Context, systemPrompt string, userPrompt string, dst any, opts GeminiCallOptions) error {
	cfg := GetAIConfig(ctx)
	_, err := callGeminiWithConfig(ctx, cfg, systemPrompt, userPrompt, dst, opts)
	return err
}

func callOpenAICompatibleJSON(ctx context.Context, cfg AIConfig, kind string, label string, baseURL string, modelsToTry []string, systemPrompt string, userPrompt string, dst any, opts GeminiCallOptions) (string, error) {
	temp := opts.Temperature
	if temp <= 0 {
		temp = 0.25
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	maxTokens := opts.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	if kind == "groq" && maxTokens > 8000 {
		maxTokens = 8000
	}

	sysText := systemPrompt
	if !strings.Contains(strings.ToLower(sysText), "json") {
		sysText += "\nReturn strictly valid JSON only."
	}

	type oaMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type oaRespFmt struct {
		Type string `json:"type"`
	}
	type oaReq struct {
		Model          string     `json:"model"`
		Messages       []oaMsg    `json:"messages"`
		Temperature    float64    `json:"temperature,omitempty"`
		MaxTokens      int        `json:"max_tokens,omitempty"`
		ResponseFormat *oaRespFmt `json:"response_format,omitempty"`
	}
	type oaResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    any    `json:"code"`
		} `json:"error,omitempty"`
	}

	var lastErr error
	var triedErrs []string
	for _, model := range modelsToTry {
		useJSONFormat := kind == "groq" || kind == "openai"

		// Up to 2 automatic retries on 429 rate-limit bursts or json_validate_failed
		for attempt := 0; attempt < 3; attempt++ {
			reqObj := oaReq{
				Model: model,
				Messages: []oaMsg{
					{Role: "system", Content: sysText},
					{Role: "user", Content: userPrompt},
				},
				Temperature: temp,
				MaxTokens:   maxTokens,
			}
			if useJSONFormat {
				reqObj.ResponseFormat = &oaRespFmt{Type: "json_object"}
			}
			payloadBytes, err := json.Marshal(reqObj)
			if err != nil {
				return "", err
			}

			client := &http.Client{Timeout: timeout}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(payloadBytes))
			if err != nil {
				return "", err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+cfg.GeminiAPIKey)
			if kind == "openrouter" {
				req.Header.Set("HTTP-Referer", "https://trendscout-iota.vercel.app")
				req.Header.Set("X-Title", "TrendScout Studio")
			}

			resp, err := client.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("%s request failed: %w", label, err)
				break
			}
			rawResp, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = fmt.Errorf("%s read failed: %w", label, err)
				break
			}

			if resp.StatusCode == 429 {
				waitSec := 3 * (attempt + 1)
				if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
					if parsedSec, errP := strconv.Atoi(ra); errP == nil && parsedSec > 0 && parsedSec <= 10 {
						waitSec = parsedSec
					}
				}
				var er oaResp
				_ = json.Unmarshal(rawResp, &er)
				msg := "rate limit exceeded"
				if er.Error != nil && er.Error.Message != "" {
					msg = er.Error.Message
				}
				lastErr = fmt.Errorf("%s api error (429): %s", strings.ToLower(label), msg)
				if attempt < 2 {
					select {
					case <-ctx.Done():
						return "", lastErr
					case <-time.After(time.Duration(waitSec) * time.Second):
						continue
					}
				}
				break // try next model bucket if available
			}

			var parsed oaResp
			if err := json.Unmarshal(rawResp, &parsed); err != nil {
				lastErr = fmt.Errorf("decode %s response (%d): %w", label, resp.StatusCode, err)
				break
			}
			if parsed.Error != nil {
				errMsgLow := strings.ToLower(parsed.Error.Message)
				lastErr = fmt.Errorf("%s api error (%d): %s", strings.ToLower(label), resp.StatusCode, parsed.Error.Message)
				if resp.StatusCode == 400 && useJSONFormat && (strings.Contains(errMsgLow, "json") || strings.Contains(errMsgLow, "response_format")) {
					// Some open-weight models reject strict response_format; retry cleanly relying on prompt JSON + cleanAIJSONResponse
					useJSONFormat = false
					continue
				}
				if resp.StatusCode == 404 || (resp.StatusCode == 400 && strings.Contains(errMsgLow, "model")) {
					triedErrs = append(triedErrs, fmt.Sprintf("%s: %s", model, parsed.Error.Message))
					break // try next live model
				}
				if resp.StatusCode == 401 || resp.StatusCode == 403 {
					return "", lastErr
				}
				break
			}
			if resp.StatusCode >= 400 {
				lastErr = fmt.Errorf("%s api error (%d): %s", strings.ToLower(label), resp.StatusCode, strings.TrimSpace(string(rawResp)))
				if resp.StatusCode == 401 || resp.StatusCode == 403 {
					return "", lastErr
				}
				break
			}
			if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
				lastErr = fmt.Errorf("%s returned empty response", label)
				break
			}

			cleaned := cleanAIJSONResponse(parsed.Choices[0].Message.Content)
			if err := json.Unmarshal([]byte(cleaned), dst); err != nil {
				lastErr = fmt.Errorf("unmarshal %s json: %w", label, err)
				break
			}
			return model, nil
		}
	}

	if len(triedErrs) > 0 && lastErr != nil {
		return "", fmt.Errorf("%s (tried: %s)", lastErr.Error(), strings.Join(triedErrs, " | "))
	}
	return "", lastErr
}

func callGeminiWithConfig(ctx context.Context, cfg AIConfig, systemPrompt string, userPrompt string, dst any, opts GeminiCallOptions) (string, error) {
	if cfg.GeminiAPIKey == "" {
		return "", fmt.Errorf("AI API key not configured")
	}

	kind, label, baseURL, defaultModels := detectProviderKind(cfg)
	var modelsToTry []string
	if isModelCompatibleWithProvider(kind, cfg.Model) {
		modelsToTry = append(modelsToTry, strings.TrimSpace(cfg.Model))
	}
	appendUnique := func(list []string) {
		for _, m := range list {
			dup := false
			for _, existing := range modelsToTry {
				if existing == m {
					dup = true
					break
				}
			}
			if !dup {
				modelsToTry = append(modelsToTry, m)
			}
		}
	}
	if kind == "groq" {
		if live := fetchLiveGroqModels(ctx, cfg.GeminiAPIKey, false); len(live) > 0 {
			// Cap to top 5 ranked live chat models from the user's actual Groq account
			if len(live) > 5 {
				live = live[:5]
			}
			appendUnique(live)
		}
	}
	appendUnique(defaultModels)

	if kind == "groq" || kind == "openrouter" || kind == "openai" {
		return callOpenAICompatibleJSON(ctx, cfg, kind, label, baseURL, modelsToTry, systemPrompt, userPrompt, dst, opts)
	}

	temp := opts.Temperature
	if temp <= 0 {
		temp = 0.25
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	isVertexExpressKey := kind == "vertex"
	isOAuthToken := strings.HasPrefix(cfg.GeminiAPIKey, "ya29.")

	var lastErr error
	for _, model := range modelsToTry {
		if model == "" || strings.Contains(model, " ") {
			model = "gemini-2.5-flash"
		}

		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, cfg.GeminiAPIKey)
		if isVertexExpressKey {
			endpoint = fmt.Sprintf("https://aiplatform.googleapis.com/v1/publishers/google/models/%s:generateContent?key=%s", model, cfg.GeminiAPIKey)
		} else if isOAuthToken {
			endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
		}
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
			return "", err
		}

		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if isOAuthToken {
			req.Header.Set("Authorization", "Bearer "+cfg.GeminiAPIKey)
		} else if !isVertexExpressKey {
			req.Header.Set("x-goog-api-key", cfg.GeminiAPIKey)
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
			if gResp.Error.Code == 404 || (gResp.Error.Code == 400 && strings.Contains(strings.ToLower(gResp.Error.Message), "model")) {
				continue
			}
			return "", lastErr
		}
		if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("gemini returned empty candidates")
			continue
		}

		text := cleanAIJSONResponse(gResp.Candidates[0].Content.Parts[0].Text)
		if err := json.Unmarshal([]byte(text), dst); err != nil {
			lastErr = fmt.Errorf("unmarshal gemini json: %w", err)
			continue
		}
		return model, nil
	}

	return "", lastErr
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
	ChapterNumber     int      `json:"chapterNumber"`
	Title             string   `json:"title"`
	StartSeconds      int      `json:"startSeconds"`
	EndSeconds        int      `json:"endSeconds"`
	NarrativeArc      string   `json:"narrativeArc"`
	SpatialMotif      string   `json:"spatialMotif"`
	StrategyExecution string   `json:"strategyExecution"`
	KeyFacts          []string `json:"keyFacts"`
	TargetShotCount   int      `json:"targetShotCount"`
}

type DocumentaryOutlineAI struct {
	DetectedField    string                        `json:"detectedField"`
	OpeningArchetype string                        `json:"openingArchetype"`
	UnderdogAngle    string                        `json:"underdogAngle"`
	CentralQuestion  string                        `json:"centralQuestion"`
	OpeningHook      string                        `json:"openingHook"`
	RecurringMotif   string                        `json:"recurringMotif"`
	LedgerPositive   []string                      `json:"ledgerPositive"`
	LedgerCritical   []string                      `json:"ledgerCritical"`
	Chapters         []DocumentaryChapterOutlineAI `json:"chapters"`
}

type DocumentaryChapterAI struct {
	ChapterNumber int                 `json:"chapterNumber"`
	Title         string              `json:"title"`
	StartSeconds  int                 `json:"startSeconds"`
	EndSeconds    int                 `json:"endSeconds"`
	Shots         []DocumentaryShotAI `json:"shots"`
}

type DocumentaryScriptAI struct {
	DetectedField    string                 `json:"detectedField"`
	OpeningArchetype string                 `json:"openingArchetype"`
	UnderdogAngle    string                 `json:"underdogAngle"`
	ModelUsed        string                 `json:"modelUsed"`
	CentralQuestion  string                 `json:"centralQuestion"`
	OpeningHook      string                 `json:"openingHook"`
	RecurringMotif   string                 `json:"recurringMotif"`
	Chapters         []DocumentaryChapterAI `json:"chapters"`
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

type CompetitivePositioningBriefAI struct {
	WinningHooks             []string `json:"winningHooks"`
	WhatsWorking             []string `json:"whatsWorking"`
	AudienceQuestions        []string `json:"audienceQuestions"`
	CompetitorBlindSpots     []string `json:"competitorBlindSpots"`
	OutperformanceStrategy   string   `json:"outperformanceStrategy"`
	RecommendedColdOpenEdge  string   `json:"recommendedColdOpenEdge"`
	UnderdogOpeningArchetype string   `json:"underdogOpeningArchetype"`
	FormatAdvantage          string   `json:"formatAdvantage"`
	ScriptExecutionPlan      []string `json:"scriptExecutionPlan"`
}

// AnalyzeCompetitorsForPositioningWithGemini executes Stage 0 pre-script competitive analysis:
// studies transcripts, outlier multipliers, formats, and top viewer comments from competing YouTube videos
// to determine what is working and how to position our documentary to outperform them.
func AnalyzeCompetitorsForPositioningWithGemini(
	ctx context.Context,
	personName string,
	biographicalField string,
	subjectContext string,
	competitorDataJSON string,
	targetDurationMinutes int,
) (*CompetitivePositioningBriefAI, error) {
	if !HasGeminiKey(ctx) {
		return nil, fmt.Errorf("gemini api key not set")
	}

	sysPrompt := `You are TrendScout Studio's Lead YouTube Competitive Strategist & Documentary Showrunner.
Before writing a documentary script, you audit the highest-performing competing YouTube videos (including their real transcripts, view velocity, channel-relative outlier multipliers, video durations/formats, and top viewer comments).
YOUR GOAL:
1. Identify EXACTLY what hooks, narrative framing, emotional tension, or curiosity gaps made the highest-outlier videos outperform their channel average—citing the actual competitor titles, channel names, and transcript opening lines.
2. Mine top viewer comments for specific unanswered questions, recurring debates, skeptical counter-arguments, and emotional resonance points (ignore generic fan praise like "GOAT" or emojis).
3. Expose concrete competitor blind spots specific to THIS subject (e.g., exact biographical turning points, disputes, or setbacks that the studied Shorts/clips skipped).
4. Formulate a concrete, subject-specific OUTPERFORMANCE POSITIONING STRATEGY and a chapter-by-chapter SCRIPT EXECUTION PLAN for a prestige, source-grounded ` + fmt.Sprintf("%d-minute", targetDurationMinutes) + ` documentary paced at <=7 seconds per visual cut.
CRITICAL: NEVER write generic boilerplate that could apply to any other person. Every field MUST name specific facts about ` + personName + ` and specific competing videos/comments from the JSON payload. NEVER recommend plagiarizing competitor lines.`

	userPrompt := fmt.Sprintf(`Subject: %s
Biographical Field: %s
Target Documentary Runtime: %d minutes

Subject Biographical Context:
%s

Competing YouTube Videos (Metrics, Transcripts & Top Viewer Comments):
%s

Return strictly valid JSON matching:
{
  "winningHooks": ["Specific hook pattern citing an exact studied video title/channel or transcript opening"],
  "whatsWorking": ["Specific narrative/structural/emotional element driving outlier views on THIS subject"],
  "audienceQuestions": ["Specific unanswered question or debate raised by viewers in comments"],
  "competitorBlindSpots": ["Specific biographical fact, setback, or arc gap missing from the studied videos"],
  "outperformanceStrategy": "2-3 concrete sentences naming the top competing video/channel and detailing how our documentary on %s will outperform them in retention, authority, and depth.",
  "recommendedColdOpenEdge": "1-2 sentences prescribing the exact 0:00-1:30 subversive Underdog Cold Open scene for %s that deliberately contrasts what competitor transcripts opened with.",
  "underdogOpeningArchetype": "Choose ONE best-fit archetype: The Hidden Low Point | The Viral Myth vs. The Record | The Unlikely Room | The Unanswered Question | The Quiet Turning Point",
  "formatAdvantage": "1-2 sentences citing the actual durations/formats of the studied videos vs our %d-minute <=7s/cut film.",
  "scriptExecutionPlan": [
    "Ch 0 (Cold Open): Exact opening contrast & hook execution",
    "Ch 2-3 (Breakthrough & Contested Ground): Exact competitor blind spot & viewer debate resolved",
    "Ch 5-6 (Reckoning & Ledger): Exact unanswered audience question closed on the record"
  ]
}`, personName, biographicalField, targetDurationMinutes, subjectContext, competitorDataJSON, personName, personName, targetDurationMinutes)

	var out CompetitivePositioningBriefAI
	if err := CallGeminiJSONWithOptions(ctx, sysPrompt, userPrompt, &out, GeminiCallOptions{
		Temperature:     0.35,
		MaxOutputTokens: 2048,
		Timeout:         25 * time.Second,
	}); err != nil {
		return nil, err
	}
	return &out, nil
}

// GenerateDocumentaryScriptWithGemini executes a 2-stage prestige investigative documentary architecture
// (optionally informed by Stage 0 YouTube Competitive Positioning Intelligence):
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
	competitiveBriefText string,
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

	// STAGE 1: Master Underdog Documentary Architecture & 7-Chapter Outline (Aligned to Stage 0 Transcripts & Comments)
	outlineSys := `You are the Lead Underdog Documentary Showrunner & Investigative Screenwriter for a prestige biographical YouTube documentary channel (measured, cinematic, source-grounded journalism—never generic or templated).
Design a 7-chapter UNDERDOG biographical documentary blueprint (Chapters 0 through 6) shaped directly by BOTH the subject's verified underdog dossier AND the Stage 0 YouTube Competitor Transcript & Comment Study.

EDITORIAL RULES:
1. ZERO COOKIE-CUTTER OPENINGS: NEVER reuse generic boilerplate openings like "Inside a silent arena tunnel, every camera turned toward..." or forced taglines like "We begin small. Very small." Every documentary MUST open with a fresh, subject-specific scene chosen specifically to SUBVERT what competing YouTube transcripts already overplayed.
2. ZERO melodrama, ZERO clickbait hype. Write with tense, austere, cinematic journalistic precision.
3. STRICT ATTRIBUTION: Base every date, name, setback, quote, and turning point on the verified dossier. Distinguish verified records, reported accounts, allegations, and denials. Never invent fake facts.
4. UNDERDOG 7-CHAPTER STORYTELLING ARC (ALIGNED TO COMPETITOR BLIND SPOTS):
   - Chapter 0 · Subversive Cold Open: Execute the Stage 0 COLD OPEN EDGE & UNDERDOG OPENING ARCHETYPE (e.g., open on the documented lowest point/injury/rejection, OR dismantle a viral clip myth from competitor transcripts, OR open on the exact unanswered viewer debate from comments). Plant one unforgettable central underdog question in the first 60 seconds.
   - Chapter 1 · The Unlikely Starting Line: Concrete geographic, financial, physical, or institutional disadvantages competitors skip—grounding the human world before anyone knew their name.
   - Chapter 2 · The Wall: Documented rejections, losses, injuries, skeptics, or systemic barriers that 60-second highlight reels erase.
   - Chapter 3 · The Crucible: The specific craft, obsession, sacrifice, or contested turning point—directly addressing the top viewer debates and competitor blind spots from Stage 0.
   - Chapter 4 · Against the Odds: The defining breakthrough or upset—re-framed so it lands with 10x emotional weight because the audience now understands what was truly at stake.
   - Chapter 5 · The Cost of Winning: The longest, deepest investigative chapter examining post-breakthrough pressure, rivals, controversies, institutional scrutiny, and unresolved accountability that short clips ignore.
   - Chapter 6 · The Underdog Verdict: Return visually to the film's opening motif, directly answer the viewer debate planted in Chapter 0, weigh both sides of the verified ledger, and close on a resonant final thought.
5. COMPETITIVE OUTPERFORMANCE EXECUTION (MANDATORY):
   - Chapter 0 MUST directly execute the Stage 0 COLD OPEN EDGE without copying competitor lines.
   - Chapters 2, 3, 4, and 5 MUST explicitly assign and answer the Stage 0 VIEWER QUESTIONS / DEBATES and fill the COMPETITOR BLIND SPOTS inside each chapter's "strategyExecution" and "narrativeArc".
6. ` + fieldDirective + `
7. Style focus: ` + style

	outlineUser := fmt.Sprintf(`Subject: %s
Known Category: %s
Country / Origin: %s
Documentary Title: %s
Target Runtime: %d minutes (%d seconds)

Exact Chapter Time Windows:
%s

Subject Assessment & Context:
%s

Pre-Script YouTube Competitive Positioning Brief (Outperform these competing videos without copying them — subvert overplayed openings, resolve viewer debates & exploit blind spots):
%s

Verified Biography Dossier:
%s

Verified Claims & Sources:
%s

Return strictly valid JSON matching:
{
  "detectedField": "Specific Biographical Domain (e.g., Olympic Track & Field, World Championship Chess, Technology & Platform Governance, etc.)",
  "openingArchetype": "The Hidden Low Point | The Viral Myth vs. The Record | The Unlikely Room | The Unanswered Question | The Quiet Turning Point",
  "underdogAngle": "1-2 crisp sentences explaining the unique editorial viewpoint separating this film from existing YouTube videos on this subject",
  "centralQuestion": "One sharp, unresolved underdog question informed by viewer comments and the subject's real arc",
  "openingHook": "2-3 original, subject-specific cold-open narration sentences (NEVER generic arena/room boilerplate)",
  "recurringMotif": "The concrete physical motif authentic to THIS specific person's craft and journey",
  "ledgerPositive": ["Concrete documented achievement/contribution 1", "Concrete documented achievement 2"],
  "ledgerCritical": ["Concrete documented controversy/cost/unresolved tension 1", "Concrete documented challenge 2"],
  "chapters": [
    {
      "chapterNumber": 0,
      "title": "Specific compelling chapter title (never generic boilerplate)",
      "startSeconds": 0,
      "endSeconds": 90,
      "narrativeArc": "Specific factual progression for this chapter",
      "spatialMotif": "The authentic physical setting for THIS person in this chapter",
      "strategyExecution": "Exact Stage 0 competitive edge, viewer question, or competitor blind spot this chapter visibly executes",
      "keyFacts": ["Specific verified fact/date/name 1", "Specific verified fact 2"],
      "targetShotCount": 14
    }
  ]
}`, personName, category, country, projectTitle, targetDurationMinutes, totalSeconds, windowSpec.String(), assessmentSummary, competitiveBriefText, biography, claimsText)

	var outline DocumentaryOutlineAI
	cfg := GetAIConfig(ctx)
	usedModel, err := callGeminiWithConfig(ctx, cfg, outlineSys, outlineUser, &outline, GeminiCallOptions{
		Temperature:     0.4,
		MaxOutputTokens: 4096,
		Timeout:         30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("stage 1 outline failed: %w", err)
	}
	if len(outline.Chapters) == 0 {
		return nil, fmt.Errorf("stage 1 returned zero chapters")
	}

	// Normalize chapter windows so all 7 chapters strictly match the target duration budget
	chaptersOut := make([]DocumentaryChapterAI, len(outline.Chapters))
	chapterErrs := make([]error, len(outline.Chapters))
	var wg sync.WaitGroup
	// Bound concurrency for Groq/OpenRouter free-tier TPM limits while keeping full 7-chapter generation fast
	maxConc := 4
	kind, _, _, _ := detectProviderKind(cfg)
	if kind == "groq" || kind == "openrouter" {
		maxConc = 2
	}
	sem := make(chan struct{}, maxConc)
	bioForChapter := biography
	if len(bioForChapter) > 2800 {
		bioForChapter = bioForChapter[:2800]
	}
	claimsForChapter := claimsText
	if len(claimsForChapter) > 2200 {
		claimsForChapter = claimsForChapter[:2200]
	}

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
4. VOLUME: Generate enough shots to completely fill the chapter's exact duration in seconds (sum of durationSeconds ~= endSeconds - startSeconds).
5. VISIBLE STAGE 0 EXECUTION: Every chapter MUST visibly deliver its assigned "Chapter Competitive Execution Target" in the spoken NARRATOR (V.O.) lines—executing the Cold Open edge in Chapter 0, answering mined audience questions, and exposing the exact verified details competing YouTube clips skip—always in restrained, source-attributed documentary prose.`

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
CHAPTER COMPETITIVE EXECUTION TARGET (MUST BE VISIBLY DELIVERED IN NARRATION): %s
Allocated Verified Facts for This Chapter: %s
Overall YouTube Competitive Positioning Brief: %s

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
				chPlan.NarrativeArc, chPlan.SpatialMotif, chPlan.StrategyExecution, string(factsJSON), competitiveBriefText,
				bioForChapter, claimsForChapter,
				chPlan.ChapterNumber, chPlan.Title, chPlan.StartSeconds, chPlan.EndSeconds)

			var built DocumentaryChapterAI
			errCh := CallGeminiJSONWithOptions(chCtx, chapterSys, chUser, &built, GeminiCallOptions{
				Temperature:     0.35,
				MaxOutputTokens: 8192,
				Timeout:         42 * time.Second,
			})
			if errCh != nil {
				chapterErrs[chIdx] = fmt.Errorf("chapter %d (%s): %w", chPlan.ChapterNumber, chPlan.Title, errCh)
				return
			}
			if len(built.Shots) == 0 {
				chapterErrs[chIdx] = fmt.Errorf("chapter %d (%s) returned zero shots", chPlan.ChapterNumber, chPlan.Title)
				return
			}
			built.ChapterNumber = chPlan.ChapterNumber
			if strings.TrimSpace(built.Title) == "" {
				built.Title = chPlan.Title
			}
			built.StartSeconds = chPlan.StartSeconds
			built.EndSeconds = chPlan.EndSeconds
			chaptersOut[chIdx] = built
		}()
	}
	wg.Wait()

	for i, c := range chaptersOut {
		if chapterErrs[i] != nil {
			return nil, fmt.Errorf("stage 2 chapter generation failed: %w", chapterErrs[i])
		}
		if len(c.Shots) == 0 {
			return nil, fmt.Errorf("stage 2 chapter %d returned zero shots", i)
		}
	}

	return &DocumentaryScriptAI{
		DetectedField:    outline.DetectedField,
		OpeningArchetype: outline.OpeningArchetype,
		UnderdogAngle:    outline.UnderdogAngle,
		ModelUsed:        usedModel,
		CentralQuestion:  outline.CentralQuestion,
		OpeningHook:      outline.OpeningHook,
		RecurringMotif:   outline.RecurringMotif,
		Chapters:         chaptersOut,
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

type VideoPackagingAI struct {
	RecommendedTitles []string `json:"recommendedTitles"`
	VideoDescription  string   `json:"videoDescription"`
	Tags              []string `json:"tags"`
	Hashtags          []string `json:"hashtags"`
}

// GenerateVideoPackagingWithGemini creates a high-CTR YouTube documentary description (with chapter timestamps),
// high-performing SEO tags (kept cleanly within YouTube's 500-char tag budget), and hashtags for an opportunity/project,
// optionally guided by Stage 0 YouTube Competitive Positioning Intelligence.
func GenerateVideoPackagingWithGemini(
	ctx context.Context,
	personName string,
	biographicalField string,
	country string,
	headline string,
	achievement string,
	biography string,
	chapterTimestamps string,
	competitiveBrief string,
	targetDurationMinutes int,
) (*VideoPackagingAI, error) {
	if !HasGeminiKey(ctx) {
		return nil, fmt.Errorf("gemini api key not set")
	}

	sysPrompt := `You are TrendScout Studio's YouTube Documentary Growth & SEO Strategist.
Write high-performing, search-optimized YouTube packaging for a prestige investigative biographical documentary.
RULES:
1. COMPETITIVE OUTPERFORMANCE: If a YouTube Competitive Positioning Brief is provided, engineer the titles, above-the-fold description hook, tags, and hashtags to outperform the competing videos' hooks, capture the exact viewer questions from comments, and exploit competitor blind spots/format gaps—WITHOUT copying competitor titles or phrasing.
2. VIDEO DESCRIPTION:
   - Open with a sharp 2-sentence above-the-fold hook optimized for YouTube search & browse previews (include subject's full name and core turning point naturally in the first 150 characters).
   - Follow with a concise 1-paragraph factual editorial overview grounded strictly in the provided dossier.
   - Include a clean "Chapters:" timestamp block using the exact provided chapter timestamps (if provided).
   - Include a brief journalistic attribution note ("Every milestone in this film is grounded in public records and attributed reporting.") and a subtle subscribe/discussion prompt.
   - End with 3–5 primary hashtags on the final line.
3. HIGH-PERFORMING TAGS (16–22 tags):
   - Mix exact-match subject name, biographical field/niche keywords, high-intent documentary search phrases ("<name> documentary", "<name> true story", "<name> biography"), event/institution keywords, and broader discovery terms.
   - Total comma-separated length of all tags MUST stay <= 480 characters (YouTube's strict 500-char tag box limit).
   - No leading # symbols on tags, no duplicate tags.
4. HASHTAGS (8–12 hashtags):
   - Every hashtag MUST begin with # and contain no spaces or punctuation (CamelCase e.g. #Documentary, #TrueStory, #Biography, plus subject & domain-specific tags).
5. RECOMMENDED TITLES (3 titles):
   - High-CTR prestige documentary titles (55–72 characters, zero clickbait spam).`

	if strings.TrimSpace(competitiveBrief) == "" {
		competitiveBrief = "No competing YouTube brief cached yet; optimize for definitive prestige biographical search intent."
	}

	userPrompt := fmt.Sprintf(`Subject: %s
Biographical Field: %s
Country / Origin: %s
Target Duration: %d minutes
Latest Headline: %s
Core Breakthrough / Turning Point: %s
Chapter Timestamps:
%s

YouTube Competitive Positioning Brief (Outperform these competitors in CTR & search positioning):
%s

Biographical Dossier Extract:
%s

Return strictly valid JSON matching:
{
  "recommendedTitles": ["Title 1", "Title 2", "Title 3"],
  "videoDescription": "Full formatted YouTube video description with Chapters and closing hashtags",
  "tags": ["exact subject name", "subject documentary", "long tail seo tag"],
  "hashtags": ["#SubjectName", "#Documentary", "#Biography"]
}`, personName, biographicalField, country, targetDurationMinutes, headline, achievement, chapterTimestamps, competitiveBrief, biography)

	var out VideoPackagingAI
	if err := CallGeminiJSONWithOptions(ctx, sysPrompt, userPrompt, &out, GeminiCallOptions{
		Temperature:     0.35,
		MaxOutputTokens: 2048,
		Timeout:         25 * time.Second,
	}); err != nil {
		return nil, err
	}
	return &out, nil
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
	doc, err := GenerateDocumentaryScriptWithGemini(ctx, personName, "", "", projectTitle, targetDurationMinutes, style, "auto", biography, claimsText, "", "")
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

