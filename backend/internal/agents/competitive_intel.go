package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"trendscout-backend/internal/ai"
	"trendscout-backend/internal/connectors"
	"trendscout-backend/internal/db"
)

type StudiedVideoSummary struct {
	VideoID             string  `json:"videoId"`
	Title               string  `json:"title"`
	ChannelName         string  `json:"channelName"`
	URL                 string  `json:"url"`
	Views               int64   `json:"views"`
	ViewsPerHour        float64 `json:"viewsPerHour"`
	ChannelMultiplier   float64 `json:"channelMultiplier"`
	EngagementRate      float64 `json:"engagementRate"`
	VideoFormat         string  `json:"videoFormat"`
	DurationSeconds     int     `json:"durationSeconds"`
	TranscriptAvailable bool    `json:"transcriptAvailable"`
	TranscriptExcerpt   string  `json:"transcriptExcerpt,omitempty"`
	CommentsCount       int     `json:"commentsCount"`
}

type CompetitiveSubjectContext struct {
	Headline      string
	Achievement   string
	Summary       string
	Origins       string
	TurningPoints string
	Obstacles     string
}

type CompetitivePositioningBrief struct {
	AnalyzedVideoCount       int                   `json:"analyzedVideoCount"`
	TranscribedVideoCount    int                   `json:"transcribedVideoCount"`
	CommentsAnalyzedCount    int                   `json:"commentsAnalyzedCount"`
	WinningHooks             []string              `json:"winningHooks"`
	WhatsWorking             []string              `json:"whatsWorking"`
	AudienceQuestions        []string              `json:"audienceQuestions"`
	CompetitorBlindSpots     []string              `json:"competitorBlindSpots"`
	OutperformanceStrategy   string                `json:"outperformanceStrategy"`
	RecommendedColdOpenEdge  string                `json:"recommendedColdOpenEdge"`
	UnderdogOpeningArchetype string                `json:"underdogOpeningArchetype,omitempty"`
	FormatAdvantage          string                `json:"formatAdvantage"`
	ScriptExecutionPlan      []string              `json:"scriptExecutionPlan,omitempty"`
	StudiedVideos            []StudiedVideoSummary `json:"studiedVideos"`
	UsedGemini               bool                  `json:"usedGemini"`
	GeneratedAt              string                `json:"generatedAt"`
}

var likesPrefixRe = regexp.MustCompile(`^\[\d+\s+likes\]\s*`)

func stripCommentLikesPrefix(c string) string {
	return strings.TrimSpace(likesPrefixRe.ReplaceAllString(strings.TrimSpace(c), ""))
}

func formatDurationShort(sec int) string {
	if sec <= 0 {
		return "short clip"
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func sanitizeAIErrorForUI(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "aiplatform.googleapis.com") && strings.Contains(low, "disabled"):
		return "Vertex AI API (aiplatform.googleapis.com) is disabled for your AQ.* key — enable Vertex AI API in Google Cloud Console or switch Settings to a Google AI Studio (AIza...) key."
	case strings.Contains(low, "api_key_service_blocked") || strings.Contains(low, "401") || strings.Contains(low, "invalid api key"):
		return "AI API key was rejected (401 Unauthorized) — open Settings and configure a valid Groq (gsk_...), OpenRouter (sk-or-...), or Gemini key."
	case strings.Contains(low, "429") || strings.Contains(low, "quota") || strings.Contains(low, "resource_exhausted") || strings.Contains(low, "rate limit"):
		return "AI rate limit exceeded (429) — wait a few seconds and retry."
	case strings.Contains(low, "not configured") || strings.Contains(low, "not set"):
		return "No AI API key configured — open Settings and add a Groq (gsk_...), OpenRouter (sk-or-...), or Gemini API key."
	default:
		if len(msg) > 220 {
			msg = msg[:217] + "..."
		}
		return msg
	}
}

