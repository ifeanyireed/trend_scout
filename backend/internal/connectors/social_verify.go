package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// VerifiedSocialAccount represents a genuinely verified official social media presence
// for the documentary subject or a notable individual/organization/corporation connected to the biography.
type VerifiedSocialAccount struct {
	EntityName      string `json:"entityName"`
	EntityType      string `json:"entityType"`              // "subject", "person", "organization"
	RelationGroup   string `json:"relationGroup,omitempty"` // subject | family_influential | coaches_mentors | teams_organizations | governing_bodies | competitors_contemporaries | media_journalists
	RelationRole    string `json:"relationRole,omitempty"`  // Human-readable role e.g. "Coach / Mentor", "Governing Body", "Rival"
	WikidataID      string `json:"wikidataId,omitempty"`
	YouTubeHandle   string `json:"youtubeHandle,omitempty"`
	YouTubeURL      string `json:"youtubeUrl,omitempty"`
	TwitterHandle   string `json:"twitterHandle,omitempty"`
	InstagramHandle string `json:"instagramHandle,omitempty"`
	VerifiedBy      string `json:"verifiedBy"`
}

// ConnectedEntityTarget represents a researched person or organization connected to the subject's biography
// across all 7 relationship categories prior to official account verification.
type ConnectedEntityTarget struct {
	Name          string
	RelationGroup string // subject | family_influential | coaches_mentors | teams_organizations | governing_bodies | competitors_contemporaries | media_journalists
	RelationRole  string
}

var fanOrImpersonatorTokens = []string{
	"fanpage", "fan_page", "fanclub", "fan_club", "fans", "fanacct", "fanaccount",
	"parody", "unofficial", "notreal", "impersonat", "tribute", "stan",
	"daily", "updates", "edits", "clips", "shorts", "backup", "lovers",
	"army", "fc_", "_fc", "hq_fan", "fake", "meme", "memes", "news_fan",
}

var validSocialHandleRe = regexp.MustCompile(`^[a-zA-Z0-9._-]{2,40}$`)
var ytCanonicalHandleRe = regexp.MustCompile(`"canonicalBaseUrl"\s*:\s*"/(@[a-zA-Z0-9._-]{2,40})"`)
var properEntityRe = regexp.MustCompile(`\b([A-Z][a-zA-Z0-9&'.-]+(?:\s+(?:[A-Z][a-zA-Z0-9&'.-]+|of|the|for|and|in)){0,3})\b`)

// IsLikelyFanOrImpersonatorHandle returns true if a handle exhibits common fan-page, parody,
// clip-reuploader, or impersonator patterns.
func IsLikelyFanOrImpersonatorHandle(handle string) bool {
	h := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(handle), "@"))
	if h == "" || !validSocialHandleRe.MatchString(h) {
		return true
	}
	for _, tok := range fanOrImpersonatorTokens {
		if strings.Contains(h, tok) {
			return true
		}
	}
	if strings.HasSuffix(h, "fan") || strings.HasPrefix(h, "fan") && len(h) > 5 {
		return true
	}
	return false
}

