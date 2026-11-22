package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type CompetitorVideoStudy struct {
	VideoID             string   `json:"videoId"`
	Title               string   `json:"title"`
	ChannelName         string   `json:"channelName"`
	URL                 string   `json:"url"`
	Views               int64    `json:"views"`
	ViewsPerHour        float64  `json:"viewsPerHour"`
	ChannelMultiplier   float64  `json:"channelMultiplier"`
	EngagementRate      float64  `json:"engagementRate"`
	VideoFormat         string   `json:"videoFormat"`
	DurationSeconds     int      `json:"durationSeconds"`
	TranscriptExcerpt   string   `json:"transcriptExcerpt,omitempty"`
	TranscriptAvailable bool     `json:"transcriptAvailable"`
	TopComments         []string `json:"topComments,omitempty"`
}

var (
	ytCaptionTracksRe = regexp.MustCompile(`"captionTracks":(\[[^\]]+\])`)
	ytHTMLTagRe       = regexp.MustCompile(`<[^>]+>`)
)

// ExtractVideoID pulls the 11-character YouTube video ID from an externalId ("yt-...") or watch URL.
func ExtractVideoID(externalID, videoURL string) string {
	if strings.HasPrefix(externalID, "yt-") && len(externalID) >= 14 {
		return strings.TrimPrefix(externalID, "yt-")
	}
	if idx := strings.Index(videoURL, "v="); idx >= 0 {
		rem := videoURL[idx+2:]
		if amp := strings.Index(rem, "&"); amp >= 0 {
			rem = rem[:amp]
		}
		if len(rem) == 11 {
			return rem
		}
	}
	return ""
}

// ExtractCaptionTrackURL locates the best English (or first available) caption track baseUrl in YouTube watch HTML.
func ExtractCaptionTrackURL(watchHTML string) string {
	m := ytCaptionTracksRe.FindStringSubmatch(watchHTML)
	if len(m) < 2 {
		return ""
	}
	var tracks []struct {
		BaseURL      string `json:"baseUrl"`
		LanguageCode string `json:"languageCode"`
		Kind         string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(m[1]), &tracks); err != nil || len(tracks) == 0 {
		return ""
	}
	// Prefer manual English captions, then auto-generated English, then any track
	bestURL := ""
	for _, tr := range tracks {
		if tr.BaseURL == "" {
			continue
		}
		lang := strings.ToLower(tr.LanguageCode)
		if strings.HasPrefix(lang, "en") && tr.Kind != "asr" {
			return strings.ReplaceAll(tr.BaseURL, `\u0026`, "&")
		}
		if strings.HasPrefix(lang, "en") && bestURL == "" {
			bestURL = strings.ReplaceAll(tr.BaseURL, `\u0026`, "&")
		}
	}
	if bestURL != "" {
		return bestURL
	}
	return strings.ReplaceAll(tracks[0].BaseURL, `\u0026`, "&")
}

// ParseYouTubeCaptionXML decodes YouTube timedtext XML (<transcript><text>...</text></transcript>) into clean prose.
func ParseYouTubeCaptionXML(rawXML string) string {
	type textNode struct {
		Value string `xml:",chardata"`
	}
	type transcriptDoc struct {
		Texts []textNode `xml:"text"`
	}
	var doc transcriptDoc
	if err := xml.Unmarshal([]byte(rawXML), &doc); err == nil && len(doc.Texts) > 0 {
		var parts []string
		for _, n := range doc.Texts {
			txt := html.UnescapeString(n.Value)
			txt = ytHTMLTagRe.ReplaceAllString(txt, " ")
			txt = strings.Join(strings.Fields(txt), " ")
			if txt != "" {
				parts = append(parts, txt)
			}
		}
		return strings.TrimSpace(strings.Join(parts, " "))
	}

	// Fallback for srv3/timedtext format="3" (<timedtext format="3"><body><p><s>...</s></p></body></timedtext>)
	type pNode struct {
		Value string `xml:",innerxml"`
	}
	type bodyDoc struct {
		Ps []pNode `xml:"body>p"`
	}
	var bdoc bodyDoc
	if err := xml.Unmarshal([]byte(rawXML), &bdoc); err == nil && len(bdoc.Ps) > 0 {
		var parts []string
		for _, n := range bdoc.Ps {
			rawInner := ytHTMLTagRe.ReplaceAllString(n.Value, "")
			txt := html.UnescapeString(rawInner)
			txt = strings.Join(strings.Fields(txt), " ")
			if txt != "" {
				parts = append(parts, txt)
			}
		}
		return strings.TrimSpace(strings.Join(parts, " "))
	}

	return ""
}

