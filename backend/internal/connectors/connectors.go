package connectors

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) TrendScoutStudio/2.0"
const WikiUserAgent = "TrendScoutStudio/2.0 (https://trendscout.studio; research@trendscout.studio)"

type VideoMetric struct {
	Title                string  `json:"title"`
	Views                int64   `json:"views"`
	Likes                int64   `json:"likes"`
	Comments             int64   `json:"comments"`
	DurationSeconds      int     `json:"durationSeconds"`
	VideoFormat          string  `json:"videoFormat"`
	EngagementRate       float64 `json:"engagementRate"`
	ThumbnailURL         string  `json:"thumbnailUrl"`
	ChannelID            string  `json:"channelId,omitempty"`
	ChannelName          string  `json:"channelName"`
	ChannelSubscribers   int64   `json:"channelSubscribers"`
	ChannelBaselineViews float64 `json:"channelBaselineViews,omitempty"`
	CompetingVideosCount int     `json:"competingVideosCount,omitempty"`
	VideoURL             string  `json:"videoUrl"`
	ExternalID           string  `json:"externalId"`
	PublishedAt          string  `json:"publishedAt"`
	ViewsPerHour         float64 `json:"viewsPerHour"`
	ViewsPerDay          float64 `json:"viewsPerDay"`
	ChannelMultiplier    float64 `json:"channelMultiplier"`
}

type SocialMetric struct {
	Platform    string  `json:"platform"`
	ExternalID  string  `json:"externalId"`
	Author      string  `json:"author"`
	Subreddit   string  `json:"subreddit"`
	Content     string  `json:"content"`
	URL         string  `json:"url"`
	Upvotes     int64   `json:"upvotes"`
	Comments    int64   `json:"comments"`
	Shares      int64   `json:"shares"`
	AgeHours    float64 `json:"ageHours,omitempty"`
	PublishedAt string  `json:"publishedAt"`
}

type DiscoveredCandidate struct {
	SourceID      string        `json:"sourceId"`
	PersonName    string        `json:"personName"`
	Aliases       []string      `json:"aliases"`
	Category      string        `json:"category"`
	Country       string        `json:"country"`
	EventTitle    string        `json:"eventTitle"`
	EventDate     string        `json:"eventDate"`
	Achievement   string        `json:"achievement"`
	Headline      string        `json:"headline"`
	SourceURL     string        `json:"sourceUrl"`
	Publisher     string        `json:"publisher"`
	Platform      string        `json:"platform"`
	Summary       string        `json:"summary"`
	ImageURL      string        `json:"imageUrl"`
	WikiExtract   string        `json:"wikiExtract"`
	SearchTraffic int64         `json:"searchTraffic,omitempty"`
	Video         *VideoMetric  `json:"video,omitempty"`
	Social        *SocialMetric `json:"social,omitempty"`
}

type WikiSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type WikiDeepProfile struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Summary     string        `json:"summary"`
	ImageURL    string        `json:"imageUrl"`
	PageURL     string        `json:"pageUrl"`
	Country     string        `json:"country"`
	Category    string        `json:"category"`
	IsHuman     bool          `json:"isHuman"`
	Sections    []WikiSection `json:"sections"`
}

var nameRegex = regexp.MustCompile(`\b([A-Z][a-z]{2,15}(?:\s+[A-Z][a-z]{2,15}){1,2})\b`)

var stopWords = map[string]bool{
	"The": true, "How": true, "Why": true, "What": true, "When": true, "Where": true,
	"Who": true, "After": true, "Before": true, "Inside": true, "World": true, "Olympic": true,
	"Olympics": true, "Grand": true, "Slam": true, "Open": true, "Cup": true, "League": true,
	"Champion": true, "Championship": true, "Record": true, "History": true, "Breaking": true,
	"News": true, "Report": true, "Watch": true, "Live": true, "United": true, "States": true,
	"New": true, "York": true, "South": true, "North": true, "East": true, "West": true,
	"First": true, "Final": true, "Gold": true, "Medal": true, "Team": true, "Coach": true,
	"National": true, "International": true, "Global": true, "Sports": true, "Daily": true,
	"Times": true, "Post": true, "Guardian": true, "Reuters": true, "Associated": true,
	"Press": true, "Winning": true, "Paris": true, "London": true, "Tokyo": true, "Los": true,
	"Angeles": true, "San": true, "Francisco": true, "Real": true, "Madrid": true, "Manchester": true,
	"City": true, "Club": true, "Premier": true, "Series": true, "Tour": true, "Games": true,
	"Super": true, "Bowl": true, "Playoffs": true, "Round": true, "Stage": true, "Match": true,
	"Game": true, "Season": true, "Year": true, "Old": true, "Youngest": true, "Ever": true,
	"Wins": true, "Beats": true, "Defeats": true, "Breaks": true, "Makes": true, "Takes": true,
	"Sets": true, "From": true, "With": true, "Without": true, "Over": true, "Under": true,
	"Against": true, "All": true, "Odds": true, "For": true, "And": true, "But": true,
	"Not": true, "This": true, "That": true, "These": true, "Those": true, "His": true,
	"Her": true, "Their": true, "Our": true, "Your": true, "Monday": true, "Tuesday": true,
	"Wednesday": true, "Thursday": true, "Friday": true, "Saturday": true, "Sunday": true,
	"January": true, "February": true, "March": true, "April": true, "May": true, "June": true,
	"July": true, "August": true, "September": true, "October": true, "November": true, "December": true,
	"ESPN": true, "BBC": true, "CNN": true, "NBC": true, "CBS": true, "Fox": true, "Sky": true,
	"Yahoo": true, "Bleacher": true, "Athletic": true, "SportsCenter": true, "Red": true, "Bull": true,
}