// ExtractCandidateEntities pulls notable organizations, federations, companies, and named individuals
// from the biography & dossier text so they can be officially verified via Wikidata.
func ExtractCandidateEntities(personName, biography, headline, achievement string, extra []string) []string {
	stopWords := map[string]bool{
		"The": true, "This": true, "That": true, "These": true, "Those": true, "When": true, "Where": true,
		"While": true, "After": true, "Before": true, "During": true, "Since": true, "Until": true, "Under": true,
		"January": true, "February": true, "March": true, "April": true, "May": true, "June": true, "July": true,
		"August": true, "September": true, "October": true, "November": true, "December": true,
		"Monday": true, "Tuesday": true, "Wednesday": true, "Thursday": true, "Friday": true, "Saturday": true, "Sunday": true,
		"In": true, "On": true, "At": true, "By": true, "For": true, "From": true, "With": true, "Into": true, "His": true, "Her": true, "Their": true,
		"He": true, "She": true, "They": true, "It": true, "But": true, "And": true, "However": true, "Despite": true, "According": true,
		"Gold": true, "Silver": true, "Bronze": true, "Medal": true, "World": true, "National": true, "International": true,
		"Chapter": true, "Biography": true, "Documentary": true, "Sources": true, "Wikipedia": true,
	}

	seen := map[string]bool{}
	var out []string
	addEntity := func(name string) {
		clean := strings.TrimSpace(strings.Trim(name, ".,;:\"'()[]"))
		if len(clean) < 3 || len(clean) > 48 {
			return
		}
		if stopWords[clean] {
			return
		}
		low := strings.ToLower(clean)
		if seen[low] {
			return
		}
		if personName != "" && low == strings.ToLower(strings.TrimSpace(personName)) {
			return
		}
		seen[low] = true
		out = append(out, clean)
	}

	for _, e := range extra {
		addEntity(e)
	}

	// Score multi-word proper nouns / institutions from headline, achievement, and biography
	corpus := headline + "\n" + achievement + "\n" + biography
	counts := map[string]int{}
	display := map[string]string{}
	matches := properEntityRe.FindAllStringSubmatch(corpus, -1)
	for _, m := range matches {
		cand := strings.TrimSpace(m[1])
		// Skip year-prefixed historical events (e.g. "2018 Asian Games") — prefer living organizations, federations & people
		if len(cand) >= 4 && cand[0] >= '0' && cand[0] <= '9' {
			continue
		}
		words := strings.Fields(cand)
		if len(words) == 0 || stopWords[words[0]] {
			continue
		}
		// Prefer multi-word names (organizations, federations, notable people) or well-known acronyms (NASA, FIFA, NBA, UEFA, IBM, IOC, ATP, WTA, FIDE, UFC)
		isAcronym := len(words) == 1 && len(cand) >= 3 && len(cand) <= 6 && strings.ToUpper(cand) == cand
		if len(words) < 2 && !isAcronym {
			continue
		}
		if len(words) > 0 && (words[len(words)-1] == "of" || words[len(words)-1] == "the" || words[len(words)-1] == "and" || words[len(words)-1] == "in" || words[len(words)-1] == "for") {
			continue
		}
		low := strings.ToLower(cand)
		if personName != "" && (low == strings.ToLower(personName) || strings.Contains(strings.ToLower(personName), low)) {
			continue
		}
		counts[low]++
		if display[low] == "" {
			display[low] = cand
		}
	}

	// Rank by frequency and pick top 7 candidate notable entities
	type kv struct {
		name  string
		score int
	}
	var ranked []kv
	for low, c := range counts {
		ranked = append(ranked, kv{name: display[low], score: c})
	}
	for i := 0; i < len(ranked); i++ {
		best := i
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].score > ranked[best].score {
				best = j
			}
		}
		ranked[i], ranked[best] = ranked[best], ranked[i]
	}
	for _, r := range ranked {
		addEntity(r.name)
		if len(out) >= 7 {
			break
		}
	}
	return out
}

// VerifyOfficialSocialAccounts verifies official social accounts for the subject and a plain list of entity names.
func VerifyOfficialSocialAccounts(
	ctx context.Context,
	personName string,
	notableEntities []string,
	youtubeAPIKey string,
) []VerifiedSocialAccount {
	var targets []ConnectedEntityTarget
	if strings.TrimSpace(personName) != "" {
		targets = append(targets, ConnectedEntityTarget{
			Name:          strings.TrimSpace(personName),
			RelationGroup: "subject",
			RelationRole:  "Documentary Subject",
		})
	}
	for _, e := range notableEntities {
		if strings.TrimSpace(e) != "" {
			targets = append(targets, ConnectedEntityTarget{
				Name:          strings.TrimSpace(e),
				RelationGroup: "teams_organizations",
				RelationRole:  "Connected Entity",
			})
		}
	}
	return VerifyConnectedEntitiesAccounts(ctx, targets, youtubeAPIKey)
}

