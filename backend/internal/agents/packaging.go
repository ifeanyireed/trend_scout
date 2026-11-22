package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"trendscout-backend/internal/ai"
	"trendscout-backend/internal/db"
)

type VideoSEOContent struct {
	RecommendedTitles []string `json:"recommendedTitles"`
	VideoDescription  string   `json:"videoDescription"`
	Tags              []string `json:"tags"`
	Hashtags          []string `json:"hashtags"`
	TagsCharCount     int      `json:"tagsCharCount"`
	UsedGemini        bool     `json:"usedGemini"`
}

var nonAlphaNumSpace = regexp.MustCompile(`[^a-zA-Z0-9\s]+`)

// NormalizeTags deduplicates, trims, strips leading #, and enforces YouTube's 500-character tag box limit (480 safe budget).
func NormalizeTags(raw []string, maxTotalChars int) []string {
	if maxTotalChars <= 0 {
		maxTotalChars = 480
	}
	seen := map[string]bool{}
	var out []string
	totalLen := 0
	for _, t := range raw {
		clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), "#"))
		clean = strings.Join(strings.Fields(clean), " ")
		if len(clean) < 2 {
			continue
		}
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		sep := 0
		if len(out) > 0 {
			sep = 2 // ", "
		}
		if totalLen+sep+len(clean) > maxTotalChars {
			continue
		}
		seen[key] = true
		out = append(out, clean)
		totalLen += sep + len(clean)
	}
	return out
}

// NormalizeHashtags ensures every hashtag starts with #, uses clean CamelCase alphanumeric characters, and has no duplicates.
func NormalizeHashtags(raw []string, maxCount int) []string {
	if maxCount <= 0 {
		maxCount = 12
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range raw {
		clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(h), "#"))
		clean = nonAlphaNumSpace.ReplaceAllString(clean, " ")
		words := strings.Fields(clean)
		if len(words) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteByte('#')
		for _, w := range words {
			if len(w) == 1 {
				b.WriteString(strings.ToUpper(w))
			} else {
				b.WriteString(strings.ToUpper(w[:1]) + w[1:])
			}
		}
		tag := b.String()
		key := strings.ToLower(tag)
		if seen[key] || len(tag) < 3 {
			continue
		}
		seen[key] = true
		out = append(out, tag)
		if len(out) >= maxCount {
			break
		}
	}
	return out
}

func TagsTotalChars(tags []string) int {
	if len(tags) == 0 {
		return 0
	}
	return len(strings.Join(tags, ", "))
}

func buildChapterTimestampsForProject(ctx context.Context, projectID string, domain BiographicalDomainProfile, targetDurationMinutes int) string {
	if projectID != "" && db.DB != nil {
		var secJSON string
		err := db.DB.QueryRowContext(ctx, `
			SELECT sv.sections_json
			FROM script_versions sv
			JOIN documentary_scripts ds ON ds.id = sv.script_id
			WHERE ds.project_id = $1
			ORDER BY sv.version_number DESC
			LIMIT 1
		`, projectID).Scan(&secJSON)
		if err == nil && secJSON != "" && secJSON != "[]" {
			var sections []ScriptSection
			if json.Unmarshal([]byte(secJSON), &sections) == nil && len(sections) > 0 {
				var sb strings.Builder
				for _, sec := range sections {
					ts := "0:00"
					if len(sec.Shots) > 0 && sec.Shots[0].Timestamp != "" {
						ts = sec.Shots[0].Timestamp
					} else if sec.TimeRange != "" {
						parts := strings.Split(sec.TimeRange, "–")
						if len(parts) > 0 {
							ts = strings.TrimSpace(parts[0])
						}
					}
					cleanTitle := sec.Title
					if idx := strings.Index(cleanTitle, "·"); idx >= 0 {
						cleanTitle = strings.TrimSpace(cleanTitle[idx+len("·"):])
					}
					sb.WriteString(fmt.Sprintf("%s %s\n", ts, cleanTitle))
				}
				if out := strings.TrimSpace(sb.String()); out != "" {
					return out
				}
			}
		}
	}

	if targetDurationMinutes < 15 || targetDurationMinutes > 25 {
		targetDurationMinutes = 20
	}
	windows := ai.ComputeChapterTimeWindows(targetDurationMinutes * 60)
	var cb strings.Builder
	for i, w := range windows {
		cb.WriteString(fmt.Sprintf("%s %s\n", formatMMSS(w[0]), domain.ChapterTitles[i]))
	}
	return strings.TrimSpace(cb.String())
}

