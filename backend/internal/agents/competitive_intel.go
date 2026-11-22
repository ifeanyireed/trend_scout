package agents

import (
	"context"
	"encoding/json"
	"fmt"
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
	TranscriptAvailable bool    `json:"transcriptAvailable"`
	CommentsCount       int     `json:"commentsCount"`
}

type CompetitivePositioningBrief struct {
	AnalyzedVideoCount      int                   `json:"analyzedVideoCount"`
	TranscribedVideoCount   int                   `json:"transcribedVideoCount"`
	CommentsAnalyzedCount   int                   `json:"commentsAnalyzedCount"`
	WinningHooks            []string              `json:"winningHooks"`
	WhatsWorking            []string              `json:"whatsWorking"`
	AudienceQuestions       []string              `json:"audienceQuestions"`
	CompetitorBlindSpots    []string              `json:"competitorBlindSpots"`
	OutperformanceStrategy  string                `json:"outperformanceStrategy"`
	RecommendedColdOpenEdge string                `json:"recommendedColdOpenEdge"`
	FormatAdvantage         string                `json:"formatAdvantage"`
	StudiedVideos           []StudiedVideoSummary `json:"studiedVideos"`
	UsedGemini              bool                  `json:"usedGemini"`
	GeneratedAt             string                `json:"generatedAt"`
}

