package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	"trendscout-backend/internal/db"
	"trendscout-backend/internal/services"
)

func EnsureInitialOperationalData(ctx context.Context) error {
	// Create default portrait fallback SVG
	defaultSvgPath := filepath.Join(db.StorageRoot, "characters", "default-portrait.svg")
	_ = services.GenerateStoryboardSVG(defaultSvgPath, "TrendScout Verified Subject", "Biographical & Documentary Subject Profile", "PORTRAIT", "VERIFIED", "Championship gold")

	// 1. Ensure default producer user & settings
	var userCount int
	_ = db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount)
	userID := "usr-producer-01"
	if userCount == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("trendscout2026"), bcrypt.DefaultCost)
		_, _ = db.DB.ExecContext(ctx, `
			INSERT INTO users (id, email, password_hash, name, role)
			VALUES ($1, 'producer@trendscout.studio', $2, 'Lead Documentary Producer', 'admin')
		`, userID, string(hash))

		discPrefs, _ := json.Marshal(map[string]any{
			"categories":            []string{"Sports & Competition", "Mind Sports & Chess", "Track & Field Athletics", "Science & Innovation"},
			"regions":               []string{"Global", "North America", "Europe", "Asia", "Africa", "Latin America"},
			"discoveryPeriodDays":   14,
			"minOpportunityScore":   70,
			"underdogPriority":      "High — Prioritize Unseeded & Breakthrough Subjects",
			"minEvidenceConfidence": 75,
			"saturationTolerance":   40,
		})
		monPrefs, _ := json.Marshal(map[string]any{
			"monitoringActive":    true,
			"frequencyMinutes":    15,
			"maxItemsPerScan":     25,
			"enableBackgroundJob": true,
		})
		gemKey := os.Getenv("GEMINI_API_KEY")
		if gemKey == "" {
			gemKey = os.Getenv("GEMINI_API")
		}
		gemModel := os.Getenv("GEMINI_MODEL")
		if gemModel == "" {
			gemModel = "gemini-3.8-flash"
		}
		aiPrefs, _ := json.Marshal(map[string]any{
			"provider":             "Google Gemini",
			"model":                gemModel,
			"geminiApiKey":         gemKey,
			"youtubeApiKey":        os.Getenv("YOUTUBE_API_KEY"),
			"researchDepth":        "Deep Live Wikipedia + Multi-Platform Verification",
			"defaultScriptMinutes": 20,
			"writingStyle":         "Cinematic Continuous Storytelling (YouTube Mini-Doc)",
			"language":             "en-US",
			"dailyTokenBudget":     250000,
		})
		notifPrefs, _ := json.Marshal(map[string]any{
			"emailAddress":        "producer@trendscout.studio",
			"earlyWhisperAlerts":  true,
			"emergingAlerts":      true,
			"acceleratingAlerts":  true,
			"majorViralAlerts":    true,
			"dailyDigestTime":     "08:00 UTC",
			"weeklyTop10Day":      "Monday",
			"chromeNotifications": true,
		})
		prodPrefs, _ := json.Marshal(map[string]any{
			"imageIntervalSeconds": 8,
			"visualStyle":          "Cinematic Archival Realism (35mm Documentary Stills)",
			"aspectRatio":          "16:9",
			"narrationProvider":    "macOS Neural Speech / OpenAI TTS",
			"subtitleFormat":       "srt",
			"autoAssembleMp4":      true,
		})

		_, _ = db.DB.ExecContext(ctx, `
			INSERT INTO user_settings (id, user_id, discovery_prefs_json, monitoring_prefs_json, ai_prefs_json, notification_prefs_json, production_prefs_json)
			VALUES ('uset-01', $1, $2, $3, $4, $5, $6)
		`, userID, string(discPrefs), string(monPrefs), string(aiPrefs), string(notifPrefs), string(prodPrefs))
	}

	// 2. Ensure connected sources exist with accurate statuses
	sources := []struct {
		id       string
		name     string
		platform string
		enabled  bool
		limit    int
		status   string
		errMsg   string
	}{
		{"src-wikipedia", "Wikipedia & Wikimedia Live Biographical Engine", "Wikipedia", true, 5000, "ready", ""},
		{"src-google-news", "Google News & Global Publishers Live RSS", "News", true, 2500, "ready", ""},
		{"src-youtube", "YouTube Live Video & Velocity Extractor", "YouTube", true, 2000, "ready", ""},
		{"src-reddit", "Reddit Multi-Subreddit Live Breakout Monitor", "Reddit", true, 2000, "ready", ""},
		{"src-google-trends", "Google Trends Daily Breakout RSS Monitor", "Google Trends", true, 1500, "ready", ""},
		{"src-x-twitter", "X / Twitter Real-Time Velocity Adapter", "X/Twitter", false, 1000, "requires_api_key", "Requires X API v2 Bearer Token in Settings/.env"},
		{"src-tiktok", "TikTok Creator Surge Monitor", "TikTok", false, 1000, "requires_api_key", "Requires TikTok Research API Key in Settings/.env"},
	}
	for _, s := range sources {
		_, _ = db.DB.ExecContext(ctx, `
			INSERT INTO connected_sources (id, name, platform, enabled, quota_used, quota_limit, last_scan_at, status, error_message)
			VALUES ($1, $2, $3, $4, 0, $5, NOW(), $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				enabled = EXCLUDED.enabled,
				status = EXCLUDED.status,
				error_message = EXCLUDED.error_message
		`, s.id, s.name, s.platform, s.enabled, s.limit, s.status, s.errMsg)
	}

	// 3. If no discoveries exist yet, run a 100% live discovery cycle (zero hardcoded seed profiles)
	var discCount int
	_ = db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM discoveries`).Scan(&discCount)
	if discCount == 0 {
		_, _ = RunDiscoveryCycle(ctx, nil)
	}

	return nil
}