func loadCompetitiveBriefPromptForPackaging(ctx context.Context, projectID, personID, discoveryID, personName string, domain BiographicalDomainProfile, targetDur int) (string, error) {
	if projectID != "" && db.DB != nil {
		var raw string
		if err := db.DB.QueryRowContext(ctx, `SELECT COALESCE(competitive_brief_json, '{}') FROM documentary_projects WHERE id = $1`, projectID).Scan(&raw); err == nil && raw != "" && raw != "{}" {
			var parsed CompetitivePositioningBrief
			if json.Unmarshal([]byte(raw), &parsed) == nil && parsed.UsedGemini && (parsed.OutperformanceStrategy != "" || parsed.AnalyzedVideoCount > 0) {
				return FormatCompetitiveBriefForPrompt(&parsed), nil
			}
		}
	}
	_, promptStr, err := BuildProjectCompetitiveBrief(ctx, projectID, personID, discoveryID, personName, domain, targetDur)
	if err != nil {
		return "", err
	}
	return promptStr, nil
}

// GenerateOpportunityVideoPackaging builds and persists YouTube video description, high-performing tags,
// and hashtags strictly via Gemini AI (no local fallback).
func GenerateOpportunityVideoPackaging(ctx context.Context, discoveryID string) (*VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, fmt.Errorf("Gemini AI required for YouTube SEO Packaging: no Gemini API key configured in Settings")
	}
	var personID, headline, achievement string
	err := db.DB.QueryRowContext(ctx, `
		SELECT person_id, COALESCE(headline, ''), COALESCE(achievement, '')
		FROM discoveries WHERE id = $1
	`, discoveryID).Scan(&personID, &headline, &achievement)
	if err != nil {
		return nil, err
	}

	var personName, category, country, summary, bio string
	_ = db.DB.QueryRowContext(ctx, `
		SELECT canonical_name, COALESCE(category, ''), COALESCE(country, ''), COALESCE(summary, ''), COALESCE(biography, '')
		FROM people WHERE id = $1
	`, personID).Scan(&personName, &category, &country, &summary, &bio)

	var linkedProjID string
	var targetDur int = 20
	_ = db.DB.QueryRowContext(ctx, `
		SELECT id, target_duration_minutes FROM documentary_projects
		WHERE discovery_id = $1 OR person_id = $2
		ORDER BY updated_at DESC LIMIT 1
	`, discoveryID, personID).Scan(&linkedProjID, &targetDur)
	if targetDur < 15 || targetDur > 25 {
		targetDur = 20
	}

	domain := ResolveBiographicalDomain(category, bio, summary, headline, "auto")
	chapterTS := buildChapterTimestampsForProject(ctx, linkedProjID, domain, targetDur)
	compBriefPrompt, err := loadCompetitiveBriefPromptForPackaging(ctx, linkedProjID, personID, discoveryID, personName, domain, targetDur)
	if err != nil {
		return nil, err
	}
	seo, err := generateVideoSEOWithAI(ctx, personName, domain, country, headline, achievement, bio, chapterTS, compBriefPrompt, targetDur)
	if err != nil {
		return nil, err
	}

	tagsJSON, _ := json.Marshal(seo.Tags)
	hashtagsJSON, _ := json.Marshal(seo.Hashtags)

	_, err = db.DB.ExecContext(ctx, `
		UPDATE discoveries
		SET video_description = $2, video_tags_json = $3, video_hashtags_json = $4, updated_at = NOW()
		WHERE id = $1
	`, discoveryID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON))
	if err != nil {
		return nil, err
	}

	if linkedProjID != "" {
		titlesJSON, _ := json.Marshal(seo.RecommendedTitles)
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE documentary_projects
			SET youtube_description = $2, youtube_tags_json = $3, youtube_hashtags_json = $4,
			    youtube_titles_json = CASE WHEN youtube_titles_json = '[]' OR youtube_titles_json = '' THEN $5 ELSE youtube_titles_json END,
			    updated_at = NOW()
			WHERE id = $1
		`, linkedProjID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON), string(titlesJSON))
	}

	return &seo, nil
}

// GenerateProjectVideoPackaging builds and persists YouTube video description (synced to the project's
// actual script chapters), high-performing tags, hashtags, and titles strictly via Gemini AI (no local fallback).
func GenerateProjectVideoPackaging(ctx context.Context, projectID string) (*VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, fmt.Errorf("Gemini AI required for YouTube SEO Packaging: no Gemini API key configured in Settings")
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

	var personName, category, country, summary, bio string
	_ = db.DB.QueryRowContext(ctx, `
		SELECT canonical_name, COALESCE(category, ''), COALESCE(country, ''), COALESCE(summary, ''), COALESCE(biography, '')
		FROM people WHERE id = $1
	`, personID).Scan(&personName, &category, &country, &summary, &bio)

	var headline, achievement string
	if discoveryID != "" {
		_ = db.DB.QueryRowContext(ctx, `
			SELECT COALESCE(headline, ''), COALESCE(achievement, '') FROM discoveries WHERE id = $1
		`, discoveryID).Scan(&headline, &achievement)
	}

	domain := ResolveBiographicalDomain(category, bio, summary, headline, "auto")
	chapterTS := buildChapterTimestampsForProject(ctx, projectID, domain, targetDur)
	compBriefPrompt, err := loadCompetitiveBriefPromptForPackaging(ctx, projectID, personID, discoveryID, personName, domain, targetDur)
	if err != nil {
		return nil, err
	}
	seo, err := generateVideoSEOWithAI(ctx, personName, domain, country, headline, achievement, bio, chapterTS, compBriefPrompt, targetDur)
	if err != nil {
		return nil, err
	}

	tagsJSON, _ := json.Marshal(seo.Tags)
	hashtagsJSON, _ := json.Marshal(seo.Hashtags)
	titlesJSON, _ := json.Marshal(seo.RecommendedTitles)

	_, err = db.DB.ExecContext(ctx, `
		UPDATE documentary_projects
		SET youtube_description = $2, youtube_tags_json = $3, youtube_hashtags_json = $4,
		    youtube_titles_json = $5, updated_at = NOW()
		WHERE id = $1
	`, projectID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON), string(titlesJSON))
	if err != nil {
		return nil, err
	}

	if discoveryID != "" {
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE discoveries
			SET video_description = $2, video_tags_json = $3, video_hashtags_json = $4, updated_at = NOW()
			WHERE id = $1
		`, discoveryID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON))
	}

	return &seo, nil
}

