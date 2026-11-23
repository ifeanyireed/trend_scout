package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"trendscout-backend/internal/ai"
	"trendscout-backend/internal/connectors"
	"trendscout-backend/internal/db"
)

type VideoSEOContent struct {
	RecommendedTitles []string                           `json:"recommendedTitles"`
	VideoDescription  string                             `json:"videoDescription"`
	Tags              []string                           `json:"tags"`
	Hashtags          []string                           `json:"hashtags"`
	YouTubeMentions   []string                           `json:"youtubeMentions"`
	TwitterCaption    string                             `json:"twitterCaption"`
	TwitterHandles    []string                           `json:"twitterHandles"`
	InstagramCaption  string                             `json:"instagramCaption"`
	InstagramHandles  []string                           `json:"instagramHandles"`
	VerifiedAccounts  []connectors.VerifiedSocialAccount `json:"verifiedAccounts"`
	TagsCharCount     int                                `json:"tagsCharCount"`
	UsedGemini        bool                               `json:"usedGemini"`
}

var nonAlphaNumSpace = regexp.MustCompile(`[^a-zA-Z0-9\s]+`)
var nonHandleChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// NormalizeHandles cleans, deduplicates, formats @handles, and filters out fan-page/impersonator patterns.
func NormalizeHandles(raw []string, maxCount int) []string {
	if maxCount <= 0 {
		maxCount = 12
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range raw {
		clean := strings.TrimSpace(h)
		// Strip full URLs if an AI returned e.g. https://youtube.com/@foo or https://x.com/foo or https://instagram.com/foo
		for _, prefix := range []string{
			"https://www.youtube.com/", "https://youtube.com/", "http://youtube.com/",
			"https://twitter.com/", "https://www.twitter.com/", "https://x.com/", "https://www.x.com/",
			"https://instagram.com/", "https://www.instagram.com/",
		} {
			if strings.HasPrefix(strings.ToLower(clean), prefix) {
				clean = clean[len(prefix):]
				break
			}
		}
		if idx := strings.IndexAny(clean, " ?/"); idx >= 0 {
			clean = clean[:idx]
		}
		clean = strings.TrimPrefix(clean, "@")
		clean = nonHandleChars.ReplaceAllString(clean, "")
		clean = strings.Trim(clean, ".-")
		if len(clean) < 2 {
			continue
		}
		handle := "@" + clean
		if connectors.IsLikelyFanOrImpersonatorHandle(handle) {
			continue
		}
		key := strings.ToLower(handle)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, handle)
		if len(out) >= maxCount {
			break
		}
	}
	return out
}

// EnsureYouTubeMentionsInDescription guarantees that all normalized YouTube @handles appear
// inside the YouTube video description (inserting a "Featured & Tagged Channels:" line before
// the closing hashtags if any handles are missing).
func EnsureYouTubeMentionsInDescription(desc string, mentions []string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" || len(mentions) == 0 {
		return desc
	}
	lowDesc := strings.ToLower(desc)
	var missing []string
	for _, m := range mentions {
		if !strings.Contains(lowDesc, strings.ToLower(m)) {
			missing = append(missing, m)
		}
	}
	if len(missing) == 0 {
		return desc
	}
	tagLine := "Featured & Tagged Channels: " + strings.Join(missing, " ")
	lines := strings.Split(desc, "\n")
	// Insert right before a trailing hashtags-only line if present
	lastIdx := len(lines) - 1
	for lastIdx >= 0 && strings.TrimSpace(lines[lastIdx]) == "" {
		lastIdx--
	}
	if lastIdx >= 0 && strings.HasPrefix(strings.TrimSpace(lines[lastIdx]), "#") {
		before := strings.TrimRight(strings.Join(lines[:lastIdx], "\n"), "\n")
		after := strings.Join(lines[lastIdx:], "\n")
		return before + "\n\n" + tagLine + "\n\n" + after
	}
	return desc + "\n\n" + tagLine
}

// ensureVerifiedHandlesInSocialCaption guarantees that verified platform-specific @handles
// appear in the X or Instagram caption without duplicating handles already mentioned or overflowing maxChars.
func ensureVerifiedHandlesInSocialCaption(caption string, verifiedHandles []string, maxChars int) string {
	caption = strings.TrimSpace(caption)
	if caption == "" || len(verifiedHandles) == 0 {
		return caption
	}
	low := strings.ToLower(caption)
	var missing []string
	budget := maxChars - len(caption) - 2
	for _, h := range verifiedHandles {
		if !strings.Contains(low, strings.ToLower(h)) {
			if maxChars > 0 && budget < len(h)+1 {
				continue
			}
			missing = append(missing, h)
			budget -= len(h) + 1
		}
	}
	if len(missing) == 0 {
		return caption
	}
	lines := strings.Split(caption, "\n")
	lastIdx := len(lines) - 1
	for lastIdx >= 0 && strings.TrimSpace(lines[lastIdx]) == "" {
		lastIdx--
	}
	if lastIdx >= 0 && strings.HasPrefix(strings.TrimSpace(lines[lastIdx]), "#") {
		before := strings.TrimRight(strings.Join(lines[:lastIdx], "\n"), "\n")
		after := strings.Join(lines[lastIdx:], "\n")
		return before + "\n\n" + strings.Join(missing, " ") + "\n\n" + after
	}
	return caption + "\n\n" + strings.Join(missing, " ")
}

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