// FormatCompetitiveBriefForPrompt renders a concise, actionable directive injected into Stage 1 & Stage 2 script prompts.
func FormatCompetitiveBriefForPrompt(brief *CompetitivePositioningBrief) string {
	if brief == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analyzed %d top YouTube videos (%d transcribed, %d viewer comments mined).\n",
		brief.AnalyzedVideoCount, brief.TranscribedVideoCount, brief.CommentsAnalyzedCount))
	if brief.OutperformanceStrategy != "" {
		sb.WriteString("OUTPERFORMANCE STRATEGY: " + brief.OutperformanceStrategy + "\n")
	}
	if brief.RecommendedColdOpenEdge != "" {
		sb.WriteString("COLD OPEN EDGE: " + brief.RecommendedColdOpenEdge + "\n")
	}
	if brief.UnderdogOpeningArchetype != "" {
		sb.WriteString("UNDERDOG OPENING ARCHETYPE: " + brief.UnderdogOpeningArchetype + "\n")
	}
	if brief.FormatAdvantage != "" {
		sb.WriteString("FORMAT ADVANTAGE: " + brief.FormatAdvantage + "\n")
	}
	if len(brief.WinningHooks) > 0 {
		sb.WriteString("WHAT'S WORKING IN TOP OUTLIERS: " + strings.Join(brief.WinningHooks, " | ") + "\n")
	}
	if len(brief.CompetitorBlindSpots) > 0 {
		sb.WriteString("COMPETITOR BLIND SPOTS TO EXPLOIT: " + strings.Join(brief.CompetitorBlindSpots, " | ") + "\n")
	}
	if len(brief.AudienceQuestions) > 0 {
		sb.WriteString("VIEWER QUESTIONS / DEBATES TO RESOLVE: " + strings.Join(brief.AudienceQuestions, " | ") + "\n")
	}
	if len(brief.ScriptExecutionPlan) > 0 {
		sb.WriteString("CHAPTER EXECUTION PLAN: " + strings.Join(brief.ScriptExecutionPlan, " | ") + "\n")
	}
	return strings.TrimSpace(sb.String())
}

func loadSubjectContextForBrief(ctx context.Context, personID, discoveryID string) CompetitiveSubjectContext {
	var sc CompetitiveSubjectContext
	if db.DB == nil {
		return sc
	}
	if personID != "" {
		_ = db.DB.QueryRowContext(ctx, `SELECT COALESCE(summary, '') FROM people WHERE id = $1`, personID).Scan(&sc.Summary)
		_ = db.DB.QueryRowContext(ctx, `
			SELECT COALESCE(origins_summary, ''), COALESCE(turning_points, ''), COALESCE(obstacles, ''), COALESCE(recent_breakthrough, '')
			FROM underdog_assessments WHERE person_id = $1 LIMIT 1
		`, personID).Scan(&sc.Origins, &sc.TurningPoints, &sc.Obstacles, &sc.Achievement)
	}
	if discoveryID != "" {
		var hl, ach string
		if db.DB.QueryRowContext(ctx, `
			SELECT COALESCE(headline, ''), COALESCE(achievement, '') FROM discoveries WHERE id = $1
		`, discoveryID).Scan(&hl, &ach) == nil {
			sc.Headline = hl
			if strings.TrimSpace(ach) != "" {
				sc.Achievement = ach
			}
		}
	}
	return sc
}