// ExtractCandidateNamesFromText returns potential 2-3 word proper names from a headline.
func ExtractCandidateNamesFromText(headline string) []string {
	matches := nameRegex.FindAllStringSubmatch(headline, -1)
	var out []string
	seen := map[string]bool{}
	for _, m := range matches {
		candidate := strings.TrimSpace(m[1])
		parts := strings.Fields(candidate)
		if len(parts) < 2 || len(parts) > 3 {
			continue
		}
		valid := true
		for _, p := range parts {
			if stopWords[p] {
				valid = false
				break
			}
		}
		if valid && !seen[candidate] {
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func ExtractPersonFromHeadline(headline string) string {
	list := ExtractCandidateNamesFromText(headline)
	if len(list) > 0 {
		return list[0]
	}
	return ""
}

func isLikelyHumanBio(title, desc, extract string) bool {
	titleTrim := strings.TrimSpace(title)
	if titleTrim == "" || (titleTrim[0] >= '0' && titleTrim[0] <= '9') {
		return false
	}
	titleLower := strings.ToLower(titleTrim)
	nonHumanTitleWords := []string{
		"university", "college", "festival", "championship", "tournament", "league",
		"association", "federation", "organization", "organisation", "committee",
		"conference", "museum", "stadium", "arena", "olympics", "paralympics",
		"games", "series", "season", "award", "trophy", "cup", "club", "team",
		"fc", "f.c.", "sc", "united", "national", "ministry", "department", "party",
		"list of", "history of", "records", "medal", "event", "match", "finals",
		" open", "grand prix", "marathon", "classic", "masters", "tour",
	}
	for _, w := range nonHumanTitleWords {
		if strings.Contains(titleLower, w) {
			return false
		}
	}

	combined := strings.ToLower(desc + " " + extract)
	nonHumanMarkers := []string{
		"is a professional sports team",
		"is a football club",
		"is a basketball team",
		"is a baseball team",
		"is a public ",
		"is a private ",
		"is a university",
		"is a college",
		"is an annual",
		"is a tournament",
		"is a championship",
		"is a competition",
		"is a festival",
		"is a stadium",
		"is a city in",
		"is a town in",
		"is a country in",
		"is a state in",
		"is a province",
		"is a television series",
		"is a video game",
		"is a company",
		"is a corporation",
		"is a brand",
		"is a film",
		"is an album",
		"is a song",
		"is a novel",
		"is a book",
		"is an organization",
		"is an organisation",
		"may refer to",
		"disambiguation",
	}
	for _, bad := range nonHumanMarkers {
		if strings.Contains(combined, bad) {
			return false
		}
	}

	// Strong biographical proof: birth date/year or explicit individual profession across all domains
	hasBirthMarker := strings.Contains(combined, "(born ") || strings.Contains(combined, "born ")
	humanProfessions := []string{
		// Science, Medicine & Tech
		"scientist", "inventor", "researcher", "mathematician", "physicist", "chemist",
		"biologist", "neuroscientist", "astronomer", "engineer", "programmer", "computer scientist",
		"physician", "doctor", "surgeon", "professor", "educator", "economist", "architect",
		// Business & Founders
		"entrepreneur", "founder", "co-founder", "businessperson", "businessman", "businesswoman",
		"executive", "philanthropist", "investor", "industrialist",
		// Arts, Film, Music & Literature
		"musician", "singer", "songwriter", "composer", "pianist", "violinist", "rapper",
		"actor", "actress", "filmmaker", "director", "screenwriter", "producer",
		"author", "novelist", "writer", "poet", "journalist", "painter", "sculptor", "designer", "chef",
		// Humanitarian, Exploration & Civic
		"activist", "humanitarian", "explorer", "astronaut", "mountaineer", "pilot", "advocate",
		// Sports & Mind Sports
		"player", "athlete", "sprinter", "runner", "swimmer", "cyclist", "gymnast",
		"boxer", "fighter", "wrestler", "grandmaster", "chess", "skater", "climber",
		"thrower", "jumper", "medalist", "olympian", "paralympian", "competitor",
		"driver", "racer", "golfer", "tennis", "footballer", "cricketer", "judoka",
		"fencer", "skier", "diver", "archer", "rower", "weightlifter", "surfer",
		"coach", "badminton", "taekwondo", "javelin", "marathon",
	}
	for _, prof := range humanProfessions {
		if strings.Contains(combined, prof) {
			return true
		}
	}
	return hasBirthMarker
}

func inferCountryAndCategory(desc, extract string) (string, string) {
	combined := desc + " " + extract
	lower := strings.ToLower(combined)

	category := "Breakthrough & Human Interest"
	switch {
	case strings.Contains(lower, "physician") || strings.Contains(lower, "surgeon") || strings.Contains(lower, "biologist") || strings.Contains(lower, "chemist") || strings.Contains(lower, "physicist") || strings.Contains(lower, "scientist") || strings.Contains(lower, "neuroscientist") || strings.Contains(lower, "mathematician") || strings.Contains(lower, "nobel prize in"):
		category = "Science & Medicine"
	case strings.Contains(lower, "computer scientist") || strings.Contains(lower, "programmer") || strings.Contains(lower, "software") || strings.Contains(lower, "inventor") || strings.Contains(lower, "engineer") || strings.Contains(lower, "artificial intelligence"):
		category = "Technology & Innovation"
	case strings.Contains(lower, "entrepreneur") || strings.Contains(lower, "founder") || strings.Contains(lower, "businessperson") || strings.Contains(lower, "businessman") || strings.Contains(lower, "businesswoman") || strings.Contains(lower, "ceo") || strings.Contains(lower, "startup"):
		category = "Business & Founders"
	case strings.Contains(lower, "filmmaker") || strings.Contains(lower, "director") || strings.Contains(lower, "musician") || strings.Contains(lower, "singer") || strings.Contains(lower, "composer") || strings.Contains(lower, "actor") || strings.Contains(lower, "actress") || strings.Contains(lower, "novelist") || strings.Contains(lower, "author") || strings.Contains(lower, "painter") || strings.Contains(lower, "chef"):
		category = "Arts, Film & Music"
	case strings.Contains(lower, "activist") || strings.Contains(lower, "humanitarian") || strings.Contains(lower, "philanthropist") || strings.Contains(lower, "nobel peace"):
		category = "Humanitarian & Civic Impact"
	case strings.Contains(lower, "astronaut") || strings.Contains(lower, "explorer") || strings.Contains(lower, "mountaineer") || strings.Contains(lower, "pilot"):
		category = "Exploration & Survival"
	case strings.Contains(lower, "chess") || strings.Contains(lower, "grandmaster"):
		category = "Mind Sports & Chess"
	case strings.Contains(lower, "sprinter") || strings.Contains(lower, "hurdler") || strings.Contains(lower, "marathon") || strings.Contains(lower, "javelin") || strings.Contains(lower, "track and field") || strings.Contains(lower, "decathlon") || strings.Contains(lower, "jump"):
		category = "Track & Field Athletics"
	case strings.Contains(lower, "cyclist") || strings.Contains(lower, "tour de france"):
		category = "Pro Cycling & Endurance"
	case strings.Contains(lower, "tennis"):
		category = "Racquet & Court Sports"
	case strings.Contains(lower, "swimmer") || strings.Contains(lower, "swimming") || strings.Contains(lower, "diving"):
		category = "Aquatics & Swimming"
	case strings.Contains(lower, "gymnast"):
		category = "Artistic Gymnastics"
	case strings.Contains(lower, "boxer") || strings.Contains(lower, "martial") || strings.Contains(lower, "wrestler") || strings.Contains(lower, "judo") || strings.Contains(lower, "taekwondo"):
		category = "Combat Sports"
	case strings.Contains(lower, "athlete") || strings.Contains(lower, "olympian") || strings.Contains(lower, "paralympian") || strings.Contains(lower, "medalist") || strings.Contains(lower, "footballer") || strings.Contains(lower, "cricketer") || strings.Contains(lower, "golfer") || strings.Contains(lower, "skater"):
		category = "Sports & Competition"
	}

	nationalityMap := map[string]string{
		"american": "United States", "british": "United Kingdom", "english": "United Kingdom",
		"scottish": "United Kingdom", "indian": "India", "canadian": "Canada",
		"australian": "Australia", "french": "France", "german": "Germany",
		"spanish": "Spain", "italian": "Italy", "japanese": "Japan",
		"chinese": "China", "south korean": "South Korea", "brazilian": "Brazil",
		"argentine": "Argentina", "mexican": "Mexico", "nigerian": "Nigeria",
		"kenyan": "Kenya", "ethiopian": "Ethiopia", "ugandan": "Uganda",
		"south african": "South Africa", "botswanan": "Botswana", "motswana": "Botswana",
		"pakistani": "Pakistan", "saint lucian": "Saint Lucia", "jamaican": "Jamaica",
		"eritrean": "Eritrea", "dutch": "Netherlands", "swedish": "Sweden",
		"norwegian": "Norway", "polish": "Poland", "ukrainian": "Ukraine",
		"serbian": "Serbia", "croatian": "Croatia", "greek": "Greece",
		"turkish": "Turkey", "egyptian": "Egypt", "moroccan": "Morocco",
		"new zealand": "New Zealand", "irish": "Ireland", "swiss": "Switzerland",
		"belgian": "Belgium", "portuguese": "Portugal", "colombian": "Colombia",
		"cuban": "Cuba", "dominican": "Dominican Republic", "filipino": "Philippines",
		"indonesian": "Indonesia", "uzbek": "Uzbekistan", "kazakh": "Kazakhstan",
	}
	country := "Global"
	for adj, cname := range nationalityMap {
		if strings.Contains(lower, adj) {
			country = cname
			break
		}
	}
	return country, strings.TrimSpace(category)
}

func FetchWikipediaProfile(ctx context.Context, subject string) (*WikiDeepProfile, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	encoded := url.PathEscape(strings.ReplaceAll(strings.TrimSpace(subject), " ", "_"))
	sumURL := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/page/summary/%s", encoded)

	req, err := http.NewRequestWithContext(ctx, "GET", sumURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", WikiUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("wikipedia summary status %d", resp.StatusCode)
	}

	var sumData struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Extract     string `json:"extract"`
		Thumbnail   struct {
			Source string `json:"source"`
		} `json:"thumbnail"`
		OriginalImage struct {
			Source string `json:"source"`
		} `json:"originalimage"`
		ContentURLs struct {
			Desktop struct {
				Page string `json:"page"`
			} `json:"desktop"`
		} `json:"content_urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sumData); err != nil {
		return nil, err
	}
	if sumData.Type == "disambiguation" || len(strings.TrimSpace(sumData.Extract)) < 50 {
		return nil, fmt.Errorf("disambiguation or stub page")
	}

	isHuman := isLikelyHumanBio(sumData.Title, sumData.Description, sumData.Extract)
	country, category := inferCountryAndCategory(sumData.Description, sumData.Extract)

	pageURL := sumData.ContentURLs.Desktop.Page
	if pageURL == "" {
		pageURL = fmt.Sprintf("https://en.wikipedia.org/wiki/%s", encoded)
	}

	imgURL := sumData.Thumbnail.Source
	if imgURL == "" {
		imgURL = sumData.OriginalImage.Source
	}

	sections := []WikiSection{
		{Title: "Biographical Overview", Content: sumData.Extract},
	}

	apiURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&prop=extracts&explaintext=1&titles=%s&format=json", encoded)
	req2, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err == nil {
		req2.Header.Set("User-Agent", WikiUserAgent)
		if resp2, err2 := client.Do(req2); err2 == nil {
			defer resp2.Body.Close()
			var apiRes struct {
				Query struct {
					Pages map[string]struct {
						Extract string `json:"extract"`
					} `json:"pages"`
				} `json:"query"`
			}
			if json.NewDecoder(resp2.Body).Decode(&apiRes) == nil {
				for _, p := range apiRes.Query.Pages {
					if len(p.Extract) > 120 {
						parsed := parseWikiSections(p.Extract)
						if len(parsed) > 0 {
							sections = parsed
						}
					}
				}
			}
		}
	}

	return &WikiDeepProfile{
		Title:       sumData.Title,
		Description: sumData.Description,
		Summary:     sumData.Extract,
		ImageURL:    imgURL,
		PageURL:     pageURL,
		Country:     country,
		Category:    category,
		IsHuman:     isHuman,
		Sections:    sections,
	}, nil
}

var sectionHeaderRe = regexp.MustCompile(`(?m)^==+\s*([^=]+?)\s*==+\s*$`)

func parseWikiSections(fullText string) []WikiSection {
	indices := sectionHeaderRe.FindAllStringSubmatchIndex(fullText, -1)
	if len(indices) == 0 {
		return []WikiSection{{Title: "Biographical Overview", Content: strings.TrimSpace(fullText)}}
	}
	var out []WikiSection
	intro := strings.TrimSpace(fullText[:indices[0][0]])
	if len(intro) > 40 {
		out = append(out, WikiSection{Title: "Overview & Career Summary", Content: intro})
	}
	for i, idx := range indices {
		title := strings.TrimSpace(fullText[idx[2]:idx[3]])
		lower := strings.ToLower(title)
		if lower == "see also" || lower == "references" || lower == "external links" || lower == "notes" || lower == "further reading" || lower == "awards" {
			continue
		}
		start := idx[1]
		end := len(fullText)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}
		body := strings.TrimSpace(fullText[start:end])
		if len(body) > 80 {
			out = append(out, WikiSection{Title: title, Content: body})
		}
	}
	return out
}

// 1. GOOGLE NEWS LIVE RSS CONNECTOR
type rssFeed struct {
	Channel struct {
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
			Source  string `xml:"source"`
		} `xml:"item"`
	} `xml:"channel"`
}

func DiscoverGoogleNews(ctx context.Context, queries []string) []DiscoveredCandidate {
	type domainQuery struct {
		query    string
		category string
	}
	var dQueries []domainQuery
	if len(queries) > 0 {
		for _, q := range queries {
			dQueries = append(dQueries, domainQuery{query: q, category: "Breakthrough & Human Interest"})
		}
	} else {
		dQueries = []domainQuery{
			{query: `("breakthrough" OR "prize" OR "discovery") scientist OR inventor OR researcher when:7d`, category: "Science & Medicine"},
			{query: `("bootstrapped" OR "self-made" OR "against all odds") founder OR entrepreneur when:7d`, category: "Business & Founders"},
			{query: `("breakout" OR "debut" OR "historic") filmmaker OR musician OR author OR artist when:7d`, category: "Arts, Film & Music"},
			{query: `("youngest" OR "prodigy" OR "self-taught") inventor OR programmer OR engineer OR AI when:7d`, category: "Technology & Innovation"},
			{query: `("humanitarian" OR "activist" OR "survived" OR "first ever") hero OR explorer OR pioneer when:7d`, category: "Humanitarian & Civic Impact"},
			{query: `("underdog" OR "unseeded" OR "upset") champion athlete OR chess OR medalist when:7d`, category: "Sports & Competition"},
		}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	var (
		out []DiscoveredCandidate
		mu  sync.Mutex
		wg  sync.WaitGroup
	)

	for _, dq := range dQueries {
		wg.Add(1)
		go func(itemQuery domainQuery) {
			defer wg.Done()
			u := fmt.Sprintf("https://news.google.com/rss/search?q=%s&hl=en-US&gl=US&ceid=US:en", url.QueryEscape(itemQuery.query))
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", UserAgent)
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var feed rssFeed
			if err := xml.Unmarshal(body, &feed); err != nil {
				return
			}
			var local []DiscoveredCandidate
			for i, item := range feed.Channel.Items {
				if i >= 4 {
					break
				}
				rawTitle := strings.TrimSpace(item.Title)
				headline := rawTitle
				publisher := strings.TrimSpace(item.Source)
				if idx := strings.LastIndex(rawTitle, " - "); idx > 0 {
					headline = strings.TrimSpace(rawTitle[:idx])
					if publisher == "" {
						publisher = strings.TrimSpace(rawTitle[idx+3:])
					}
				}
				if publisher == "" {
					publisher = "Google News"
				}
				eventDate := time.Now().Format("2006-01-02")
				if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
					eventDate = t.Format("2006-01-02")
				} else if t2, err2 := time.Parse(time.RFC1123, item.PubDate); err2 == nil {
					eventDate = t2.Format("2006-01-02")
				}
				local = append(local, DiscoveredCandidate{
					SourceID:    "src-google-news",
					PersonName:  ExtractPersonFromHeadline(headline),
					Category:    itemQuery.category,
					Country:     "Global",
					EventTitle:  headline,
					EventDate:   eventDate,
					Achievement: headline,
					Headline:    headline,
					SourceURL:   item.Link,
					Publisher:   publisher,
					Platform:    "News",
					Summary:     fmt.Sprintf("Live coverage from %s: %s", publisher, headline),
				})
			}
			if len(local) > 0 {
				mu.Lock()
				out = append(out, local...)
				mu.Unlock()
			}
		}(dq)
	}
	wg.Wait()
	return out
}

// 2. MULTI-DOMAIN LIVE REDDIT CONNECTOR
func DiscoverRedditBreakouts(ctx context.Context) []DiscoveredCandidate {
	subreddits := []struct {
		name     string
		category string
	}{
		{"UpliftingNews", "Humanitarian & Civic Impact"},
		{"NextFuckingLevel", "Breakthrough & Human Interest"},
		{"science", "Science & Medicine"},
		{"technology", "Technology & Innovation"},
		{"Entrepreneur", "Business & Founders"},
		{"movies", "Arts, Film & Music"},
		{"sports", "Sports & Competition"},
		{"chess", "Mind Sports & Chess"},
	}
	client := &http.Client{Timeout: 5 * time.Second}
	var (
		out []DiscoveredCandidate
		mu  sync.Mutex
		wg  sync.WaitGroup
	)

	for _, sub := range subreddits {
		wg.Add(1)
		go func(subreddit, defaultCat string) {
			defer wg.Done()
			u := fmt.Sprintf("https://www.reddit.com/r/%s/top.json?t=week&limit=5", subreddit)
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", WikiUserAgent)
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}
			var payload struct {
				Data struct {
					Children []struct {
						Data struct {
							ID          string  `json:"id"`
							Title       string  `json:"title"`
							Author      string  `json:"author"`
							Subreddit   string  `json:"subreddit"`
							Permalink   string  `json:"permalink"`
							Ups         int64   `json:"ups"`
							NumComments int64   `json:"num_comments"`
							CreatedUTC  float64 `json:"created_utc"`
						} `json:"data"`
					} `json:"children"`
				} `json:"data"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&payload)
			resp.Body.Close()

			var local []DiscoveredCandidate
			for _, c := range payload.Data.Children {
				p := c.Data
				if p.Title == "" || p.Ups < 20 {
					continue
				}
				dt := time.Unix(int64(p.CreatedUTC), 0).Format("2006-01-02")
				ageHours := math.Max(1.0, time.Since(time.Unix(int64(p.CreatedUTC), 0)).Hours())
				postURL := fmt.Sprintf("https://www.reddit.com%s", p.Permalink)
				local = append(local, DiscoveredCandidate{
					SourceID:    "src-reddit",
					PersonName:  ExtractPersonFromHeadline(p.Title),
					Category:    defaultCat,
					Country:     "Global",
					EventTitle:  p.Title,
					EventDate:   dt,
					Achievement: p.Title,
					Headline:    p.Title,
					SourceURL:   postURL,
					Publisher:   fmt.Sprintf("Reddit r/%s", p.Subreddit),
					Platform:    "Reddit",
					Summary:     fmt.Sprintf("Live Reddit breakout thread on r/%s (%d upvotes, %d comments): %s", p.Subreddit, p.Ups, p.NumComments, p.Title),
					Social: &SocialMetric{
						Platform:    "Reddit",
						ExternalID:  "reddit-" + p.ID,
						Author:      "u/" + p.Author,
						Subreddit:   "r/" + p.Subreddit,
						Content:     p.Title,
						URL:         postURL,
						Upvotes:     p.Ups,
						Comments:    p.NumComments,
						Shares:      0,
						AgeHours:    ageHours,
						PublishedAt: dt,
					},
				})
			}
			if len(local) > 0 {
				mu.Lock()
				out = append(out, local...)
				mu.Unlock()
			}
		}(sub.name, sub.category)
	}
	wg.Wait()
	return out
}

