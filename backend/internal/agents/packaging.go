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

// BuildDeterministicVideoPackaging creates high-performing YouTube description, tags (<=480 chars),
// hashtags, and titles grounded in the subject's biographical domain and chapter timestamps.
func BuildDeterministicVideoPackaging(
	personName string,
	domain BiographicalDomainProfile,
	country string,
	headline string,
	achievement string,
	summary string,
	chapterTimestamps string,
	targetDurationMinutes int,
) VideoSEOContent {
	name := strings.TrimSpace(personName)
	if name == "" {
		name = "Documentary Subject"
	}
	if targetDurationMinutes < 15 || targetDurationMinutes > 25 {
		targetDurationMinutes = 20
	}
	if strings.TrimSpace(chapterTimestamps) == "" {
		windows := ai.ComputeChapterTimeWindows(targetDurationMinutes * 60)
		var cb strings.Builder
		for i, w := range windows {
			cb.WriteString(fmt.Sprintf("%s %s\n", formatMMSS(w[0]), domain.ChapterTitles[i]))
		}
		chapterTimestamps = strings.TrimSpace(cb.String())
	}

	hookAchievement := cleanShortSentence(firstNonEmpty(achievement, headline, summary), 26)
	if hookAchievement == "" {
		hookAchievement = fmt.Sprintf("How %s rose from %s to reshape %s.", name, domain.OriginRoom, domain.Label)
	}
	overview := cleanShortSentence(firstNonEmpty(summary, headline, achievement), 42)
	if overview == "" {
		overview = fmt.Sprintf("An investigative %d-minute biographical documentary tracing %s from early origins to public reckoning.", targetDurationMinutes, name)
	}

	titles := []string{
		fmt.Sprintf("%s: The Complete %s Documentary", name, domain.Label),
		fmt.Sprintf("How %s Changed %s — And What the Record Shows", name, domain.Label),
		fmt.Sprintf("Inside the Rise & Reckoning of %s (Full Documentary)", name),
	}

	// Domain-specific high-intent SEO keywords
	domainKeywords := map[string][]string{
		"tech_innovation":    {"tech documentary", "silicon valley history", "tech founder biography", "platform accountability", "innovation story", "big tech"},
		"business_founders":  {"business documentary", "founder biography", "corporate history", "entrepreneur story", "billionaire rise", "business profile"},
		"science_medicine":   {"science documentary", "medical breakthrough", "scientist biography", "research history", "nobel story", "scientific discovery"},
		"politics_law":       {"political documentary", "legal history", "public inquiry", "investigative biography", "governance", "courtroom record"},
		"sports_competition": {"sports documentary", "underdog story", "athlete biography", "championship comeback", "sports history", "unseeded champion"},
		"arts_culture":       {"arts documentary", "cultural biography", "cinema history", "creator profile", "behind the work", "artist documentary"},
		"humanitarian_civic": {"humanitarian documentary", "civic leadership", "social impact story", "activist biography", "investigative profile", "changemaker"},
	}
	domTags := domainKeywords[domain.Key]
	if len(domTags) == 0 {
		domTags = []string{"biographical documentary", "investigative documentary", "true story", "untold biography", "mini documentary"}
	}

	rawTags := []string{
		name,
		name + " documentary",
		name + " biography",
		name + " true story",
		name + " full story",
		name + " rise and reckoning",
		strings.ToLower(domain.Label),
		"investigative documentary",
		"biographical documentary",
		"mini documentary",
		"underdog documentary",
		"documentary film",
	}
	if strings.TrimSpace(country) != "" && !strings.EqualFold(country, "Global") {
		rawTags = append(rawTags, name+" "+country, country+" documentary")
	}
	rawTags = append(rawTags, domTags...)
	// Add concise keywords from headline if useful
	for _, w := range extractHeadlineKeywords(headline) {
		rawTags = append(rawTags, name+" "+w)
	}
	tags := NormalizeTags(rawTags, 480)

	rawHashtags := []string{
		name,
		domain.Label,
		"Documentary",
		"Biography",
		"InvestigativeDocumentary",
		"TrueStory",
		"MiniDoc",
		"UnderdogStory",
	}
	if strings.TrimSpace(country) != "" && !strings.EqualFold(country, "Global") {
		rawHashtags = append(rawHashtags, country)
	}
	for _, dt := range domTags[:minInt(3, len(domTags))] {
		rawHashtags = append(rawHashtags, dt)
	}
	hashtags := NormalizeHashtags(rawHashtags, 10)

	footerTags := hashtags
	if len(footerTags) > 4 {
		footerTags = footerTags[:4]
	}

	desc := fmt.Sprintf(`%s — %s

In this %d-minute investigative biographical documentary, we examine the documented trajectory of %s inside %s: %s

%s

Chapters:
%s

Editorial Note:
Every milestone, date, and attribution in this film is grounded in public records, archival journalism, and verified reporting. Where claims are contested, both sides of the public record are presented side by side.

If you value restrained, source-grounded biographical documentaries, subscribe and share which moment in %s's record stood out most to you in the comments.

%s`,
		name,
		hookAchievement,
		targetDurationMinutes,
		name,
		domain.Label,
		overview,
		domain.CentralQuestion,
		chapterTimestamps,
		name,
		strings.Join(footerTags, " "),
	)

	return VideoSEOContent{
		RecommendedTitles: titles,
		VideoDescription:  strings.TrimSpace(desc),
		Tags:              tags,
		Hashtags:          hashtags,
		TagsCharCount:     TagsTotalChars(tags),
		UsedGemini:        false,
	}
}