// VerifyConnectedEntitiesAccounts queries Wikipedia/Wikidata's curated official identity registry
// (P2002 X/Twitter, P2003 Instagram, P11245 YouTube handle, P2397 YouTube channel ID) plus YouTube Data API
// across all 7 relationship categories (subject, family/influential, coaches/mentors, teams/orgs,
// governing bodies, competitors/contemporaries, journalists/media) — strictly rejecting fan pages & impersonators.
func VerifyConnectedEntitiesAccounts(
	ctx context.Context,
	rawTargets []ConnectedEntityTarget,
	youtubeAPIKey string,
) []VerifiedSocialAccount {
	var targets []ConnectedEntityTarget
	seen := map[string]bool{}
	for _, t := range rawTargets {
		clean := strings.TrimSpace(t.Name)
		if clean == "" || seen[strings.ToLower(clean)] {
			continue
		}
		seen[strings.ToLower(clean)] = true
		if t.RelationGroup == "" {
			t.RelationGroup = "teams_organizations"
		}
		t.Name = clean
		targets = append(targets, t)
		if len(targets) >= 14 {
			break
		}
	}
	if len(targets) == 0 {
		return nil
	}

	vCtx, cancel := context.WithTimeout(ctx, 11*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	results := make([]*VerifiedSocialAccount, len(targets))

	// Bound concurrency to 5 so Wikimedia APIs never throttle concurrent verification lookups
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for i, tgt := range targets {
		wg.Add(1)
		go func(idx int, t ConnectedEntityTarget) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			defType := "notable"
			if t.RelationGroup == "subject" {
				defType = "subject"
			}
			acc := lookupWikidataOfficialAccounts(vCtx, client, t.Name, defType, youtubeAPIKey)
			if acc != nil {
				acc.RelationGroup = t.RelationGroup
				if strings.TrimSpace(t.RelationRole) != "" {
					acc.RelationRole = strings.TrimSpace(t.RelationRole)
				}
				results[idx] = acc
			}
		}(i, tgt)
	}
	wg.Wait()

	var verified []VerifiedSocialAccount
	seenQID := map[string]bool{}
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.WikidataID != "" && seenQID[r.WikidataID] {
			continue
		}
		if r.YouTubeHandle == "" && r.TwitterHandle == "" && r.InstagramHandle == "" {
			continue
		}
		if r.WikidataID != "" {
			seenQID[r.WikidataID] = true
		}
		verified = append(verified, *r)
	}
	return verified
}