// 3. GOOGLE TRENDS LIVE DAILY BREAKOUT RSS CONNECTOR
type trendsRssFeed struct {
	Channel struct {
		Items []struct {
			Title      string `xml:"title"`
			Link       string `xml:"link"`
			PubDate    string `xml:"pubDate"`
			Traffic    string `xml:"approx_traffic"`
			NewsTitle  string `xml:"news_item>news_item_title"`
			NewsURL    string `xml:"news_item>news_item_url"`
			NewsSource string `xml:"news_item>news_item_source"`
		} `xml:"item"`
	} `xml:"channel"`
}

func DiscoverGoogleTrends(ctx context.Context) []DiscoveredCandidate {
	geos := []string{"US", "GB", "IN", "CA", "AU"}
	client := &http.Client{Timeout: 5 * time.Second}
	var (
		out []DiscoveredCandidate
		mu  sync.Mutex
		wg  sync.WaitGroup
	)

	for _, geo := range geos {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			u := fmt.Sprintf("https://trends.google.com/trending/rss?geo=%s", region)
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", UserAgent)
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var feed trendsRssFeed
			if err := xml.Unmarshal(body, &feed); err != nil {
				return
			}
			var local []DiscoveredCandidate
			for i, item := range feed.Channel.Items {
				if i >= 6 {
					break
				}
				trendTitle := strings.TrimSpace(item.Title)
				newsTitle := strings.TrimSpace(item.NewsTitle)
				newsTitle = strings.ReplaceAll(newsTitle, "&#39;", "'")
				newsTitle = strings.ReplaceAll(newsTitle, "&quot;", `"`)
				if newsTitle == "" {
					newsTitle = fmt.Sprintf("Breakout search surge (%s searches) for %s", item.Traffic, trendTitle)
				}
				link := strings.TrimSpace(item.NewsURL)
				if link == "" {
					link = "https://trends.google.com/trending?geo=" + region
				}
				pub := strings.TrimSpace(item.NewsSource)
				if pub == "" {
					pub = fmt.Sprintf("Google Trends (%s • %s)", region, item.Traffic)
				} else {
					pub = fmt.Sprintf("%s (Google Trends %s)", pub, item.Traffic)
				}
				local = append(local, DiscoveredCandidate{
					SourceID:      "src-google-trends",
					PersonName:    trendTitle,
					Category:      "Global Breakout Trend",
					Country:       region,
					EventTitle:    newsTitle,
					EventDate:     time.Now().Format("2006-01-02"),
					Achievement:   newsTitle,
					Headline:      newsTitle,
					SourceURL:     link,
					Publisher:     pub,
					Platform:      "Google Trends",
					Summary:       fmt.Sprintf("Google Trends surge (%s searches in %s): %s", item.Traffic, region, newsTitle),
					SearchTraffic: ParseCompactCount(item.Traffic),
				})
			}
			if len(local) > 0 {
				mu.Lock()
				out = append(out, local...)
				mu.Unlock()
			}
		}(geo)
	}
	wg.Wait()
	return out
}