// BuildProjectCompetitiveBrief runs Stage 0 pre-script YouTube competitor study (transcribing top videos,
// mining top viewer comments, and synthesizing positioning via Gemini AI). No local fallback: returns an error if AI is unavailable or fails.
func BuildProjectCompetitiveBrief(
	ctx context.Context,
	projectID string,
	personID string,
	discoveryID string,
	personName string,
	domain BiographicalDomainProfile,
	targetDurationMinutes int,
) (*CompetitivePositioningBrief, string, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, "", fmt.Errorf("Gemini AI required for Stage 0 Competitor Study: no Gemini API key configured in Settings")
	}
	if targetDurationMinutes < 15 || targetDurationMinutes > 25 {
		targetDurationMinutes = 20
	}

	aiCfg := ai.GetAIConfig(ctx)
	subjectCtx := loadSubjectContextForBrief(ctx, personID, discoveryID)

	// 1. Load tracked top YouTube videos from DB (or fetch live if none tracked yet)
	var topVideos []connectors.VideoMetric
	if db.DB != nil {
		rows, err := db.DB.QueryContext(ctx, `
			SELECT external_video_id, title, channel_name, channel_subscribers, url,
			       views, likes, comments, views_per_hour, views_per_day, channel_relative_multiplier,
			       published_at, COALESCE(duration_seconds, 0), COALESCE(video_format, 'Standard'),
			       COALESCE(engagement_rate, 0), COALESCE(channel_baseline_views, 0), COALESCE(thumbnail_url, '')
			FROM videos
			WHERE discovery_id = $1 OR person_id = $2
			ORDER BY views_per_hour DESC, views DESC
			LIMIT 4
		`, discoveryID, personID)
		if err == nil && rows != nil {
			for rows.Next() {
				var v connectors.VideoMetric
				if rows.Scan(&v.ExternalID, &v.Title, &v.ChannelName, &v.ChannelSubscribers, &v.VideoURL,
					&v.Views, &v.Likes, &v.Comments, &v.ViewsPerHour, &v.ViewsPerDay, &v.ChannelMultiplier,
					&v.PublishedAt, &v.DurationSeconds, &v.VideoFormat, &v.EngagementRate,
					&v.ChannelBaselineViews, &v.ThumbnailURL) == nil {
					topVideos = append(topVideos, v)
				}
			}
			rows.Close()
		}
	}

	if len(topVideos) == 0 && personName != "" {
		ytCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
		fetched, errYT := connectors.LookupYouTubeIntelForPerson(ytCtx, personName, aiCfg.YouTubeAPIKey)
		cancel()
		if errYT == nil && len(fetched) > 0 {
			topVideos = fetched
			if discoveryID != "" && db.DB != nil {
				for _, v := range fetched {
					_, _ = db.DB.ExecContext(ctx, `
						INSERT INTO videos (
							id, person_id, discovery_id, external_video_id, platform, title, channel_name,
							channel_subscribers, url, views, likes, comments, views_per_hour, views_per_day,
							channel_relative_multiplier, published_at, duration_seconds, video_format,
							engagement_rate, channel_baseline_views, thumbnail_url
						) VALUES ($1,$2,$3,$4,'YouTube',$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
						ON CONFLICT (external_video_id) DO NOTHING
					`, "vid-"+strings.ReplaceAll(v.ExternalID, "yt-", ""), personID, discoveryID, v.ExternalID, v.Title,
						v.ChannelName, v.ChannelSubscribers, v.VideoURL, v.Views, v.Likes, v.Comments,
						v.ViewsPerHour, v.ViewsPerDay, v.ChannelMultiplier, v.PublishedAt,
						v.DurationSeconds, v.VideoFormat, v.EngagementRate, v.ChannelBaselineViews, v.ThumbnailURL)
				}
			}
		}
	}

	// 2. Concurrently transcribe top 3 competing videos & fetch top viewer comments
	studies := connectors.AnalyzeCompetitorVideosConcurrently(ctx, topVideos, aiCfg.YouTubeAPIKey, 3)

	// 3. Synthesize competitive brief strictly with Gemini AI (zero local fallback)
	studiesJSON := "[]"
	if len(studies) > 0 {
		if b, errM := json.Marshal(studies); errM == nil {
			studiesJSON = string(b)
		}
	}
	subjectSummary := fmt.Sprintf("Headline: %s\nBreakthrough: %s\nOrigins: %s\nTurning Points: %s\nObstacles: %s\nSummary: %s",
		subjectCtx.Headline, subjectCtx.Achievement, subjectCtx.Origins, subjectCtx.TurningPoints, subjectCtx.Obstacles, subjectCtx.Summary)

	aiCtx, cancelAI := context.WithTimeout(ctx, 25*time.Second)
	aiBrief, errAI := ai.AnalyzeCompetitorsForPositioningWithGemini(aiCtx, personName, domain.Label, subjectSummary, studiesJSON, targetDurationMinutes)
	cancelAI()
	if errAI != nil || aiBrief == nil {
		return nil, "", fmt.Errorf("Gemini AI Stage 0 Competitor Study failed: %s", sanitizeAIErrorForUI(errAI))
	}

	brief := CompetitivePositioningBrief{
		AnalyzedVideoCount:       len(studies),
		WinningHooks:             aiBrief.WinningHooks,
		WhatsWorking:             aiBrief.WhatsWorking,
		AudienceQuestions:        aiBrief.AudienceQuestions,
		CompetitorBlindSpots:     aiBrief.CompetitorBlindSpots,
		OutperformanceStrategy:   strings.TrimSpace(aiBrief.OutperformanceStrategy),
		RecommendedColdOpenEdge:  strings.TrimSpace(aiBrief.RecommendedColdOpenEdge),
		UnderdogOpeningArchetype: strings.TrimSpace(aiBrief.UnderdogOpeningArchetype),
		FormatAdvantage:          strings.TrimSpace(aiBrief.FormatAdvantage),
		ScriptExecutionPlan:      aiBrief.ScriptExecutionPlan,
		UsedGemini:               true,
		GeneratedAt:              time.Now().UTC().Format(time.RFC3339),
	}

	for _, st := range studies {
		if st.TranscriptAvailable {
			brief.TranscribedVideoCount++
		}
		brief.CommentsAnalyzedCount += len(st.TopComments)
		var previewExcerpt string
		if st.TranscriptAvailable && strings.TrimSpace(st.TranscriptExcerpt) != "" {
			previewExcerpt = truncateText(strings.Join(strings.Fields(st.TranscriptExcerpt), " "), 420)
		}
		brief.StudiedVideos = append(brief.StudiedVideos, StudiedVideoSummary{
			VideoID:             st.VideoID,
			Title:               st.Title,
			ChannelName:         st.ChannelName,
			URL:                 st.URL,
			Views:               st.Views,
			ViewsPerHour:        st.ViewsPerHour,
			ChannelMultiplier:   st.ChannelMultiplier,
			EngagementRate:      st.EngagementRate,
			VideoFormat:         st.VideoFormat,
			DurationSeconds:     st.DurationSeconds,
			TranscriptAvailable: st.TranscriptAvailable,
			TranscriptExcerpt:   previewExcerpt,
			CommentsCount:       len(st.TopComments),
		})
	}

	// 4. Persist onto documentary_projects for Studio inspection & export
	if projectID != "" && db.DB != nil {
		if b, errM := json.Marshal(brief); errM == nil {
			_, _ = db.DB.ExecContext(ctx, `
				UPDATE documentary_projects SET competitive_brief_json = $2, updated_at = NOW() WHERE id = $1
			`, projectID, string(b))
		}
	}

	return &brief, FormatCompetitiveBriefForPrompt(&brief), nil
}