func extractHeadlineKeywords(headline string) []string {
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true, "that": true,
		"this": true, "into": true, "over": true, "after": true, "before": true, "about": true,
		"their": true, "have": true, "been": true, "will": true, "amid": true, "says": true,
	}
	cleaned := nonAlphaNumSpace.ReplaceAllString(strings.ToLower(headline), " ")
	var out []string
	for _, w := range strings.Fields(cleaned) {
		if len(w) >= 5 && !stop[w] {
			out = append(out, w)
			if len(out) >= 3 {
				break
			}
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildChapterTimestampsForProject(ctx context.Context, projectID string, domain BiographicalDomainProfile, targetDurationMinutes int) string {
	if projectID != "" {
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

// GenerateOpportunityVideoPackaging builds and persists YouTube video description, high-performing tags,
// and hashtags for a discovery opportunity (and syncs to any linked documentary project).
func GenerateOpportunityVideoPackaging(ctx context.Context, discoveryID string) (*VideoSEOContent, error) {
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
	seo := generateVideoSEOWithFallback(ctx, personName, domain, country, headline, achievement, summary, bio, chapterTS, targetDur)

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
// actual script chapters), high-performing tags, hashtags, and titles for a documentary project.
func GenerateProjectVideoPackaging(ctx context.Context, projectID string) (*VideoSEOContent, error) {
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
	seo := generateVideoSEOWithFallback(ctx, personName, domain, country, headline, achievement, summary, bio, chapterTS, targetDur)

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

func generateVideoSEOWithFallback(
	ctx context.Context,
	personName string,
	domain BiographicalDomainProfile,
	country string,
	headline string,
	achievement string,
	summary string,
	bio string,
	chapterTS string,
	targetDur int,
) VideoSEOContent {
	fallback := BuildDeterministicVideoPackaging(personName, domain, country, headline, achievement, summary, chapterTS, targetDur)
	if !ai.HasGeminiKey(ctx) {
		return fallback
	}

	bioInput := bio
	if len(bioInput) > 4500 {
		bioInput = bioInput[:4500]
	}
	aiRes, err := ai.GenerateVideoPackagingWithGemini(
		ctx, personName, domain.Label, country, headline, achievement, bioInput, chapterTS, targetDur,
	)
	if err != nil || aiRes == nil {
		return fallback
	}

	out := fallback
	out.UsedGemini = true
	if strings.TrimSpace(aiRes.VideoDescription) != "" {
		out.VideoDescription = strings.TrimSpace(aiRes.VideoDescription)
	}
	if len(aiRes.RecommendedTitles) > 0 {
		out.RecommendedTitles = aiRes.RecommendedTitles
	}
	if len(aiRes.Tags) > 0 {
		if norm := NormalizeTags(aiRes.Tags, 480); len(norm) > 0 {
			out.Tags = norm
		}
	}
	if len(aiRes.Hashtags) > 0 {
		if norm := NormalizeHashtags(aiRes.Hashtags, 12); len(norm) > 0 {
			out.Hashtags = norm
		}
	}
	out.TagsCharCount = TagsTotalChars(out.Tags)
	return out
}
