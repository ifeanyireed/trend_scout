package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"trendscout-backend/internal/agents"
	"trendscout-backend/internal/api"
	"trendscout-backend/internal/db"
	"trendscout-backend/internal/queue"
)

func TestEndToEndTrendScoutPipeline(t *testing.T) {
	conn, err := db.ConnectAndMigrate()
	if err != nil {
		t.Fatalf("Failed to connect and migrate PostgreSQL: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	agents.RegisterQueueHandlers()

	if err := agents.EnsureInitialOperationalData(ctx); err != nil {
		t.Fatalf("Failed to seed operational data: %v", err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// 1. Test Authentication
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "producer@trendscout.studio",
		"password": "trendscout2026",
	})
	resp, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("Login failed: err=%v status=%d", err, resp.StatusCode)
	}
	var authRes struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&authRes)
	resp.Body.Close()
	if authRes.Token == "" {
		t.Fatalf("Expected non-empty JWT token")
	}

	// 2. Test Chrome Extension Summary Endpoint
	extResp, err := http.Get(server.URL + "/api/extension/summary")
	if err != nil || extResp.StatusCode != 200 {
		t.Fatalf("Extension summary failed: %v", err)
	}
	var extData struct {
		MonitoringActive    bool             `json:"monitoringActive"`
		NewDiscoveriesCount int              `json:"newDiscoveriesCount"`
		LatestDiscoveries   []map[string]any `json:"latestDiscoveries"`
	}
	_ = json.NewDecoder(extResp.Body).Decode(&extData)
	extResp.Body.Close()
	if !extData.MonitoringActive || extData.NewDiscoveriesCount < 5 || len(extData.LatestDiscoveries) == 0 {
		t.Fatalf("Unexpected extension summary payload: %+v", extData)
	}

	// 3. Test Opportunity Detail & 8-Dimension Underdog Scoring
	firstDiscID := extData.LatestDiscoveries[0]["id"].(string)
	detResp, err := http.Get(server.URL + "/api/discoveries/" + firstDiscID)
	if err != nil || detResp.StatusCode != 200 {
		t.Fatalf("Discovery detail failed: %v", err)
	}
	var detData map[string]any
	_ = json.NewDecoder(detResp.Body).Decode(&detData)
	detResp.Body.Close()
	if detData["underdogAssessment"] == nil || detData["scoreRationale"] == nil {
		t.Fatalf("Missing underdogAssessment or scoreRationale in opportunity detail")
	}

	// 4. Test Documentary Project Workspace, 5-10s Visual Timeline & Character Consistency Sheets
	projectsResp, err := http.Get(server.URL + "/api/projects")
	if err != nil || projectsResp.StatusCode != 200 {
		t.Fatalf("Projects list failed: %v", err)
	}
	var projListData struct {
		Projects []map[string]any `json:"projects"`
	}
	_ = json.NewDecoder(projectsResp.Body).Decode(&projListData)
	projectsResp.Body.Close()

	var testProjectID string
	if len(projListData.Projects) > 0 {
		testProjectID = projListData.Projects[0]["id"].(string)
	} else {
		apprResp, errA := http.Post(server.URL+"/api/discoveries/"+firstDiscID+"/approve", "application/json", bytes.NewReader([]byte("{}")))
		if errA == nil && apprResp.StatusCode == 200 {
			var appData struct {
				ProjectID string `json:"projectId"`
			}
			_ = json.NewDecoder(apprResp.Body).Decode(&appData)
			apprResp.Body.Close()
			testProjectID = appData.ProjectID
		}
	}
	if testProjectID == "" {
		testProjectID = "proj-gukesh-world-champ"
	}

	wsResp, err := http.Get(server.URL + "/api/projects/" + testProjectID)
	if err != nil || wsResp.StatusCode != 200 {
		t.Fatalf("Project workspace failed: %v", err)
	}
	var wsData struct {
		CharacterReferences []map[string]any `json:"characterReferences"`
		VisualSegments      []map[string]any `json:"visualSegments"`
		AudioAssets         []map[string]any `json:"audioAssets"`
		SubtitleAssets      []map[string]any `json:"subtitleAssets"`
	}
	_ = json.NewDecoder(wsResp.Body).Decode(&wsData)
	wsResp.Body.Close()

	if len(wsData.CharacterReferences) < 3 {
		t.Fatalf("Expected >= 3 character consistency reference sheets, got %d", len(wsData.CharacterReferences))
	}
	if len(wsData.VisualSegments) < 10 {
		t.Fatalf("Expected >= 10 visual timeline segments, got %d", len(wsData.VisualSegments))
	}
	for _, seg := range wsData.VisualSegments {
		dur := seg["durationSeconds"].(float64)
		if dur < 5.0 || dur > 10.0 {
			t.Fatalf("Segment duration %f out of required 5-10s bounds", dur)
		}
	}
	if len(wsData.AudioAssets) == 0 || len(wsData.SubtitleAssets) == 0 {
		t.Fatalf("Expected master narration audio and SRT subtitle assets")
	}

	// 5. Test Complete Production Package Export + FFmpeg 16:9 Video Assembly
	expResp, err := http.Post(server.URL+"/api/projects/proj-gukesh-world-champ/assemble-video", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil || expResp.StatusCode != 200 {
		t.Fatalf("Export & FFmpeg video assembly failed: err=%v status=%d", err, expResp.StatusCode)
	}
	var expData struct {
		PackageZipURL string `json:"packageZipUrl"`
		VideoMp4URL   string `json:"videoMp4Url"`
		FileSizeBytes int64  `json:"fileSizeBytes"`
	}
	_ = json.NewDecoder(expResp.Body).Decode(&expData)
	expResp.Body.Close()

	if expData.PackageZipURL == "" || expData.FileSizeBytes < 1000 {
		t.Fatalf("Invalid ZIP package export result: %+v", expData)
	}
	if expData.VideoMp4URL == "" {
		t.Fatalf("Expected FFmpeg assembled 16:9 MP4 video URL")
	}

	// 6. Test Persistent Cloud Queue Idempotency & Execution
	idemKey := fmt.Sprintf("test-idem-%d", time.Now().UnixNano())
	j1, err := queue.EnqueueJob(ctx, "trend_monitoring_cycle", 9, idemKey, map[string]any{"test": true})
	if err != nil {
		t.Fatalf("EnqueueJob failed: %v", err)
	}
	j2, err := queue.EnqueueJob(ctx, "trend_monitoring_cycle", 9, idemKey, map[string]any{"test": true})
	if err != nil || j1 != j2 {
		t.Fatalf("Expected idempotent job IDs to match: %s vs %s", j1, j2)
	}
	processed := queue.GlobalQueue.ProcessNextJob(ctx)
	if !processed {
		t.Fatalf("Expected worker to process queued background job")
	}

	// 7. Verify Chrome Extension Manifest V3 files exist
	manifestCandidates := []string{
		"extension/manifest.json",
		"../extension/manifest.json",
		"../../extension/manifest.json",
		"../../../extension/manifest.json",
		filepath.Join(db.StorageRoot, "..", "extension", "manifest.json"),
	}
	var manifestFound bool
	for _, c := range manifestCandidates {
		if _, err := os.Stat(c); err == nil {
			manifestFound = true
			break
		}
	}
	if !manifestFound {
		t.Fatalf("Chrome extension manifest.json missing in workspace")
	}
}
