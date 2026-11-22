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