// 4. DYNAMIC MULTI-DOMAIN WIKIPEDIA SEARCH & BREAKOUT CONNECTOR (Zero Hardcoded Names, Rotating Across Domains)
func DiscoverWikipediaBreakouts(ctx context.Context, queries []string) []DiscoveredCandidate {
	if len(queries) == 0 {
		// Balanced multi-domain query pool spanning Science, Tech, Business, Arts, Humanitarian & Sports
		pool := []string{
			"self-taught scientist inventor breakthrough discovery biography",
			"bootstrapped founder entrepreneur overcame poverty biography",
			"independent filmmaker debut director award breakthrough biography",
			"humanitarian activist nobel laureate overcame adversity biography",
			"computer scientist open source creator prodigy biography",
			"unseeded champion paralympic underdog gold medalist biography",
			"physician surgeon medical pioneer breakthrough biography",
			"breakout musician composer debut album biography",
		}
		// Rotate starting slice & search offset over time so repeated scans discover fresh multi-domain subjects
		rot := int(time.Now().Unix()/60) % len(pool)
		queries = make([]string, 0, 5)
		for i := 0; i < 5; i++ {
			queries = append(queries, pool[(rot+i)%len(pool)])
		}
	}
	sroffset := int((time.Now().Unix() / 90) % 15)
	client := &http.Client{Timeout: 5 * time.Second}
	var (
		titles []string
		mu     sync.Mutex
		seen   = map[string]bool{}
		wg     sync.WaitGroup
	)

	for _, q := range queries {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			u := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&srlimit=4&sroffset=%d&format=json", url.QueryEscape(query), sroffset)
			req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", WikiUserAgent)
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}
			var searchRes struct {
				Query struct {
					Search []struct {
						Title   string `json:"title"`
						Snippet string `json:"snippet"`
					} `json:"search"`
				} `json:"query"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&searchRes)
			resp.Body.Close()

			mu.Lock()
			for _, hit := range searchRes.Query.Search {
				title := strings.TrimSpace(hit.Title)
				words := strings.Fields(title)
				if !seen[title] && len(words) >= 2 && len(words) <= 3 {
					seen[title] = true
					titles = append(titles, title)
				}
			}
			mu.Unlock()
		}(q)
	}
	wg.Wait()

	if len(titles) > 10 {
		titles = titles[:10]
	}

	var out []DiscoveredCandidate
	sem := make(chan struct{}, 4)
	for _, title := range titles {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			prof, err := FetchWikipediaProfile(ctx, t)
			if err != nil || prof == nil || !prof.IsHuman || prof.Summary == "" {
				return
			}
			firstSentence := strings.Split(prof.Summary, ". ")[0]
			mu.Lock()
			out = append(out, DiscoveredCandidate{
				SourceID:    "src-wikipedia",
				PersonName:  prof.Title,
				Category:    prof.Category,
				Country:     prof.Country,
				EventTitle:  fmt.Sprintf("%s — %s", prof.Title, prof.Description),
				EventDate:   time.Now().Format("2006-01-02"),
				Achievement: firstSentence,
				Headline:    firstSentence,
				SourceURL:   prof.PageURL,
				Publisher:   "Wikipedia Biographical Archive",
				Platform:    "Wikipedia",
				Summary:     prof.Summary,
				ImageURL:    prof.ImageURL,
				WikiExtract: prof.Summary,
			})
			mu.Unlock()
		}(title)
	}
	wg.Wait()
	return out
}

// 5. REAL YOUTUBE VIDEO, CHANNEL BASELINE & VELOCITY LOOKUP (YouTube Data API v3 + Public Channel Baseline Extraction)
var (
	ytVideoBlockRe     = regexp.MustCompile(`"videoRenderer":\{"videoId":"([a-zA-Z0-9_-]{11})".*?"title":\{"runs":\[\{"text":"([^"]+)"\}\]`)
	ytViewsRe          = regexp.MustCompile(`"viewCountText":\{"simpleText":"([0-9,\.]+[KMBkmb]?)\s+views?"\}`)
	ytShortViewsRe     = regexp.MustCompile(`"shortViewCountText":\{[^}]*"simpleText":"([0-9,\.]+[KMBkmb]?)\s+views?"`)
	ytPubTimeRe        = regexp.MustCompile(`"publishedTimeText":\{"simpleText":"([^"]+)"\}`)
	ytOwnerRe          = regexp.MustCompile(`"ownerText":\{"runs":\[\{"text":"([^"]+)"`)
	ytBrowseIDRe       = regexp.MustCompile(`"browseId":"(UC[a-zA-Z0-9_-]{20,28})"`)
	ytSubscriberRe     = regexp.MustCompile(`"subscriberCountText":\{[^}]*"(?:simpleText|text)":"([0-9,\.]+[KMBkmb]?)\s+subscribers?"`)
	ytLengthRe         = regexp.MustCompile(`"lengthText":\{[^}]*"simpleText":"(\d+:\d{2}(?::\d{2})?)"`)
	ytISO8601DurRe     = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)
	ytCompactNumRegexp = regexp.MustCompile(`([0-9]+(?:,[0-9]+)*(?:\.[0-9]+)?)\s*([KMBkmb])?`)
)

// ParseISO8601Duration parses YouTube API v3 ISO-8601 durations (e.g., "PT18M42S", "PT1H2M5S", "PT59S") into seconds.
func ParseISO8601Duration(raw string) int {
	m := ytISO8601DurRe.FindStringSubmatch(strings.TrimSpace(strings.ToUpper(raw)))
	if len(m) == 0 {
		return 0
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec, _ := strconv.Atoi(m[3])
	return h*3600 + min*60 + sec
}

// ParseClockDuration parses "18:42" or "1:02:15" into seconds.
func ParseClockDuration(raw string) int {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) == 2 {
		m, _ := strconv.Atoi(parts[0])
		s, _ := strconv.Atoi(parts[1])
		return m*60 + s
	}
	if len(parts) == 3 {
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		s, _ := strconv.Atoi(parts[2])
		return h*3600 + m*60 + s
	}
	return 0
}

// ClassifyVideoFormat classifies a video's duration into documentary-relevant formats.
func ClassifyVideoFormat(durationSec int, title string) string {
	lower := strings.ToLower(title)
	if (durationSec > 0 && durationSec <= 65) || strings.Contains(lower, "#shorts") || strings.Contains(lower, "#short") {
		return "Short (<60s)"
	}
	if durationSec >= 480 {
		return "Long-Form (8m+)"
	}
	if durationSec > 0 {
		return "Mid-Form (1–8m)"
	}
	return "Standard"
}

// ComputeEngagementRate calculates ((likes + comments) / views) * 100 rounded to 2 decimal places.
func ComputeEngagementRate(views, likes, comments int64) float64 {
	if views <= 0 || (likes <= 0 && comments <= 0) {
		return 0
	}
	return math.Round((float64(likes+comments)/float64(views))*10000) / 100
}

// ParseCompactCount converts human-formatted counts ("1.42M subscribers", "84.5K views", "500,000+", "20K+") into int64.
func ParseCompactCount(raw string) int64 {
	m := ytCompactNumRegexp.FindStringSubmatch(strings.TrimSpace(raw))
	if len(m) < 2 {
		return 0
	}
	numStr := strings.ReplaceAll(m[1], ",", "")
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil || val <= 0 {
		return 0
	}
	suffix := ""
	if len(m) > 2 {
		suffix = strings.ToUpper(strings.TrimSpace(m[2]))
	}
	switch suffix {
	case "K":
		val *= 1_000
	case "M":
		val *= 1_000_000
	case "B":
		val *= 1_000_000_000
	}
	return int64(math.Round(val))
}

// ComputeMedian returns the median of positive int64 samples.
func ComputeMedian(vals []int64) float64 {
	var clean []int64
	for _, v := range vals {
		if v > 0 {
			clean = append(clean, v)
		}
	}
	if len(clean) == 0 {
		return 0
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i] < clean[j] })
	mid := len(clean) / 2
	if len(clean)%2 == 1 {
		return float64(clean[mid])
	}
	return float64(clean[mid-1]+clean[mid]) / 2.0
}