func lookupWikipediaEntityInfo(ctx context.Context, client *http.Client, entityName string) (qid string, title string, tw string, ig string, ytHandle string, ytChanID string) {
	wpURL := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&redirects=1&prop=pageprops|extlinks&ppprop=wikibase_item&ellimit=60&titles=%s&format=json",
		url.QueryEscape(entityName),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wpURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", WikiUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 500_000))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		return
	}
	var wpRes struct {
		Query struct {
			Pages map[string]struct {
				Title     string `json:"title"`
				PageProps struct {
					WikibaseItem string `json:"wikibase_item"`
				} `json:"pageprops"`
				ExtLinks []struct {
					URL  string `json:"url"`
					Star string `json:"*"`
				} `json:"extlinks"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &wpRes); err != nil {
		return
	}
	for _, p := range wpRes.Query.Pages {
		if p.PageProps.WikibaseItem != "" {
			qid = p.PageProps.WikibaseItem
			title = p.Title
		}
		for _, el := range p.ExtLinks {
			u := strings.TrimSpace(el.URL)
			if u == "" {
				u = strings.TrimSpace(el.Star)
			}
			low := strings.ToLower(u)
			switch {
			case tw == "" && (strings.Contains(low, "twitter.com/") || strings.Contains(low, "x.com/")):
				if parsed, errU := url.Parse(u); errU == nil {
					seg := strings.Trim(parsed.Path, "/")
					if idx := strings.Index(seg, "/"); idx > 0 {
						seg = seg[:idx]
					}
					segLow := strings.ToLower(seg)
					if seg != "" && segLow != "intent" && segLow != "share" && segLow != "status" && segLow != "search" && segLow != "hashtag" && !strings.Contains(low, "/status/") {
						tw = seg
					}
				}
			case ig == "" && strings.Contains(low, "instagram.com/"):
				if parsed, errU := url.Parse(u); errU == nil {
					seg := strings.Trim(parsed.Path, "/")
					if idx := strings.Index(seg, "/"); idx > 0 {
						seg = seg[:idx]
					}
					segLow := strings.ToLower(seg)
					if seg != "" && segLow != "p" && segLow != "reel" && segLow != "explore" && segLow != "tv" {
						ig = seg
					}
				}
			case ytHandle == "" && strings.Contains(low, "youtube.com/@"):
				if idx := strings.Index(low, "youtube.com/@"); idx >= 0 {
					rest := u[idx+len("youtube.com/"):]
					if cut := strings.IndexAny(rest, "/?"); cut > 0 {
						rest = rest[:cut]
					}
					ytHandle = rest
				}
			case ytChanID == "" && strings.Contains(low, "youtube.com/channel/uc"):
				if idx := strings.Index(low, "youtube.com/channel/"); idx >= 0 {
					rest := u[idx+len("youtube.com/channel/"):]
					if cut := strings.IndexAny(rest, "/?"); cut > 0 {
						rest = rest[:cut]
					}
					if strings.HasPrefix(rest, "UC") {
						ytChanID = rest
					}
				}
			}
		}
	}
	return
}

func lookupWikidataOfficialAccounts(
	ctx context.Context,
	client *http.Client,
	entityName string,
	defaultType string,
	youtubeAPIKey string,
) *VerifiedSocialAccount {
	// Step 1: Resolve exact authoritative Wikidata Q-ID & official external links via English Wikipedia first
	qid, wpTitle, wpTW, wpIG, wpYTHandle, wpYTChanID := lookupWikipediaEntityInfo(ctx, client, entityName)
	label := entityName
	if wpTitle != "" {
		label = wpTitle
	}
	desc := ""

	// Fallback to Wikidata search if Wikipedia title lookup didn't yield a Q-ID
	if qid == "" {
		searchURL := fmt.Sprintf(
			"https://www.wikidata.org/w/api.php?action=wbsearchentities&search=%s&language=en&limit=3&format=json",
			url.QueryEscape(entityName),
		)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
		if err != nil {
			return nil
		}
		req.Header.Set("User-Agent", WikiUserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 256_000))
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			return nil
		}

		var searchRes struct {
			Search []struct {
				ID          string `json:"id"`
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"search"`
		}
		if err := json.Unmarshal(body, &searchRes); err != nil || len(searchRes.Search) == 0 {
			return nil
		}

		for _, item := range searchRes.Search {
			dLow := strings.ToLower(item.Description)
			if strings.Contains(dLow, "disambiguation") || strings.Contains(dLow, "wikimedia") {
				continue
			}
			qid = item.ID
			if strings.TrimSpace(item.Label) != "" {
				label = item.Label
			}
			desc = item.Description
			break
		}
	}
	if qid == "" {
		return nil
	}

	// Step 2: Fetch verified official social media claims (P31, P2002, P2003, P11245, P2397)
	entURL := fmt.Sprintf(
		"https://www.wikidata.org/w/api.php?action=wbgetentities&ids=%s&props=claims&format=json",
		url.QueryEscape(qid),
	)
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, entURL, nil)
	if err != nil {
		return nil
	}
	req2.Header.Set("User-Agent", WikiUserAgent)
	resp2, err := client.Do(req2)
	if err != nil {
		return nil
	}
	body2, err := io.ReadAll(io.LimitReader(resp2.Body, 1_500_000))
	resp2.Body.Close()
	if err != nil || resp2.StatusCode != 200 {
		return nil
	}

	var entRes struct {
		Entities map[string]struct {
			Claims map[string][]struct {
				Rank      string `json:"rank"`
				MainSnak  struct {
					DataValue *struct {
						Value any `json:"value"`
					} `json:"datavalue"`
				} `json:"mainsnak"`
			} `json:"claims"`
		} `json:"entities"`
	}
	if err := json.Unmarshal(body2, &entRes); err != nil {
		return nil
	}
	entity, ok := entRes.Entities[qid]
	if !ok {
		return nil
	}

	extractClaimStr := func(prop string) string {
		claims := entity.Claims[prop]
		if len(claims) == 0 {
			return ""
		}
		// Prefer "preferred" rank over "normal", skip "deprecated"
		best := ""
		for _, c := range claims {
			if c.Rank == "deprecated" || c.MainSnak.DataValue == nil {
				continue
			}
			val, ok := c.MainSnak.DataValue.Value.(string)
			if !ok || strings.TrimSpace(val) == "" {
				continue
			}
			if c.Rank == "preferred" {
				return strings.TrimSpace(val)
			}
			if best == "" {
				best = strings.TrimSpace(val)
			}
		}
		return best
	}

	// Determine entity type (subject / person / organization)
	entType := defaultType
	if entType != "subject" {
		entType = "organization"
		for _, c := range entity.Claims["P31"] {
			if c.MainSnak.DataValue != nil {
				if m, ok := c.MainSnak.DataValue.Value.(map[string]any); ok {
					if m["id"] == "Q5" { // Q5 = human
						entType = "person"
						break
					}
				}
			}
		}
		if entType == "organization" && (strings.Contains(strings.ToLower(desc), "athlete") ||
			strings.Contains(strings.ToLower(desc), "player") ||
			strings.Contains(strings.ToLower(desc), "coach") ||
			strings.Contains(strings.ToLower(desc), "founder") ||
			strings.Contains(strings.ToLower(desc), "actor") ||
			strings.Contains(strings.ToLower(desc), "politician")) {
			entType = "person"
		}
	}

	twRaw := extractClaimStr("P2002")     // X / Twitter username
	if twRaw == "" {
		twRaw = wpTW
	}
	igRaw := extractClaimStr("P2003")     // Instagram username
	if igRaw == "" {
		igRaw = wpIG
	}
	ytHandleRaw := extractClaimStr("P11245") // YouTube handle
	if ytHandleRaw == "" {
		ytHandleRaw = wpYTHandle
	}
	ytChanID := extractClaimStr("P2397")  // YouTube channel ID (UC...)
	if ytChanID == "" {
		ytChanID = wpYTChanID
	}

	acc := &VerifiedSocialAccount{
		EntityName: label,
		EntityType: entType,
		WikidataID: qid,
		VerifiedBy: "Wikidata Official Registry (" + qid + ")",
	}

	if twRaw != "" {
		clean := "@" + strings.TrimPrefix(twRaw, "@")
		if !IsLikelyFanOrImpersonatorHandle(clean) {
			acc.TwitterHandle = clean
		}
	}
	if igRaw != "" {
		clean := "@" + strings.TrimPrefix(igRaw, "@")
		if !IsLikelyFanOrImpersonatorHandle(clean) {
			acc.InstagramHandle = clean
		}
	}
	if ytHandleRaw != "" {
		clean := "@" + strings.TrimPrefix(ytHandleRaw, "@")
		if !IsLikelyFanOrImpersonatorHandle(clean) {
			acc.YouTubeHandle = clean
			acc.YouTubeURL = "https://www.youtube.com/" + clean
		}
	} else if ytChanID != "" {
		resolved := resolveOfficialYouTubeHandle(ctx, client, ytChanID, youtubeAPIKey)
		if resolved != "" && !IsLikelyFanOrImpersonatorHandle(resolved) {
			acc.YouTubeHandle = resolved
			acc.YouTubeURL = "https://www.youtube.com/" + resolved
			acc.VerifiedBy = "Wikidata (" + qid + ") + YouTube Official Channel"
		}
	}

	if acc.YouTubeHandle == "" && acc.TwitterHandle == "" && acc.InstagramHandle == "" {
		return nil
	}
	return acc
}