func loadMediaAndPublishersForPerson(ctx context.Context, personID string) ([]string, string) {
	if personID == "" || db.DB == nil {
		return nil, ""
	}
	seen := map[string]bool{}
	var names []string
	addName := func(n string) {
		clean := strings.TrimSpace(n)
		low := strings.ToLower(clean)
		if len(clean) < 2 || seen[low] || strings.Contains(low, "wikipedia") || strings.Contains(low, "wikimedia") {
			return
		}
		seen[low] = true
		names = append(names, clean)
	}

	if rows, err := db.DB.QueryContext(ctx, `SELECT DISTINCT publisher FROM news_articles WHERE person_id = $1 LIMIT 6`, personID); err == nil && rows != nil {
		for rows.Next() {
			var pub string
			if rows.Scan(&pub) == nil {
				addName(pub)
			}
		}
		rows.Close()
	}
	if rows, err := db.DB.QueryContext(ctx, `SELECT DISTINCT publisher FROM research_sources WHERE person_id = $1 LIMIT 6`, personID); err == nil && rows != nil {
		for rows.Next() {
			var pub string
			if rows.Scan(&pub) == nil {
				addName(pub)
			}
		}
		rows.Close()
	}
	if rows, err := db.DB.QueryContext(ctx, `SELECT DISTINCT channel_name FROM videos WHERE person_id = $1 ORDER BY channel_name LIMIT 6`, personID); err == nil && rows != nil {
		for rows.Next() {
			var ch string
			if rows.Scan(&ch) == nil {
				addName(ch)
			}
		}
		rows.Close()
	}
	return names, strings.Join(names, ", ")
}