// ComputeChannelRelativeMultiplier compares a video's actual views against the channel's baseline
// average/median views per video. Returns a neutral 1.0x when no real baseline is available,
// and clamps extreme ratios cleanly to [0.2, 50.0].
func ComputeChannelRelativeMultiplier(videoViews int64, channelBaselineViews float64) float64 {
	if videoViews <= 0 || channelBaselineViews <= 0 {
		return 1.0
	}
	ratio := float64(videoViews) / channelBaselineViews
	if ratio < 0.2 {
		ratio = 0.2
	}
	if ratio > 50.0 {
		ratio = 50.0
	}
	return math.Round(ratio*10) / 10
}

// ParseChannelBaselineFromHTML extracts real subscriber count and median views across recent uploads
// from a public YouTube channel /videos page.
func ParseChannelBaselineFromHTML(html string) (subscribers int64, medianRecentViews float64) {
	if sm := ytSubscriberRe.FindStringSubmatch(html); len(sm) > 1 {
		subscribers = ParseCompactCount(sm[1])
	}
	var recentViews []int64
	for _, m := range ytViewsRe.FindAllStringSubmatch(html, 15) {
		if len(m) > 1 {
			if v := ParseCompactCount(m[1]); v > 0 {
				recentViews = append(recentViews, v)
			}
		}
	}
	if len(recentViews) == 0 {
		for _, m := range ytShortViewsRe.FindAllStringSubmatch(html, 15) {
			if len(m) > 1 {
				if v := ParseCompactCount(m[1]); v > 0 {
					recentViews = append(recentViews, v)
				}
			}
		}
	}
	return subscribers, ComputeMedian(recentViews)
}

