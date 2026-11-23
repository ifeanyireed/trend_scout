package connectors

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseCompactCount(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"1.42M subscribers", 1_420_000},
		{"84.5K views", 84_500},
		{"500,000+", 500_000},
		{"20K+", 20_000},
		{"1,234,567 views", 1_234_567},
		{"2.5B views", 2_500_000_000},
		{"no views", 0},
	}
	for _, tc := range cases {
		if got := ParseCompactCount(tc.in); got != tc.want {
			t.Fatalf("ParseCompactCount(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestComputeChannelRelativeMultiplier(t *testing.T) {
	// Breakout video: 500k views on a 50k-median channel -> 10.0x
	if got := ComputeChannelRelativeMultiplier(500_000, 50_000); got != 10.0 {
		t.Fatalf("expected 10.0x breakout multiplier, got %.1fx", got)
	}
	// Underperforming video: 25k views on a 100k-median channel -> 0.3x
	if got := ComputeChannelRelativeMultiplier(25_000, 100_000); got != 0.3 {
		t.Fatalf("expected 0.3x underperforming multiplier, got %.1fx", got)
	}
	// Neutral 1.0x when no baseline exists (zero synthetic inflation)
	if got := ComputeChannelRelativeMultiplier(50_000, 0); got != 1.0 {
		t.Fatalf("expected neutral 1.0x when baseline is 0, got %.1fx", got)
	}
	// Clamped ceiling at 50.0x
	if got := ComputeChannelRelativeMultiplier(10_000_000, 1_000); got != 50.0 {
		t.Fatalf("expected 50.0x clamp, got %.1fx", got)
	}
}

func TestParseChannelBaselineFromHTML(t *testing.T) {
	sampleHTML := `
	{"subscriberCountText":{"simpleText":"1.25M subscribers"}}
	{"viewCountText":{"simpleText":"42,000 views"}}
	{"viewCountText":{"simpleText":"58,000 views"}}
	{"viewCountText":{"simpleText":"50,000 views"}}
	{"viewCountText":{"simpleText":"120,000 views"}}
	{"viewCountText":{"simpleText":"48,000 views"}}
	`
	subs, median := ParseChannelBaselineFromHTML(sampleHTML)
	if subs != 1_250_000 {
		t.Fatalf("expected 1,250,000 subscribers, got %d", subs)
	}
	if median != 50_000 {
		t.Fatalf("expected median baseline 50,000 views, got %.1f", median)
	}
}

func TestYouTubeDurationFormatAndEngagement(t *testing.T) {
	if got := ParseISO8601Duration("PT18M42S"); got != 1122 {
		t.Fatalf("expected 1122s, got %d", got)
	}
	if got := ParseISO8601Duration("PT1H2M5S"); got != 3725 {
		t.Fatalf("expected 3725s, got %d", got)
	}
	if got := ParseISO8601Duration("PT59S"); got != 59 {
		t.Fatalf("expected 59s, got %d", got)
	}
	if got := ClassifyVideoFormat(59, "59 Seconds #shorts"); got != "Short (<60s)" {
		t.Fatalf("expected Short (<60s), got %q", got)
	}
	if got := ClassifyVideoFormat(240, "Press interview clip"); got != "Mid-Form (1–8m)" {
		t.Fatalf("expected Mid-Form (1–8m), got %q", got)
	}
	if got := ClassifyVideoFormat(1122, "Full Documentary"); got != "Long-Form (8m+)" {
		t.Fatalf("expected Long-Form (8m+), got %q", got)
	}
	if got := ComputeEngagementRate(100_000, 5_500, 500); got != 6.0 {
		t.Fatalf("expected 6.0%% engagement rate, got %.2f%%", got)
	}
}

func TestYouTubeTranscriptExtractionAndParsing(t *testing.T) {
	sampleWatchHTML := `{"captions":{"playerCaptionsTracklistRenderer":{"captionTracks":[{"baseUrl":"https://www.youtube.com/api/timedtext?v=abc12345678\u0026lang=es","languageCode":"es"},{"baseUrl":"https://www.youtube.com/api/timedtext?v=abc12345678\u0026lang=en","languageCode":"en"}]}}}`
	u := ExtractCaptionTrackURL(sampleWatchHTML)
	if !strings.Contains(u, "lang=en") || strings.Contains(u, `\u0026`) {
		t.Fatalf("expected unescaped English caption track URL, got %q", u)
	}

	sampleXML := `<transcript><text start="0.0" dur="3.2">Nobody expected him to win.</text><text start="3.2" dur="4.1">He grew up training on a dirt field &amp; broke the record.</text></transcript>`
	parsed := ParseYouTubeCaptionXML(sampleXML)
	if parsed != "Nobody expected him to win. He grew up training on a dirt field & broke the record." {
		t.Fatalf("unexpected parsed transcript: %q", parsed)
	}
}

func TestLiveYouTubeDataAPIV3(t *testing.T) {
	if os.Getenv("RUN_LIVE_YT_TEST") != "1" {
		t.Skip("set RUN_LIVE_YT_TEST=1 to verify live YouTube Data API v3 lookup")
	}
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		if raw, err := os.ReadFile("../../../.env"); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "YOUTUBE_API_KEY=") {
					apiKey = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "YOUTUBE_API_KEY="))
				}
			}
		}
	}
	if apiKey == "" {
		t.Fatal("YOUTUBE_API_KEY not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	vid, err := LookupYouTubeVideoForPerson(ctx, "Neeraj Chopra", apiKey)
	if err != nil {
		t.Fatalf("LookupYouTubeVideoForPerson failed: %v", err)
	}
	if vid.Views <= 0 || vid.ChannelSubscribers <= 0 || vid.ChannelBaselineViews <= 0 {
		t.Fatalf("expected real YouTube API v3 stats (views, subscribers, baseline), got %+v", vid)
	}
	t.Logf("LIVE YT API v3 OK -> title=%q views=%d likes=%d comments=%d vph=%.0f channel=%q subs=%d baseline=%.0f mult=%.1fx competing=%d url=%s",
		vid.Title, vid.Views, vid.Likes, vid.Comments, vid.ViewsPerHour, vid.ChannelName, vid.ChannelSubscribers, vid.ChannelBaselineViews, vid.ChannelMultiplier, vid.CompetingVideosCount, vid.VideoURL)
}

func TestSocialHandleVerificationAndEntityExtraction(t *testing.T) {
	// Reject fan pages, parody accounts, clip channels, and impersonators
	for _, bad := range []string{"@neeraj_fanpage", "@olympics_daily", "@parody_account", "@unofficial_clips", "@tribute_edits", "@star_stan"} {
		if !IsLikelyFanOrImpersonatorHandle(bad) {
			t.Fatalf("expected %q to be rejected as fan/impersonator", bad)
		}
	}
	// Accept legitimate official handles
	for _, good := range []string{"@Neeraj_chopra1", "@Olympics", "@WorldAthletics", "@NVIDIA", "@neeraj____chopra"} {
		if IsLikelyFanOrImpersonatorHandle(good) {
			t.Fatalf("expected %q to be accepted as legitimate handle", good)
		}
	}

	entities := ExtractCandidateEntities(
		"Neeraj Chopra",
		"Neeraj Chopra won Olympic gold at the Tokyo Olympics under World Athletics and Sports Authority of India. He trained with Klaus Bartonietz and competed against Arshad Nadeem.",
		"Neeraj Chopra makes World Athletics history",
		"Gold medal for India at World Athletics Championships",
		nil,
	)
	if len(entities) < 2 {
		t.Fatalf("expected notable entities extracted, got %v", entities)
	}
}

func TestLiveWikidataSocialVerification(t *testing.T) {
	if os.Getenv("RUN_LIVE_SOCIAL_VERIFY") != "1" {
		t.Skip("set RUN_LIVE_SOCIAL_VERIFY=1 to test live Wikidata official social account verification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	accounts := VerifyOfficialSocialAccounts(ctx, "Neeraj Chopra", []string{"World Athletics", "International Olympic Committee"}, "")
	if len(accounts) == 0 {
		t.Fatalf("expected at least 1 verified official account from Wikidata")
	}
	for _, a := range accounts {
		t.Logf("VERIFIED OFFICIAL: %s (%s) [%s] YT=%q X=%q IG=%q", a.EntityName, a.EntityType, a.VerifiedBy, a.YouTubeHandle, a.TwitterHandle, a.InstagramHandle)
	}
}