func loadSourcesCreditsForPerson(ctx context.Context, personID string) string {
	if personID == "" || db.DB == nil {
		return ""
	}
	rows, err := db.DB.QueryContext(ctx, `
		SELECT title, publisher, url
		FROM research_sources
		WHERE person_id = $1
		ORDER BY reliability_score DESC, created_at ASC
		LIMIT 6
	`, personID)
	if err != nil || rows == nil {
		return ""
	}
	defer rows.Close()
	var sb strings.Builder
	for rows.Next() {
		var t, pub, u string
		if rows.Scan(&t, &pub, &u) == nil {
			if strings.TrimSpace(pub) != "" && strings.TrimSpace(u) != "" {
				sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", strings.TrimSpace(t), strings.TrimSpace(pub), strings.TrimSpace(u)))
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// GenerateOpportunityVideoPackaging builds and persists 3 platform-specific publishing packages:
// YouTube description (with verified @mentions & sources/credits), X/Twitter caption, and Instagram caption.
func GenerateOpportunityVideoPackaging(ctx context.Context, discoveryID string) (*VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, fmt.Errorf("AI required for YouTube SEO Packaging: no AI API key configured in Settings")
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
	sourcesCredits := loadSourcesCreditsForPerson(ctx, personID)
	mediaNames, mediaSummary := loadMediaAndPublishersForPerson(ctx, personID)
	seo, err := generateVideoSEOWithAI(ctx, personName, domain, country, headline, achievement, bio, chapterTS, compBriefPrompt, sourcesCredits, mediaNames, mediaSummary, targetDur)
	if err != nil {
		return nil, err
	}

	tagsJSON, _ := json.Marshal(seo.Tags)
	hashtagsJSON, _ := json.Marshal(seo.Hashtags)
	ytMentionsJSON, _ := json.Marshal(seo.YouTubeMentions)
	twHandlesJSON, _ := json.Marshal(seo.TwitterHandles)
	igHandlesJSON, _ := json.Marshal(seo.InstagramHandles)
	verifiedJSON, _ := json.Marshal(seo.VerifiedAccounts)

	_, err = db.DB.ExecContext(ctx, `
		UPDATE discoveries
		SET video_description = $2, video_tags_json = $3, video_hashtags_json = $4,
		    youtube_mentions_json = $5, twitter_caption = $6, twitter_handles_json = $7,
		    instagram_caption = $8, instagram_handles_json = $9, verified_accounts_json = $10,
		    updated_at = NOW()
		WHERE id = $1
	`, discoveryID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON),
		string(ytMentionsJSON), seo.TwitterCaption, string(twHandlesJSON),
		seo.InstagramCaption, string(igHandlesJSON), string(verifiedJSON))
	if err != nil {
		return nil, err
	}

	if linkedProjID != "" {
		titlesJSON, _ := json.Marshal(seo.RecommendedTitles)
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE documentary_projects
			SET youtube_description = $2, youtube_tags_json = $3, youtube_hashtags_json = $4,
			    youtube_mentions_json = $5, twitter_caption = $6, twitter_handles_json = $7,
			    instagram_caption = $8, instagram_handles_json = $9, verified_accounts_json = $10,
			    youtube_titles_json = CASE WHEN youtube_titles_json = '[]' OR youtube_titles_json = '' THEN $11 ELSE youtube_titles_json END,
			    updated_at = NOW()
			WHERE id = $1
		`, linkedProjID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON),
			string(ytMentionsJSON), seo.TwitterCaption, string(twHandlesJSON),
			seo.InstagramCaption, string(igHandlesJSON), string(verifiedJSON), string(titlesJSON))
	}

	return &seo, nil
}

// GenerateProjectVideoPackaging builds and persists 3 platform-specific publishing packages synced to
// the project's actual script chapters and verified official social accounts strictly via AI.
func GenerateProjectVideoPackaging(ctx context.Context, projectID string) (*VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return nil, fmt.Errorf("AI required for YouTube SEO Packaging: no AI API key configured in Settings")
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
	sourcesCredits := loadSourcesCreditsForPerson(ctx, personID)
	mediaNames, mediaSummary := loadMediaAndPublishersForPerson(ctx, personID)
	seo, err := generateVideoSEOWithAI(ctx, personName, domain, country, headline, achievement, bio, chapterTS, compBriefPrompt, sourcesCredits, mediaNames, mediaSummary, targetDur)
	if err != nil {
		return nil, err
	}

	tagsJSON, _ := json.Marshal(seo.Tags)
	hashtagsJSON, _ := json.Marshal(seo.Hashtags)
	titlesJSON, _ := json.Marshal(seo.RecommendedTitles)
	ytMentionsJSON, _ := json.Marshal(seo.YouTubeMentions)
	twHandlesJSON, _ := json.Marshal(seo.TwitterHandles)
	igHandlesJSON, _ := json.Marshal(seo.InstagramHandles)
	verifiedJSON, _ := json.Marshal(seo.VerifiedAccounts)

	_, err = db.DB.ExecContext(ctx, `
		UPDATE documentary_projects
		SET youtube_description = $2, youtube_tags_json = $3, youtube_hashtags_json = $4,
		    youtube_titles_json = $5, youtube_mentions_json = $6, twitter_caption = $7, twitter_handles_json = $8,
		    instagram_caption = $9, instagram_handles_json = $10, verified_accounts_json = $11,
		    updated_at = NOW()
		WHERE id = $1
	`, projectID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON), string(titlesJSON),
		string(ytMentionsJSON), seo.TwitterCaption, string(twHandlesJSON),
		seo.InstagramCaption, string(igHandlesJSON), string(verifiedJSON))
	if err != nil {
		return nil, err
	}

	if discoveryID != "" {
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE discoveries
			SET video_description = $2, video_tags_json = $3, video_hashtags_json = $4,
			    youtube_mentions_json = $5, twitter_caption = $6, twitter_handles_json = $7,
			    instagram_caption = $8, instagram_handles_json = $9, verified_accounts_json = $10,
			    updated_at = NOW()
			WHERE id = $1
		`, discoveryID, seo.VideoDescription, string(tagsJSON), string(hashtagsJSON),
			string(ytMentionsJSON), seo.TwitterCaption, string(twHandlesJSON),
			seo.InstagramCaption, string(igHandlesJSON), string(verifiedJSON))
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
	sourcesCredits string,
	mediaNames []string,
	mediaSummary string,
	targetDur int,
) (VideoSEOContent, error) {
	if !ai.HasGeminiKey(ctx) {
		return VideoSEOContent{}, fmt.Errorf("AI required for YouTube SEO Packaging: no AI API key configured in Settings")
	}

	// Step 1: Identify all connected people & organizations across all 7 relationship categories:
	// subject, family/influential, coaches/mentors/trainers, teams/clubs/orgs, governing bodies,
	// competitors/contemporaries, and journalists/commentators/media organizations.
	aiCfg := ai.GetAIConfig(ctx)
	targets := []connectors.ConnectedEntityTarget{
		{Name: personName, RelationGroup: "subject", RelationRole: "Documentary Subject"},
	}
	if identified, errID := ai.IdentifyConnectedEntitiesWithAI(ctx, personName, domain.Label, country, headline, achievement, bio, mediaSummary); errID == nil && len(identified) > 0 {
		for _, ent := range identified {
			if strings.TrimSpace(ent.Name) == "" {
				continue
			}
			targets = append(targets, connectors.ConnectedEntityTarget{
				Name:          ent.Name,
				RelationGroup: ent.RelationGroup,
				RelationRole:  ent.RelationRole,
			})
		}
	}
	// Supplement with regex-extracted entities & tracked media outlets if slots remain
	for _, m := range mediaNames {
		targets = append(targets, connectors.ConnectedEntityTarget{
			Name:          m,
			RelationGroup: "media_journalists",
			RelationRole:  "Media Coverage",
		})
	}
	for _, cand := range connectors.ExtractCandidateEntities(personName, bio, headline, achievement, nil) {
		targets = append(targets, connectors.ConnectedEntityTarget{
			Name:          cand,
			RelationGroup: "teams_organizations",
			RelationRole:  "Notable Entity",
		})
	}

	// Step 2: Independently verify genuine official social accounts via Wikipedia/Wikidata Official Claims
	// (P2002 X, P2003 IG, P11245/P2397 YouTube) & YouTube Official Channel API (zero fan pages/impersonators).
	verifiedAccounts := connectors.VerifyConnectedEntitiesAccounts(ctx, targets, aiCfg.YouTubeAPIKey)
	verifiedRegistryPrompt := connectors.FormatVerifiedAccountsForPrompt(verifiedAccounts)

	bioInput := bio
	if len(bioInput) > 4500 {
		bioInput = bioInput[:4500]
	}
	aiRes, err := ai.GenerateVideoPackagingWithGemini(
		ctx, personName, domain.Label, country, headline, achievement, bioInput, chapterTS,
		competitiveBrief, verifiedRegistryPrompt, sourcesCredits, targetDur,
	)
	if err != nil || aiRes == nil {
		return VideoSEOContent{}, fmt.Errorf("%s AI YouTube SEO Packaging failed: %s", ai.ActiveProviderLabel(ctx), sanitizeAIErrorForUI(err))
	}
	if strings.TrimSpace(aiRes.VideoDescription) == "" || len(aiRes.RecommendedTitles) == 0 {
		return VideoSEOContent{}, fmt.Errorf("%s AI returned incomplete YouTube SEO packaging (missing description or titles)", ai.ActiveProviderLabel(ctx))
	}

	// Prioritize verified official handles first, followed by AI-suggested official handles (fan/impersonator filtered)
	var verifiedYT, verifiedTW, verifiedIG []string
	for _, acc := range verifiedAccounts {
		if acc.YouTubeHandle != "" {
			verifiedYT = append(verifiedYT, acc.YouTubeHandle)
		}
		if acc.TwitterHandle != "" {
			verifiedTW = append(verifiedTW, acc.TwitterHandle)
		}
		if acc.InstagramHandle != "" {
			verifiedIG = append(verifiedIG, acc.InstagramHandle)
		}
	}

	tags := NormalizeTags(aiRes.Tags, 480)
	hashtags := NormalizeHashtags(aiRes.Hashtags, 12)
	ytMentions := NormalizeHandles(append(verifiedYT, aiRes.YouTubeMentions...), 10)
	twHandles := NormalizeHandles(append(verifiedTW, aiRes.TwitterHandles...), 10)
	igHandles := NormalizeHandles(append(verifiedIG, aiRes.InstagramHandles...), 10)
	desc := EnsureYouTubeMentionsInDescription(aiRes.VideoDescription, ytMentions)
	twCaption := ensureVerifiedHandlesInSocialCaption(aiRes.TwitterCaption, verifiedTW, 278)
	igCaption := ensureVerifiedHandlesInSocialCaption(aiRes.InstagramCaption, verifiedIG, 2100)
	if verifiedAccounts == nil {
		verifiedAccounts = []connectors.VerifiedSocialAccount{}
	}

	return VideoSEOContent{
		RecommendedTitles: aiRes.RecommendedTitles,
		VideoDescription:  desc,
		Tags:              tags,
		Hashtags:          hashtags,
		YouTubeMentions:   ytMentions,
		TwitterCaption:    twCaption,
		TwitterHandles:    twHandles,
		InstagramCaption:  igCaption,
		InstagramHandles:  igHandles,
		VerifiedAccounts:  verifiedAccounts,
		TagsCharCount:     TagsTotalChars(tags),
		UsedGemini:        true,
	}, nil
}