// GenerateProjectCompetitiveBrief triggers Stage 0 on demand for an existing documentary project.
func GenerateProjectCompetitiveBrief(ctx context.Context, projectID string) (*CompetitivePositioningBrief, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, fmt.Errorf("Gemini AI required for Stage 0 Competitor Study: no Gemini API key configured in Settings")
	}
	var personID, discoveryID string
	var targetDur int
	err := db.DB.QueryRowContext(ctx, `
		SELECT person_id, COALESCE(discovery_id, ''), target_duration_minutes
		FROM documentary_projects WHERE id = $1
	`, projectID).Scan(&personID, &discoveryID, &targetDur)
	if err != nil {
		return nil, err
	}
	if targetDur < 15 || targetDur > 25 {
		targetDur = 20
	}
	var personName, category, summary, bio string
	_ = db.DB.QueryRowContext(ctx, `
		SELECT canonical_name, COALESCE(category, ''), COALESCE(summary, ''), COALESCE(biography, '')
		FROM people WHERE id = $1
	`, personID).Scan(&personName, &category, &summary, &bio)

	domain := ResolveBiographicalDomain(category, bio, summary, "", "auto")
	brief, _, err := BuildProjectCompetitiveBrief(ctx, projectID, personID, discoveryID, personName, domain, targetDur)
	if err != nil {
		return nil, err
	}
	return brief, nil
}