// BuildDeterministicCompetitiveBrief synthesizes a grounded competitive brief from real YouTube metrics,
// transcripts, and viewer comments when Gemini is unavailable (or as baseline before Gemini refinement).
func BuildDeterministicCompetitiveBrief(
	personName string,
	domain BiographicalDomainProfile,
	studies []connectors.CompetitorVideoStudy,
	targetDurationMinutes int,
) CompetitivePositioningBrief {
	brief := CompetitivePositioningBrief{
		AnalyzedVideoCount: len(studies),
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	shortCount := 0
	longCount := 0
	topOutlier := 1.0
	topTitle := ""
	for _, st := range studies {
		if st.TranscriptAvailable {
			brief.TranscribedVideoCount++
		}
		brief.CommentsAnalyzedCount += len(st.TopComments)
		if strings.Contains(st.VideoFormat, "Short") {
			shortCount++
		} else if strings.Contains(st.VideoFormat, "Long-Form") {
			longCount++
		}
		if st.ChannelMultiplier > topOutlier {
			topOutlier = st.ChannelMultiplier
			topTitle = st.Title
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
			TranscriptAvailable: st.TranscriptAvailable,
			CommentsCount:       len(st.TopComments),
		})
	}

	if len(studies) == 0 {
		brief.WinningHooks = []string{
			fmt.Sprintf("No high-velocity YouTube documentary yet dominates %s—first-mover prestige framing owns the search space.", personName),
		}
		brief.WhatsWorking = []string{
			fmt.Sprintf("Organic audience curiosity around %s in %s with virtually zero long-form documentary supply.", personName, domain.Label),
		}
		brief.AudienceQuestions = []string{
			fmt.Sprintf("How did %s go from %s to their defining public turning point?", personName, domain.OriginRoom),
			"What verified primary records separate public myth from documented history?",
		}
		brief.CompetitorBlindSpots = []string{
			"Zero competing long-form biographical documentaries currently anchor the search results.",
			"Existing news clips summarize outcomes without examining formative failures or institutional tension.",
		}
		brief.OutperformanceStrategy = fmt.Sprintf("Establish the definitive %d-minute source-grounded biographical documentary on %s, pairing restrained archival journalism with <=7s visual pacing.", targetDurationMinutes, personName)
		brief.RecommendedColdOpenEdge = fmt.Sprintf("Open in media res inside %s before pulling back to %s to lock curiosity in the first 30 seconds.", domain.OpeningRoom, domain.OriginRoom)
		brief.FormatAdvantage = fmt.Sprintf("Full %d-minute chaptered documentary captures high watch-time sessions that short clips cannot retain.", targetDurationMinutes)
		return brief
	}

	if topTitle != "" {
		brief.WinningHooks = append(brief.WinningHooks, fmt.Sprintf("Top outlier (%.1fx channel baseline): \"%s\" proves strong viewer appetite for %s's turning-point narrative.", topOutlier, truncateText(topTitle, 80), personName))
	}
	for _, st := range studies {
		if st.TranscriptAvailable && len(st.TranscriptExcerpt) > 40 {
			openSnippet := truncateText(st.TranscriptExcerpt, 130)
			brief.WinningHooks = append(brief.WinningHooks, fmt.Sprintf("Transcribed opening cadence on %s (%s): \"%s\"", st.ChannelName, st.VideoFormat, openSnippet))
		}
		if len(brief.WinningHooks) >= 3 {
			break
		}
	}

	brief.WhatsWorking = []string{
		fmt.Sprintf("Outlier demand up to %.1fx channel baseline demonstrates high browse/search pull for %s.", topOutlier, personName),
		"Immediate human stakes and recognizable turning points drive strong viewer shares and comment debate.",
	}

	// Surface real top viewer comments as audience questions/signals
	for _, st := range studies {
		for _, c := range st.TopComments {
			brief.AudienceQuestions = append(brief.AudienceQuestions, c)
			if len(brief.AudienceQuestions) >= 4 {
				break
			}
		}
		if len(brief.AudienceQuestions) >= 4 {
			break
		}
	}
	if len(brief.AudienceQuestions) == 0 {
		brief.AudienceQuestions = []string{
			fmt.Sprintf("What happened behind the scenes before %s's public breakthrough?", personName),
			"What competing claims, setbacks, or unresolved controversies remain on the record?",
		}
	}

	if longCount == 0 {
		brief.CompetitorBlindSpots = append(brief.CompetitorBlindSpots,
			fmt.Sprintf("Wide open long-form gap: %d of %d top competing videos are Shorts/clips with zero deep 15–25 minute documentaries.", shortCount, len(studies)))
	} else {
		brief.CompetitorBlindSpots = append(brief.CompetitorBlindSpots,
			"Competing uploads rely on surface recaps rather than rigorously attributed multi-chapter investigative journalism.")
	}
	brief.CompetitorBlindSpots = append(brief.CompetitorBlindSpots,
		"Lack of balanced ledger framing contrasting documented achievements against setbacks, disputes, and open questions.",
		"Slow static B-roll pacing compared to disciplined <=7-second visual storytelling.")

	brief.OutperformanceStrategy = fmt.Sprintf(
		"Outperform all %d analyzed YouTube videos by opening inside %s with immediate unresolved tension, delivering the complete chronological record (%s) that Shorts and news clips skip, and maintaining <=7s visual cuts throughout all %d minutes.",
		len(studies), domain.OpeningRoom, domain.Label, targetDurationMinutes,
	)
	brief.RecommendedColdOpenEdge = fmt.Sprintf("Contrast %s's most consequential public moment directly against %s within the first 45 seconds.", personName, domain.OriginRoom)
	if longCount == 0 {
		brief.FormatAdvantage = fmt.Sprintf("Exploit the wide-open documentary gap: existing coverage is dominated by Shorts/short clips (%d Short(s), 0 Long-Form), leaving high-retention %d-minute search and browse traffic uncontested.", shortCount, targetDurationMinutes)
	} else {
		brief.FormatAdvantage = fmt.Sprintf("Beat existing long-form uploads (%d detected) through austere source-attributed journalism, balanced ledger framing, and disciplined <=7s visual pacing.", longCount)
	}

	return brief
}

