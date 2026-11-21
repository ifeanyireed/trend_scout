package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthAndSessionExpiry(t *testing.T) {
	protected := requireAuth(func(w http.ResponseWriter, r *http.Request) {
		info, _ := r.Context().Value(ctxAuthUserKey).(AuthClaimsInfo)
		writeJSON(w, http.StatusOK, map[string]any{
			"userId":    info.UserID,
			"email":     info.Email,
			"expiresAt": info.ExpiresAt.Format(time.RFC3339),
		})
	})

	// 1. Missing token -> 401 AUTH_REQUIRED
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	protected(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"] != "AUTH_REQUIRED" {
		t.Fatalf("expected AUTH_REQUIRED, got %+v", body)
	}

	// 2. Expired access token -> 401 TOKEN_EXPIRED
	expiredTok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "usr-producer-01",
		"type": "access",
		"exp":  time.Now().Add(-2 * time.Minute).Unix(),
	}).SignedString(jwtSecret())
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTok)
	protected(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", rec.Code)
	}
	body = map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["code"] != "TOKEN_EXPIRED" {
		t.Fatalf("expected TOKEN_EXPIRED, got %+v", body)
	}

	// 3. Valid IssueTokensWithTTL access token succeeds & refresh token rejected on protected routes
	accessTok, refreshTok, accessExp, refreshExp, err := IssueTokensWithTTL("usr-01", "producer@trendscout.studio", "Lead Producer", "admin", true)
	if err != nil {
		t.Fatalf("IssueTokensWithTTL failed: %v", err)
	}
	if !accessExp.After(time.Now()) || !refreshExp.After(accessExp) {
		t.Fatalf("unexpected token expiry windows: access=%v refresh=%v", accessExp, refreshExp)
	}

	// Refresh token rejected for API access
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer "+refreshTok)
	protected(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when using refresh token as access token, got %d", rec.Code)
	}

	// Valid access token succeeds
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/dashboard/overview", nil)
	req.Header.Set("Authorization", "Bearer "+accessTok)
	protected(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid access token, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Non-rememberMe refresh token has 24h TTL
	_, _, _, shortRefExp, err := IssueTokensWithTTL("usr-01", "producer@trendscout.studio", "Lead Producer", "admin", false)
	if err != nil {
		t.Fatalf("IssueTokensWithTTL(rememberMe=false) failed: %v", err)
	}
	if time.Until(shortRefExp) > 25*time.Hour {
		t.Fatalf("expected <=24h refresh TTL when rememberMe=false, got %v", time.Until(shortRefExp))
	}
}