func fetchPublicChannelBaseline(ctx context.Context, client *http.Client, channelID string) (int64, float64) {
	if !strings.HasPrefix(channelID, "UC") {
		return 0, 0
	}
	chCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(chCtx, "GET", fmt.Sprintf("https://www.youtube.com/channel/%s/videos", channelID), nil)
	if err != nil {
		return 0, 0
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return 0, 0
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1_500_000))
	resp.Body.Close()
	if err != nil {
		return 0, 0
	}
	return ParseChannelBaselineFromHTML(string(bodyBytes))
}

func parseRelativeHours(pubText string) float64 {
	lower := strings.ToLower(strings.TrimSpace(pubText))
	numRe := regexp.MustCompile(`(\d+)`)
	m := numRe.FindStringSubmatch(lower)
	val := 24.0
	if len(m) > 1 {
		if n, err := strconv.ParseFloat(m[1], 64); err == nil && n > 0 {
			val = n
		}
	}
	switch {
	case strings.Contains(lower, "minute"):
		return math.Max(0.5, val/60.0)
	case strings.Contains(lower, "hour"):
		return val
	case strings.Contains(lower, "day"):
		return val * 24.0
	case strings.Contains(lower, "week"):
		return val * 24.0 * 7.0
	case strings.Contains(lower, "month"):
		return val * 24.0 * 30.0
	case strings.Contains(lower, "year"):
		return val * 24.0 * 365.0
	default:
		return 72.0
	}
}

