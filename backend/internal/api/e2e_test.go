package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"trendscout-backend/internal/agents"
	"trendscout-backend/internal/api"
	"trendscout-backend/internal/db"
	"trendscout-backend/internal/queue"
)

func authRequest(t *testing.T, method, url, token string, body []byte) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("Failed to build request %s %s: %v", method, url, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s failed: %v", method, url, err)
	}
	return resp
}

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

	// 1. Verify Unauthenticated & Expired JWT Rejection
	unauthResp := authRequest(t, http.MethodGet, server.URL+"/api/auth/me", "", nil)
	unauthResp.Body.Close()
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for unauthenticated /api/auth/me, got %d", unauthResp.StatusCode)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "trendscout-studio-production-secret-key-2026"
	}
	expiredTok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "usr-producer-01",
		"type": "access",
		"exp":  time.Now().Add(-5 * time.Minute).Unix(),
	}).SignedString([]byte(secret))

	expTokResp := authRequest(t, http.MethodGet, server.URL+"/api/dashboard/overview", expiredTok, nil)
	var expErrBody map[string]any
	_ = json.NewDecoder(expTokResp.Body).Decode(&expErrBody)
	expTokResp.Body.Close()
	if expTokResp.StatusCode != http.StatusUnauthorized || expErrBody["code"] != "TOKEN_EXPIRED" {
		t.Fatalf("Expected 401 TOKEN_EXPIRED for expired JWT, got status=%d body=%+v", expTokResp.StatusCode, expErrBody)
	}

	// 2. Test Login & JWT Refresh Session Flow
	loginBody, _ := json.Marshal(map[string]any{
		"email":      "producer@trendscout.studio",
		"password":   "trendscout2026",
		"rememberMe": true,
	})
	resp := authRequest(t, http.MethodPost, server.URL+"/api/auth/login", "", loginBody)
	if resp.StatusCode != 200 {
		t.Fatalf("Login failed: status=%d", resp.StatusCode)
	}
	var authRes struct {
		Token            string `json:"token"`
		RefreshToken     string `json:"refreshToken"`
		ExpiresAt        string `json:"expiresAt"`
		ExpiresIn        int64  `json:"expiresIn"`
		RefreshExpiresAt string `json:"refreshExpiresAt"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&authRes)
	resp.Body.Close()
	if authRes.Token == "" || authRes.RefreshToken == "" || authRes.ExpiresAt == "" || authRes.ExpiresIn <= 0 {
		t.Fatalf("Expected valid JWT session payload, got %+v", authRes)
	}

	// Refresh token must NOT work as an access token
	wrongTypeResp := authRequest(t, http.MethodGet, server.URL+"/api/auth/me", authRes.RefreshToken, nil)
	wrongTypeResp.Body.Close()
	if wrongTypeResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 when using refresh token as access token, got %d", wrongTypeResp.StatusCode)
	}

	// Refresh endpoint returns a fresh valid access token
	refBody, _ := json.Marshal(map[string]string{"refreshToken": authRes.RefreshToken})
	refResp := authRequest(t, http.MethodPost, server.URL+"/api/auth/refresh", "", refBody)
	if refResp.StatusCode != 200 {
		t.Fatalf("Token refresh failed: status=%d", refResp.StatusCode)
	}
	_ = json.NewDecoder(refResp.Body).Decode(&authRes)
	refResp.Body.Close()
	if authRes.Token == "" {
		t.Fatalf("Expected refreshed JWT token")
	}

	// Verify /api/auth/me returns active session metadata
	meResp := authRequest(t, http.MethodGet, server.URL+"/api/auth/me", authRes.Token, nil)
	var meData map[string]any
	_ = json.NewDecoder(meResp.Body).Decode(&meData)
	meResp.Body.Close()
	if meResp.StatusCode != 200 || meData["session"] == nil {
		t.Fatalf("Expected 200 + session metadata from /api/auth/me, got status=%d body=%+v", meResp.StatusCode, meData)
	}

	// 3. Verify Authenticated Live Discovery Scan Endpoint
	scanResp := authRequest(t, http.MethodPost, server.URL+"/api/discoveries/scan", authRes.Token, []byte(`{}`))
	if scanResp.StatusCode != 200 {
		t.Fatalf("Live discovery scan failed: status=%d", scanResp.StatusCode)
	}
	var scanData map[string]any
	_ = json.NewDecoder(scanResp.Body).Decode(&scanData)
	scanResp.Body.Close()
	t.Logf("Live scan telemetry: %+v", scanData)
	if okVal, _ := scanData["ok"].(bool); !okVal {
		t.Fatalf("Expected ok=true from /api/discoveries/scan, got %+v", scanData)
	}

	// 3B. Verify Suggest a Person & Pull Opportunity Endpoint (/api/discoveries/suggest)
	suggestBody, _ := json.Marshal(map[string]any{
		"personName": "Katalin Karikó",
		"focusAngle": "Decades of rejected mRNA grant applications before winning the Nobel Prize",
		"category":   "Science & Medicine",
	})
	suggestResp := authRequest(t, http.MethodPost, server.URL+"/api/discoveries/suggest", authRes.Token, suggestBody)
	if suggestResp.StatusCode != 200 {
		t.Fatalf("Suggest person endpoint failed: status=%d", suggestResp.StatusCode)
	}
	var suggestData map[string]any
	_ = json.NewDecoder(suggestResp.Body).Decode(&suggestData)
	suggestResp.Body.Close()
	if okVal, _ := suggestData["ok"].(bool); !okVal || suggestData["discoveryId"] == nil || suggestData["discoveryId"] == "" {
		t.Fatalf("Expected ok=true and discoveryId from /api/discoveries/suggest, got %+v", suggestData)
	}

	// 4. Test Chrome Extension Summary Endpoint (Authenticated)
	extResp := authRequest(t, http.MethodGet, server.URL+"/api/extension/summary", authRes.Token, nil)
	if extResp.StatusCode != 200 {
		t.Fatalf("Extension summary failed: status=%d", extResp.StatusCode)
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

	// 5. Test Opportunity Detail & 8-Dimension Underdog Scoring
	firstDiscID := extData.LatestDiscoveries[0]["id"].(string)
	detResp := authRequest(t, http.MethodGet, server.URL+"/api/discoveries/"+firstDiscID, authRes.Token, nil)
	if detResp.StatusCode != 200 {
		t.Fatalf("Discovery detail failed: status=%d", detResp.StatusCode)
	}
	var detData map[string]any
	_ = json.NewDecoder(detResp.Body).Decode(&detData)
	detResp.Body.Close()
	if detData["underdogAssessment"] == nil || detData["scoreRationale"] == nil {
		t.Fatalf("Missing underdogAssessment or scoreRationale in opportunity detail")
	}

	// 6. Test Documentary Project Workspace, <=7s Visual Timeline & Character Consistency Sheets
	projectsResp := authRequest(t, http.MethodGet, server.URL+"/api/projects", authRes.Token, nil)
	if projectsResp.StatusCode != 200 {
		t.Fatalf("Projects list failed: status=%d", projectsResp.StatusCode)
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
		apprResp := authRequest(t, http.MethodPost, server.URL+"/api/discoveries/"+firstDiscID+"/approve", authRes.Token, []byte("{}"))
		if apprResp.StatusCode == 200 {
			var appData struct {
				ProjectID string `json:"projectId"`
			}
			_ = json.NewDecoder(apprResp.Body).Decode(&appData)
			testProjectID = appData.ProjectID
		}
		apprResp.Body.Close()
	}

	wsResp := authRequest(t, http.MethodGet, server.URL+"/api/projects/"+testProjectID, authRes.Token, nil)
	if wsResp.StatusCode != 200 {
		t.Fatalf("Project workspace failed: status=%d", wsResp.StatusCode)
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
		if dur < 3.0 || dur > 7.0 {
			t.Fatalf("Segment duration %f out of required <=7s bounds (3-7s)", dur)
		}
	}
	if len(wsData.AudioAssets) == 0 || len(wsData.SubtitleAssets) == 0 {
		t.Fatalf("Expected master narration audio and SRT subtitle assets")
	}

	// 7. Test Complete Production Package Export + FFmpeg 16:9 Video Assembly
	expResp := authRequest(t, http.MethodPost, server.URL+"/api/projects/"+testProjectID+"/assemble-video", authRes.Token, []byte(`{}`))
	if expResp.StatusCode != 200 {
		t.Fatalf("Export & FFmpeg video assembly failed: status=%d", expResp.StatusCode)
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

	// 8. Test Persistent Cloud Queue Idempotency & Execution
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

	// 9. Verify Chrome Extension Manifest V3 files exist
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