func resolveOfficialYouTubeHandle(ctx context.Context, client *http.Client, channelID, youtubeAPIKey string) string {
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return ""
	}
	// Primary: YouTube Data API v3 channels?part=snippet&id=UC... (1 quota unit, official customUrl)
	if key := strings.TrimSpace(youtubeAPIKey); key != "" {
		apiURL := fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/channels?part=snippet&id=%s&key=%s",
			url.QueryEscape(channelID), url.QueryEscape(key),
		)
		if req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil); err == nil {
			if resp, errDo := client.Do(req); errDo == nil {
				var data struct {
					Items []struct {
						Snippet struct {
							CustomURL string `json:"customUrl"`
						} `json:"snippet"`
					} `json:"items"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&data)
				resp.Body.Close()
				if len(data.Items) > 0 && strings.TrimSpace(data.Items[0].Snippet.CustomURL) != "" {
					cu := strings.TrimSpace(data.Items[0].Snippet.CustomURL)
					if !strings.HasPrefix(cu, "@") {
						cu = "@" + cu
					}
					return cu
				}
			}
		}
	}

	// Fallback: Fetch public YouTube channel page canonicalBaseUrl (0 quota units)
	chURL := "https://www.youtube.com/channel/" + url.PathEscape(channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 500_000))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	if m := ytCanonicalHandleRe.FindStringSubmatch(string(body)); len(m) > 1 {
		return m[1]
	}
	return ""
}

// FormatVerifiedAccountsForPrompt formats the verified official account registry into strict
// platform-separated grounding blocks so YouTube, X, and Instagram handles are never mixed up.
func FormatVerifiedAccountsForPrompt(accounts []VerifiedSocialAccount) string {
	if len(accounts) == 0 {
		return "No pre-verified Wikidata handles returned for this entity set. ONLY tag official, well-known verified accounts you are 100% certain are authentic official accounts (NEVER fan pages, parody accounts, clip channels, or impersonators)."
	}
	var sb strings.Builder
	sb.WriteString("VERIFIED OFFICIAL ACCOUNTS REGISTRY (Verified via Wikipedia/Wikidata Official Claims & YouTube Official Channel API — ZERO fan pages or impersonators):\n")
	sb.WriteString("IMPORTANT: Use ONLY YouTube handles in YouTube description, ONLY X handles in X caption, and ONLY Instagram handles in Instagram caption — NEVER cross-contaminate platform handles!\n")
	for _, a := range accounts {
		var parts []string
		if a.YouTubeHandle != "" {
			parts = append(parts, "YouTube="+a.YouTubeHandle)
		}
		if a.TwitterHandle != "" {
			parts = append(parts, "X="+a.TwitterHandle)
		}
		if a.InstagramHandle != "" {
			parts = append(parts, "Instagram="+a.InstagramHandle)
		}
		roleTag := a.EntityType
		if a.RelationRole != "" {
			roleTag = a.RelationRole
		}
		sb.WriteString(fmt.Sprintf("- %s (%s) [%s] -> %s\n", a.EntityName, roleTag, a.VerifiedBy, strings.Join(parts, " | ")))
	}
	return strings.TrimSpace(sb.String())
}