type ytCaptionTrack struct {
	BaseURL      string `json:"baseUrl"`
	LanguageCode string `json:"languageCode"`
	Kind         string `json:"kind"`
}

func selectBestCaptionTrackURL(tracks []ytCaptionTrack) string {
	if len(tracks) == 0 {
		return ""
	}
	bestURL := ""
	for _, tr := range tracks {
		if tr.BaseURL == "" {
			continue
		}
		lang := strings.ToLower(tr.LanguageCode)
		if strings.HasPrefix(lang, "en") && tr.Kind != "asr" {
			return strings.ReplaceAll(tr.BaseURL, `\u0026`, "&")
		}
		if strings.HasPrefix(lang, "en") && bestURL == "" {
			bestURL = strings.ReplaceAll(tr.BaseURL, `\u0026`, "&")
		}
	}
	if bestURL != "" {
		return bestURL
	}
	return strings.ReplaceAll(tracks[0].BaseURL, `\u0026`, "&")
}

// fetchCaptionTrackURLViaInnerTube queries YouTube's public InnerTube ANDROID player endpoint
// (0 Data API quota units), which reliably returns working timedtext caption URLs even when watch HTML returns 0 bytes.
func fetchCaptionTrackURLViaInnerTube(ctx context.Context, client *http.Client, videoID string) string {
	payload := fmt.Sprintf(`{"context":{"client":{"clientName":"ANDROID","clientVersion":"20.10.38","hl":"en"}},"videoId":"%s"}`, videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.youtube.com/youtubei/v1/player?prettyPrint=false", bytes.NewBufferString(payload))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "com.google.android.youtube/20.10.38 (Linux; U; Android 14)")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1_500_000))
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		return ""
	}

	var parsed struct {
		Captions struct {
			PlayerCaptionsTracklistRenderer struct {
				CaptionTracks []ytCaptionTrack `json:"captionTracks"`
			} `json:"playerCaptionsTracklistRenderer"`
		} `json:"captions"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return ""
	}
	return selectBestCaptionTrackURL(parsed.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks)
}

func fetchAndParseCaptionURL(ctx context.Context, client *http.Client, captionURL string) string {
	if captionURL == "" {
		return ""
	}
	cReq, err := http.NewRequestWithContext(ctx, http.MethodGet, captionURL, nil)
	if err != nil {
		return ""
	}
	cReq.Header.Set("User-Agent", UserAgent)
	cResp, err := client.Do(cReq)
	if err != nil {
		return ""
	}
	capBytes, err := io.ReadAll(io.LimitReader(cResp.Body, 750_000))
	cResp.Body.Close()
	if err != nil || cResp.StatusCode != 200 || len(capBytes) == 0 {
		return ""
	}
	return ParseYouTubeCaptionXML(string(capBytes))
}

// FetchYouTubeTranscript retrieves a YouTube video's public caption track (0 API quota units)
// using InnerTube ANDROID primary + Watch HTML fallback, returning a bounded excerpt (~3,200 chars).
func FetchYouTubeTranscript(ctx context.Context, videoID string) (string, error) {
	if len(videoID) != 11 {
		return "", fmt.Errorf("invalid youtube video id: %q", videoID)
	}
	tCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 6 * time.Second}

	// Primary: InnerTube ANDROID player endpoint (reliably returns non-empty timedtext XML)
	fullText := ""
	if innerURL := fetchCaptionTrackURLViaInnerTube(tCtx, client, videoID); innerURL != "" {
		fullText = fetchAndParseCaptionURL(tCtx, client, innerURL)
	}

	// Fallback: Public watch page HTML captionTracks
	if fullText == "" {
		watchURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s&hl=en", videoID)
		req, err := http.NewRequestWithContext(tCtx, http.MethodGet, watchURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", UserAgent)
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			if resp, errDo := client.Do(req); errDo == nil {
				bodyBytes, errRead := io.ReadAll(io.LimitReader(resp.Body, 2_000_000))
				resp.Body.Close()
				if errRead == nil && resp.StatusCode == 200 {
					if captionURL := ExtractCaptionTrackURL(string(bodyBytes)); captionURL != "" {
						fullText = fetchAndParseCaptionURL(tCtx, client, captionURL)
					}
				}
			}
		}
	}

	if fullText == "" {
		return "", fmt.Errorf("no public caption transcript available for %s", videoID)
	}
	// Preserve opening hook + closing beat cleanly within ~3,200 chars
	if len(fullText) > 3200 {
		opening := fullText[:2000]
		closing := fullText[len(fullText)-1000:]
		return strings.TrimSpace(opening) + " [...] " + strings.TrimSpace(closing), nil
	}
	return fullText, nil
}

// FetchYouTubeTopComments retrieves top relevance-ranked viewer comments via YouTube Data API v3 (1 quota unit).
func FetchYouTubeTopComments(ctx context.Context, videoID string, youtubeAPIKey string, limit int) ([]string, error) {
	apiKey := strings.TrimSpace(youtubeAPIKey)
	if apiKey == "" || len(videoID) != 11 {
		return nil, nil
	}
	if limit <= 0 || limit > 15 {
		limit = 8
	}
	cCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/commentThreads?part=snippet&order=relevance&textFormat=plainText&maxResults=%d&videoId=%s&key=%s",
		limit, videoID, apiKey,
	)
	req, err := http.NewRequestWithContext(cCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("commentThreads status %d", resp.StatusCode)
	}

	var data struct {
		Items []struct {
			Snippet struct {
				TopLevelComment struct {
					Snippet struct {
						TextDisplay string `json:"textDisplay"`
						LikeCount   int64  `json:"likeCount"`
					} `json:"snippet"`
				} `json:"topLevelComment"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var out []string
	for _, it := range data.Items {
		txt := strings.Join(strings.Fields(it.Snippet.TopLevelComment.Snippet.TextDisplay), " ")
		if len(txt) < 12 {
			continue
		}
		if len(txt) > 220 {
			txt = txt[:217] + "..."
		}
		if likes := it.Snippet.TopLevelComment.Snippet.LikeCount; likes > 0 {
			out = append(out, fmt.Sprintf("[%d likes] %s", likes, txt))
		} else {
			out = append(out, txt)
		}
	}
	return out, nil
}

// AnalyzeCompetitorVideosConcurrently fetches transcripts and top viewer comments across up to maxVideos
// competing YouTube videos in parallel within a tight wall-clock budget.
func AnalyzeCompetitorVideosConcurrently(ctx context.Context, videos []VideoMetric, youtubeAPIKey string, maxVideos int) []CompetitorVideoStudy {
	if len(videos) == 0 {
		return nil
	}
	if maxVideos <= 0 || maxVideos > len(videos) {
		maxVideos = len(videos)
	}
	if maxVideos > 4 {
		maxVideos = 4
	}

	target := videos[:maxVideos]
	results := make([]CompetitorVideoStudy, len(target))

	studyCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i, v := range target {
		wg.Add(1)
		go func(idx int, vm VideoMetric) {
			defer wg.Done()
			vidID := ExtractVideoID(vm.ExternalID, vm.VideoURL)
			st := CompetitorVideoStudy{
				VideoID:           vidID,
				Title:             vm.Title,
				ChannelName:       vm.ChannelName,
				URL:               vm.VideoURL,
				Views:             vm.Views,
				ViewsPerHour:      vm.ViewsPerHour,
				ChannelMultiplier: vm.ChannelMultiplier,
				EngagementRate:    vm.EngagementRate,
				VideoFormat:       vm.VideoFormat,
				DurationSeconds:   vm.DurationSeconds,
			}
			if vidID != "" {
				var innerWG sync.WaitGroup
				innerWG.Add(2)
				go func() {
					defer innerWG.Done()
					if tr, err := FetchYouTubeTranscript(studyCtx, vidID); err == nil && tr != "" {
						st.TranscriptExcerpt = tr
						st.TranscriptAvailable = true
					}
				}()
				go func() {
					defer innerWG.Done()
					if cmts, err := FetchYouTubeTopComments(studyCtx, vidID, youtubeAPIKey, 8); err == nil && len(cmts) > 0 {
						st.TopComments = cmts
					}
				}()
				innerWG.Wait()
			}
			results[idx] = st
		}(i, v)
	}
	wg.Wait()
	return results
}