func LookupYouTubeVideoForPerson(ctx context.Context, personName string, youtubeAPIKey string) (*VideoMetric, error) {
	list, err := LookupYouTubeIntelForPerson(ctx, personName, youtubeAPIKey)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	top := list[0]
	return &top, nil
}

// LookupYouTubeIntelForPerson retrieves up to 5 real competing/breakout YouTube videos for a subject,
// including duration, format (Short vs Mid-Form vs Long-Form), engagement rate, real channel subscribers,
// real channel baseline views, and true channel-relative outlier multipliers (sorted by VPH descending).
func LookupYouTubeIntelForPerson(ctx context.Context, personName string, youtubeAPIKey string) ([]VideoMetric, error) {
	client := &http.Client{Timeout: 6 * time.Second}

	// 1. Official YouTube Data API v3 (batched 3 calls = 102 units total for up to 5 full video + channel profiles)
	if strings.TrimSpace(youtubeAPIKey) != "" {
		apiKey := strings.TrimSpace(youtubeAPIKey)
		q := url.QueryEscape(personName + " documentary OR interview OR story")
		searchURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&type=video&maxResults=5&order=relevance&q=%s&key=%s", q, apiKey)
		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err == nil {
			if resp, err2 := client.Do(req); err2 == nil && resp.StatusCode == 200 {
				var sRes struct {
					Items []struct {
						ID struct {
							VideoID string `json:"videoId"`
						} `json:"id"`
						Snippet struct {
							Title        string `json:"title"`
							ChannelID    string `json:"channelId"`
							ChannelTitle string `json:"channelTitle"`
							PublishedAt  string `json:"publishedAt"`
							Thumbnails   struct {
								Medium struct {
									URL string `json:"url"`
								} `json:"medium"`
								High struct {
									URL string `json:"url"`
								} `json:"high"`
							} `json:"thumbnails"`
						} `json:"snippet"`
					} `json:"items"`
				}
				_ = json.NewDecoder(resp.Body).Decode(&sRes)
				resp.Body.Close()

				if len(sRes.Items) > 0 {
					var ids []string
					chIDSeen := map[string]bool{}
					var chIDs []string
					for _, it := range sRes.Items {
						if it.ID.VideoID != "" {
							ids = append(ids, it.ID.VideoID)
						}
						if it.Snippet.ChannelID != "" && !chIDSeen[it.Snippet.ChannelID] {
							chIDSeen[it.Snippet.ChannelID] = true
							chIDs = append(chIDs, it.Snippet.ChannelID)
						}
					}

					type vStat struct {
						views, likes, comments int64
						durationSec            int
					}
					statsByID := map[string]vStat{}
					statsURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=statistics,contentDetails&id=%s&key=%s", strings.Join(ids, ","), apiKey)
					if sReq, err3 := http.NewRequestWithContext(ctx, "GET", statsURL, nil); err3 == nil {
						if sResp, err4 := client.Do(sReq); err4 == nil && sResp.StatusCode == 200 {
							var stData struct {
								Items []struct {
									ID         string `json:"id"`
									Statistics struct {
										ViewCount    string `json:"viewCount"`
										LikeCount    string `json:"likeCount"`
										CommentCount string `json:"commentCount"`
									} `json:"statistics"`
									ContentDetails struct {
										Duration string `json:"duration"`
									} `json:"contentDetails"`
								} `json:"items"`
							}
							_ = json.NewDecoder(sResp.Body).Decode(&stData)
							sResp.Body.Close()
							for _, st := range stData.Items {
								v, _ := strconv.ParseInt(st.Statistics.ViewCount, 10, 64)
								l, _ := strconv.ParseInt(st.Statistics.LikeCount, 10, 64)
								c, _ := strconv.ParseInt(st.Statistics.CommentCount, 10, 64)
								statsByID[st.ID] = vStat{
									views:       v,
									likes:       l,
									comments:    c,
									durationSec: ParseISO8601Duration(st.ContentDetails.Duration),
								}
							}
						}
					}

					// Batch-fetch ALL channels' real subscriber counts & avg views/video in 1 single API call (1 unit!)
					type chStat struct {
						subs     int64
						baseline float64
					}
					chStats := map[string]chStat{}
					if len(chIDs) > 0 {
						chURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=statistics&id=%s&key=%s", strings.Join(chIDs, ","), apiKey)
						if cReq, errC := http.NewRequestWithContext(ctx, "GET", chURL, nil); errC == nil {
							if cResp, errC2 := client.Do(cReq); errC2 == nil && cResp.StatusCode == 200 {
								var chData struct {
									Items []struct {
										ID         string `json:"id"`
										Statistics struct {
											SubscriberCount string `json:"subscriberCount"`
											ViewCount       string `json:"viewCount"`
											VideoCount      string `json:"videoCount"`
										} `json:"statistics"`
									} `json:"items"`
								}
								_ = json.NewDecoder(cResp.Body).Decode(&chData)
								cResp.Body.Close()
								for _, ch := range chData.Items {
									subs, _ := strconv.ParseInt(ch.Statistics.SubscriberCount, 10, 64)
									chViews, _ := strconv.ParseInt(ch.Statistics.ViewCount, 10, 64)
									chVids, _ := strconv.ParseInt(ch.Statistics.VideoCount, 10, 64)
									var base float64
									if chViews > 0 && chVids > 0 {
										base = float64(chViews) / float64(chVids)
									}
									chStats[ch.ID] = chStat{subs: subs, baseline: base}
								}
							}
						}
					}

					competingCount := 0
					var peerViews []int64
					for _, it := range sRes.Items {
						st := statsByID[it.ID.VideoID]
						if st.views > 0 {
							peerViews = append(peerViews, st.views)
							if st.views >= 50_000 {
								competingCount++
							}
						}
					}
					peerMedian := ComputeMedian(peerViews)

					var metrics []VideoMetric
					for _, it := range sRes.Items {
						st := statsByID[it.ID.VideoID]
						if st.views <= 0 {
							continue
						}
						hoursSince := 72.0
						pubDate := time.Now().Format("2006-01-02")
						if t, errT := time.Parse(time.RFC3339, it.Snippet.PublishedAt); errT == nil {
							hoursSince = math.Max(1.0, time.Since(t).Hours())
							pubDate = t.Format("2006-01-02")
						}
						cs := chStats[it.Snippet.ChannelID]
						baseline := cs.baseline
						if baseline <= 0 {
							baseline = peerMedian
						}
						vph := math.Round(float64(st.views) / hoursSince)
						vpd := math.Round(vph * 24.0)
						mult := ComputeChannelRelativeMultiplier(st.views, baseline)
						thumb := it.Snippet.Thumbnails.Medium.URL
						if thumb == "" {
							thumb = it.Snippet.Thumbnails.High.URL
						}
						if thumb == "" {
							thumb = fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", it.ID.VideoID)
						}

						metrics = append(metrics, VideoMetric{
							Title:                it.Snippet.Title,
							Views:                st.views,
							Likes:                st.likes,
							Comments:             st.comments,
							DurationSeconds:      st.durationSec,
							VideoFormat:          ClassifyVideoFormat(st.durationSec, it.Snippet.Title),
							EngagementRate:       ComputeEngagementRate(st.views, st.likes, st.comments),
							ThumbnailURL:         thumb,
							ChannelID:            it.Snippet.ChannelID,
							ChannelName:          strings.TrimSpace(it.Snippet.ChannelTitle),
							ChannelSubscribers:   cs.subs,
							ChannelBaselineViews: math.Round(baseline),
							CompetingVideosCount: competingCount,
							VideoURL:             "https://www.youtube.com/watch?v=" + it.ID.VideoID,
							ExternalID:           "yt-" + it.ID.VideoID,
							PublishedAt:          pubDate,
							ViewsPerHour:         vph,
							ViewsPerDay:          vpd,
							ChannelMultiplier:    mult,
						})
					}

					if len(metrics) > 0 {
						sort.Slice(metrics, func(i, j int) bool {
							if metrics[i].ViewsPerHour != metrics[j].ViewsPerHour {
								return metrics[i].ViewsPerHour > metrics[j].ViewsPerHour
							}
							return metrics[i].Views > metrics[j].Views
						})
						return metrics, nil
					}
				}
			} else if resp != nil {
				resp.Body.Close()
			}
		}
	}

	// 2. Public YouTube Search Page + Public Channel /videos Baseline Extraction (No API Key Required)
	q := url.QueryEscape(personName + " story OR interview OR documentary")
	searchPageURL := fmt.Sprintf("https://www.youtube.com/results?search_query=%s", q)
	req, err := http.NewRequestWithContext(ctx, "GET", searchPageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("youtube search status %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2_000_000))
	if err != nil {
		return nil, err
	}
	html := string(bodyBytes)

	vMatches := ytVideoBlockRe.FindAllStringSubmatchIndex(html, 8)
	if len(vMatches) == 0 {
		return nil, fmt.Errorf("no youtube videoRenderer found")
	}

	type parsedYT struct {
		videoID     string
		title       string
		views       int64
		durationSec int
		hoursSince  float64
		vph         float64
		channelName string
		channelID   string
	}
	var candidates []parsedYT
	var peerViews []int64
	competingCount := 0

	for _, match := range vMatches {
		videoID := html[match[2]:match[3]]
		title := strings.ReplaceAll(html[match[4]:match[5]], `\u0026`, "&")
		windowEnd := match[1] + 3200
		if windowEnd > len(html) {
			windowEnd = len(html)
		}
		snippet := html[match[0]:windowEnd]

		var views int64
		if vm := ytViewsRe.FindStringSubmatch(snippet); len(vm) > 1 {
			views = ParseCompactCount(vm[1])
		} else if sm := ytShortViewsRe.FindStringSubmatch(snippet); len(sm) > 1 {
			views = ParseCompactCount(sm[1])
		}
		if views <= 0 {
			continue
		}

		durSec := 0
		if lm := ytLengthRe.FindStringSubmatch(snippet); len(lm) > 1 {
			durSec = ParseClockDuration(lm[1])
		}

		hoursSince := 72.0
		if pm := ytPubTimeRe.FindStringSubmatch(snippet); len(pm) > 1 {
			hoursSince = parseRelativeHours(pm[1])
		}
		channelName := "YouTube Channel"
		if cm := ytOwnerRe.FindStringSubmatch(snippet); len(cm) > 1 {
			channelName = strings.TrimSpace(cm[1])
		}
		channelID := ""
		if bm := ytBrowseIDRe.FindStringSubmatch(snippet); len(bm) > 1 {
			channelID = bm[1]
		}

		peerViews = append(peerViews, views)
		if views >= 50_000 {
			competingCount++
		}
		vph := math.Round(float64(views) / math.Max(1.0, hoursSince))
		candidates = append(candidates, parsedYT{
			videoID:     videoID,
			title:       title,
			views:       views,
			durationSec: durSec,
			hoursSince:  hoursSince,
			vph:         vph,
			channelName: channelName,
			channelID:   channelID,
		})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no youtube videos with real view counts found")
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].vph != candidates[j].vph {
			return candidates[i].vph > candidates[j].vph
		}
		return candidates[i].views > candidates[j].views
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	peerMedian := ComputeMedian(peerViews)
	var out []VideoMetric
	for idx, c := range candidates {
		var subs int64
		var baseline float64
		// Fetch real channel baseline for top 2 breakout videos to stay fast
		if idx < 2 && c.channelID != "" {
			subs, baseline = fetchPublicChannelBaseline(ctx, client, c.channelID)
		}
		if baseline <= 0 {
			baseline = peerMedian
		}
		mult := ComputeChannelRelativeMultiplier(c.views, baseline)
		pubDate := time.Now().Add(-time.Duration(c.hoursSince) * time.Hour).Format("2006-01-02")
		out = append(out, VideoMetric{
			Title:                c.title,
			Views:                c.views,
			Likes:                0,
			Comments:             0,
			DurationSeconds:      c.durationSec,
			VideoFormat:          ClassifyVideoFormat(c.durationSec, c.title),
			EngagementRate:       0,
			ThumbnailURL:         fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", c.videoID),
			ChannelID:            c.channelID,
			ChannelName:          c.channelName,
			ChannelSubscribers:   subs,
			ChannelBaselineViews: math.Round(baseline),
			CompetingVideosCount: competingCount,
			VideoURL:             "https://www.youtube.com/watch?v=" + c.videoID,
			ExternalID:           "yt-" + c.videoID,
			PublishedAt:          pubDate,
			ViewsPerHour:         c.vph,
			ViewsPerDay:          math.Round(c.vph * 24.0),
			ChannelMultiplier:    mult,
		})
	}
	return out, nil
}