func generateVideoSEOWithAI(
	ctx context.Context,
	personName string,
	domain BiographicalDomainProfile,
	country string,
	headline string,
	achievement string,
	bio string,
	chapterTS string,
	competitiveBrief string,
	targetDur int,
) (VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return VideoSEOContent{}, fmt.Errorf("Gemini AI required for YouTube SEO Packaging: no Gemini API key configured in Settings")
	}

	bioInput := bio
	if len(bioInput) > 4500 {
		bioInput = bioInput[:4500]
	}
	aiRes, err := ai.GenerateVideoPackagingWithGemini(
		ctx, personName, domain.Label, country, headline, achievement, bioInput, chapterTS, competitiveBrief, targetDur,
	)
	if err != nil || aiRes == nil {
		return VideoSEOContent{}, fmt.Errorf("Gemini AI YouTube SEO Packaging failed: %s", sanitizeAIErrorForUI(err))
	}
	if strings.TrimSpace(aiRes.VideoDescription) == "" || len(aiRes.RecommendedTitles) == 0 {
		return VideoSEOContent{}, fmt.Errorf("Gemini AI returned incomplete YouTube SEO packaging (missing description or titles)")
	}

	tags := NormalizeTags(aiRes.Tags, 480)
	hashtags := NormalizeHashtags(aiRes.Hashtags, 12)
	return VideoSEOContent{
		RecommendedTitles: aiRes.RecommendedTitles,
		VideoDescription:  strings.TrimSpace(aiRes.VideoDescription),
		Tags:              tags,
		Hashtags:          hashtags,
		TagsCharCount:     TagsTotalChars(tags),
		UsedGemini:        true,
	}, nil
}