// FormatCompetitiveBriefForPrompt renders a concise, actionable directive injected into Stage 1 & Stage 2 script prompts.
func FormatCompetitiveBriefForPrompt(brief *CompetitivePositioningBrief) string {
	if brief == nil {
		return "No competing YouTube videos analyzed yet; focus on definitive prestige investigative storytelling."
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
	return strings.TrimSpace(sb.String())
}

// BuildProjectCompetitiveBrief runs Stage 0 pre-script YouTube competitor study (transcribing top videos,
// mining top viewer comments, synthesizing positioning via Gemini, and saving onto documentary_projects).
func BuildProjectCompetitiveBrief(
	ctx context.Context,
	projectID string,
	personID string,
	discoveryID string,
	personName string,
	domain BiographicalDomainProfile,
	targetDurationMinutes int,
) (*CompetitivePositioningBrief, string) {
	aiCfg := ai.GetAIConfig(ctx)

	// 1. Load tracked top YouTube videos from DB (or fetch live if none tracked yet)
	var topVideos []connectors.VideoMetric
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

	if len(topVideos) == 0 && personName != "" {
		ytCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
		fetched, errYT := connectors.LookupYouTubeIntelForPerson(ytCtx, personName, aiCfg.YouTubeAPIKey)
		cancel()
		if errYT == nil && len(fetched) > 0 {
			topVideos = fetched
			if discoveryID != "" {
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

	// 3. Build deterministic baseline brief, then upgrade via Gemini if API key is active
	brief := BuildDeterministicCompetitiveBrief(personName, domain, studies, targetDurationMinutes)

	if len(studies) > 0 && ai.HasGeminiKey(ctx) {
		studiesJSON, _ := json.Marshal(studies)
		aiCtx, cancelAI := context.WithTimeout(ctx, 25*time.Second)
		aiBrief, errAI := ai.AnalyzeCompetitorsForPositioningWithGemini(aiCtx, personName, domain.Label, string(studiesJSON), targetDurationMinutes)
		cancelAI()
		if errAI == nil && aiBrief != nil {
			brief.UsedGemini = true
			if len(aiBrief.WinningHooks) > 0 {
				brief.WinningHooks = aiBrief.WinningHooks
			}
			if len(aiBrief.WhatsWorking) > 0 {
				brief.WhatsWorking = aiBrief.WhatsWorking
			}
			if len(aiBrief.AudienceQuestions) > 0 {
				brief.AudienceQuestions = aiBrief.AudienceQuestions
			}
			if len(aiBrief.CompetitorBlindSpots) > 0 {
				brief.CompetitorBlindSpots = aiBrief.CompetitorBlindSpots
			}
			if strings.TrimSpace(aiBrief.OutperformanceStrategy) != "" {
				brief.OutperformanceStrategy = strings.TrimSpace(aiBrief.OutperformanceStrategy)
			}
			if strings.TrimSpace(aiBrief.RecommendedColdOpenEdge) != "" {
				brief.RecommendedColdOpenEdge = strings.TrimSpace(aiBrief.RecommendedColdOpenEdge)
			}
			if strings.TrimSpace(aiBrief.FormatAdvantage) != "" {
				brief.FormatAdvantage = strings.TrimSpace(aiBrief.FormatAdvantage)
			}
		}
	}

	// 4. Persist onto documentary_projects for Studio inspection & export
	if projectID != "" {
		if b, errM := json.Marshal(brief); errM == nil {
			_, _ = db.DB.ExecContext(ctx, `
				UPDATE documentary_projects SET competitive_brief_json = $2, updated_at = NOW() WHERE id = $1
			`, projectID, string(b))
		}
	}

	return &brief, FormatCompetitiveBriefForPrompt(&brief)
}

// GenerateProjectCompetitiveBrief triggers Stage 0 on demand for an existing documentary project.
func GenerateProjectCompetitiveBrief(ctx context.Context, projectID string) (*CompetitivePositioningBrief, error) {
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
	brief, _ := BuildProjectCompetitiveBrief(ctx, projectID, personID, discoveryID, personName, domain, targetDur)
	return brief, nil
}
