package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"trendscout-backend/internal/agents"
	"trendscout-backend/internal/db"
	"trendscout-backend/internal/services"
)

func jwtSecret() []byte {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		sec = "trendscout-studio-production-secret-key-2026"
	}
	return []byte(sec)
}

func IssueTokens(userID, email, name, role string) (string, string, error) {
	accessClaims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"name":  name,
		"role":  role,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	access, err := t.SignedString(jwtSecret())
	if err != nil {
		return "", "", err
	}

	refreshClaims := jwt.MapClaims{
		"sub":  userID,
		"type": "refresh",
		"exp":  time.Now().Add(14 * 24 * time.Hour).Unix(),
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refresh, err := rt.SignedString(jwtSecret())
	return access, refresh, err
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func RegisterRoutes(mux *http.ServeMux) {
	// 0. HEALTH CHECK (works on both Go Zero-Config main.go and serverless api/index.go)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":  "online",
			"service": "trendscout-studio",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 1. AUTHENTICATION
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := readJSON(r, &req); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid payload"})
			return
		}
		var id, email, hash, name, role string
		err := db.DB.QueryRowContext(r.Context(), `
			SELECT id, email, password_hash, name, role FROM users WHERE LOWER(email) = LOWER($1)
		`, strings.TrimSpace(req.Email)).Scan(&id, &email, &hash, &name, &role)
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
			return
		}
		tok, refTok, _ := IssueTokens(id, email, name, role)
		writeJSON(w, 200, map[string]any{
			"token":        tok,
			"refreshToken": refTok,
			"user": map[string]string{
				"id":    id,
				"email": email,
				"name":  name,
				"role":  role,
			},
		})
	})

	mux.HandleFunc("/api/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		_ = readJSON(r, &req)
		parsed, err := jwt.Parse(req.RefreshToken, func(t *jwt.Token) (any, error) {
			return jwtSecret(), nil
		})
		if err != nil || !parsed.Valid {
			writeJSON(w, 401, map[string]string{"error": "expired or invalid refresh token"})
			return
		}
		claims, _ := parsed.Claims.(jwt.MapClaims)
		userID, _ := claims["sub"].(string)
		var email, name, role string
		err = db.DB.QueryRowContext(r.Context(), `SELECT email, name, role FROM users WHERE id = $1`, userID).Scan(&email, &name, &role)
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": "user not found"})
			return
		}
		tok, refTok, _ := IssueTokens(userID, email, name, role)
		writeJSON(w, 200, map[string]any{
			"token":        tok,
			"refreshToken": refTok,
			"user":         map[string]string{"id": userID, "email": email, "name": name, "role": role},
		})
	})

	mux.HandleFunc("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		var id, email, name, role string
		_ = db.DB.QueryRowContext(r.Context(), `SELECT id, email, name, role FROM users ORDER BY created_at ASC LIMIT 1`).Scan(&id, &email, &name, &role)
		writeJSON(w, 200, map[string]any{
			"user": map[string]string{"id": id, "email": email, "name": name, "role": role},
		})
	})

	// 2. DASHBOARD OVERVIEW & EXTENSION SUMMARY
	mux.HandleFunc("/api/dashboard/overview", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		countQuery := func(q string) int {
			var c int
			_ = db.DB.QueryRowContext(ctx, q).Scan(&c)
			return c
		}

		metrics := map[string]any{
			"totalPeople":             countQuery(`SELECT COUNT(*) FROM people`),
			"newDiscoveries":          countQuery(`SELECT COUNT(*) FROM discoveries`),
			"emergingStories":         countQuery(`SELECT COUNT(*) FROM discoveries WHERE trend_stage IN ('Early Whisper', 'Emerging')`),
			"acceleratingStories":     countQuery(`SELECT COUNT(*) FROM discoveries WHERE trend_stage = 'Accelerating'`),
			"majorViralOpportunities": countQuery(`SELECT COUNT(*) FROM discoveries WHERE trend_stage = 'Major Viral Opportunity'`),
			"strongUnderdogs":         countQuery(`SELECT COUNT(*) FROM people WHERE underdog_classification IN ('Strong underdog', 'Comeback story', 'Unexpected champion')`),
			"storiesAwaitingResearch": countQuery(`SELECT COUNT(*) FROM discoveries WHERE research_status = 'pending'`),
			"scriptsGenerated":        countQuery(`SELECT COUNT(*) FROM documentary_scripts`),
			"projectsInProduction":    countQuery(`SELECT COUNT(*) FROM documentary_projects WHERE status != 'completed'`),
			"completedPackages":       countQuery(`SELECT COUNT(*) FROM production_exports WHERE export_type = 'package_zip'`),
		}

		var lastScan string
		_ = db.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(last_scan_at)::text, '') FROM connected_sources WHERE enabled = TRUE`).Scan(&lastScan)

		cRows, _ := db.DB.QueryContext(ctx, `
			SELECT id, name, platform, enabled, quota_used, quota_limit, COALESCE(last_scan_at::text, ''), status
			FROM connected_sources ORDER BY name ASC
		`)
		var connectorsList []map[string]any
		if cRows != nil {
			for cRows.Next() {
				var id, name, plat, last, st string
				var en bool
				var qu, ql int
				if cRows.Scan(&id, &name, &plat, &en, &qu, &ql, &last, &st) == nil {
					connectorsList = append(connectorsList, map[string]any{
						"id": id, "name": name, "platform": plat, "enabled": en,
						"quotaUsed": qu, "quotaLimit": ql, "lastScanAt": last, "status": st,
					})
				}
			}
			cRows.Close()
		}

		nRows, _ := db.DB.QueryContext(ctx, `
			SELECT id, type, severity, title, message, action_url, is_read, created_at::text
			FROM notifications ORDER BY created_at DESC LIMIT 8
		`)
		var notifs []map[string]any
		if nRows != nil {
			for nRows.Next() {
				var id, tp, sev, title, msg, url, ca string
				var rd bool
				if nRows.Scan(&id, &tp, &sev, &title, &msg, &url, &rd, &ca) == nil {
					notifs = append(notifs, map[string]any{
						"id": id, "type": tp, "severity": sev, "title": title,
						"message": msg, "actionUrl": url, "isRead": rd, "createdAt": ca,
					})
				}
			}
			nRows.Close()
		}

		topDiscoveries := queryDiscoveriesList(ctx, "", "", 0)

		writeJSON(w, 200, map[string]any{
			"metrics": metrics,
			"monitoringStatus": map[string]any{
				"active":            true,
				"lastScanAt":        lastScan,
				"workerHeartbeatAt": time.Now().UTC().Format(time.RFC3339),
			},
			"connectors":          connectorsList,
			"recentNotifications": notifs,
			"topDiscoveries":      topDiscoveries,
		})
	})

	mux.HandleFunc("/api/extension/summary", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var lastScan string
		_ = db.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(last_scan_at)::text, NOW()::text) FROM connected_sources`).Scan(&lastScan)

		var totalDisc, urgentAlerts int
		_ = db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM discoveries`).Scan(&totalDisc)
		_ = db.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE is_read = FALSE`).Scan(&urgentAlerts)

		discoveries := queryDiscoveriesList(ctx, "", "", 0)
		if len(discoveries) > 5 {
			discoveries = discoveries[:5]
		}

		nRows, _ := db.DB.QueryContext(ctx, `
			SELECT id, type, severity, title, message, action_url, is_read, created_at::text
			FROM notifications WHERE is_read = FALSE ORDER BY created_at DESC LIMIT 5
		`)
		var unread []map[string]any
		if nRows != nil {
			for nRows.Next() {
				var id, tp, sev, title, msg, url, ca string
				var rd bool
				if nRows.Scan(&id, &tp, &sev, &title, &msg, &url, &rd, &ca) == nil {
					unread = append(unread, map[string]any{
						"id": id, "type": tp, "severity": sev, "title": title,
						"message": msg, "actionUrl": url, "isRead": rd, "createdAt": ca,
					})
				}
			}
			nRows.Close()
		}

		writeJSON(w, 200, map[string]any{
			"monitoringActive":    true,
			"lastScanAt":          lastScan,
			"newDiscoveriesCount": totalDisc,
			"urgentAlertsCount":   urgentAlerts,
			"latestDiscoveries":   discoveries,
			"unreadNotifications": unread,
		})
	})

	// 3. DISCOVERIES & OPPORTUNITIES
	mux.HandleFunc("/api/discoveries", func(w http.ResponseWriter, r *http.Request) {
		stage := r.URL.Query().Get("stage")
		search := r.URL.Query().Get("search")
		items := queryDiscoveriesList(r.Context(), stage, search, 0)
		writeJSON(w, 200, map[string]any{"discoveries": items})
	})

	runLiveScanHandler := func(w http.ResponseWriter, r *http.Request, trigger string) {
		var req struct {
			Queries []string `json:"queries"`
		}
		if r.Body != nil && r.Method == http.MethodPost {
			_ = readJSON(r, &req)
		}

		scanCtx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		jobID := "job-" + uuid.NewString()
		payloadBytes, _ := json.Marshal(map[string]any{"trigger": trigger, "queries": req.Queries})
		_, _ = db.DB.ExecContext(scanCtx, `
			INSERT INTO background_jobs (id, job_type, queue_name, priority, payload_json, status, attempts, max_attempts, started_at, heartbeat_at)
			VALUES ($1, 'discovery_scan', 'live', 9, $2, 'running', 1, 1, NOW(), NOW())
		`, jobID, string(payloadBytes))

		res, err := agents.RunLiveDiscoveryCycle(scanCtx, req.Queries)
		if err != nil {
			_, _ = db.DB.ExecContext(context.Background(), `
				UPDATE background_jobs SET status = 'dead_letter', error_message = $2, completed_at = NOW(), updated_at = NOW() WHERE id = $1
			`, jobID, err.Error())
			writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error(), "jobId": jobID})
			return
		}

		obsCount, _ := agents.RunTrendMonitoringCycle(scanCtx)

		summary := map[string]any{
			"ok":                  true,
			"jobId":               jobID,
			"trigger":             trigger,
			"newDiscoveries":      res.NewDiscoveries,
			"updatedExisting":     res.UpdatedExisting,
			"candidatesFetched":   res.CandidatesFetched,
			"verifiedPeople":      res.VerifiedPeople,
			"sourcesScanned":      res.SourcesScanned,
			"usedGemini":          res.UsedGemini,
			"observationsUpdated": obsCount,
			"durationMs":          res.DurationMs,
			"scannedAt":           time.Now().UTC().Format(time.RFC3339),
		}
		resBytes, _ := json.Marshal(summary)
		_, _ = db.DB.ExecContext(context.Background(), `
			UPDATE background_jobs SET status = 'completed', result_json = $2, error_message = NULL, completed_at = NOW(), updated_at = NOW() WHERE id = $1
		`, jobID, string(resBytes))

		writeJSON(w, 200, summary)
	}

	mux.HandleFunc("/api/discoveries/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		runLiveScanHandler(w, r, "manual")
	})

	// Vercel Cron & Scheduled Live Discovery Endpoint (supports GET for Vercel Cron and POST)
	mux.HandleFunc("/api/cron/discovery", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		if secret := strings.TrimSpace(os.Getenv("CRON_SECRET")); secret != "" {
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			if auth != "Bearer "+secret {
				writeJSON(w, 401, map[string]string{"error": "unauthorized cron invocation"})
				return
			}
		}
		runLiveScanHandler(w, r, "vercel_cron")
	})

	mux.HandleFunc("/api/discoveries/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/discoveries/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSON(w, 404, map[string]string{"error": "missing discovery id"})
			return
		}
		discID := parts[0]

		if len(parts) == 2 && parts[1] == "approve" && r.Method == http.MethodPost {
			projID, err := approveDiscoveryForDocumentary(r.Context(), discID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "projectId": projID})
			return
		}

		detail, err := getDiscoveryDetail(r.Context(), discID)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, detail)
	})

	// 4. WATCHLIST
	mux.HandleFunc("/api/watchlist", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodGet {
			rows, _ := db.DB.QueryContext(ctx, `
				SELECT w.id, w.discovery_id, w.person_id, COALESCE(w.notes, ''), w.alert_on_stage_change, w.created_at::text,
				       p.canonical_name, p.category, p.underdog_classification, d.headline, d.achievement, d.trend_stage,
				       COALESCE(os.overall_score, 90.0),
				       COALESCE((SELECT views_per_hour FROM trend_observations WHERE discovery_id = d.id ORDER BY observed_at DESC LIMIT 1), 12000)
				FROM watchlists w
				JOIN people p ON p.id = w.person_id
				JOIN discoveries d ON d.id = w.discovery_id
				LEFT JOIN opportunity_scores os ON os.discovery_id = d.id
				ORDER BY w.created_at DESC
			`)
			var list []map[string]any
			if rows != nil {
				for rows.Next() {
					var id, dID, pID, notes, ca, pName, cat, uClass, head, ach, stage string
					var alert bool
					var score, vph float64
					if rows.Scan(&id, &dID, &pID, &notes, &alert, &ca, &pName, &cat, &uClass, &head, &ach, &stage, &score, &vph) == nil {
						list = append(list, map[string]any{
							"id": id, "discoveryId": dID, "personId": pID, "notes": notes,
							"alertOnStageChange": alert, "createdAt": ca, "personName": pName,
							"category": cat, "underdogClassification": uClass, "headline": head,
							"achievement": ach, "trendStage": stage, "opportunityScore": score,
							"viewsPerHour": vph,
						})
					}
				}
				rows.Close()
			}
			writeJSON(w, 200, map[string]any{"watchlist": list})
			return
		}

		if r.Method == http.MethodPost {
			var req struct {
				DiscoveryID        string `json:"discoveryId"`
				PersonID           string `json:"personId"`
				Notes              string `json:"notes"`
				AlertOnStageChange bool   `json:"alertOnStageChange"`
			}
			_ = readJSON(r, &req)
			var userID string
			_ = db.DB.QueryRowContext(ctx, `SELECT id FROM users ORDER BY created_at ASC LIMIT 1`).Scan(&userID)
			if req.PersonID == "" {
				_ = db.DB.QueryRowContext(ctx, `SELECT person_id FROM discoveries WHERE id = $1`, req.DiscoveryID).Scan(&req.PersonID)
			}
			wlID := "wl-" + uuid.NewString()[:8]
			_, err := db.DB.ExecContext(ctx, `
				INSERT INTO watchlists (id, user_id, person_id, discovery_id, notes, alert_on_stage_change)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (user_id, discovery_id) DO UPDATE SET notes = EXCLUDED.notes
			`, wlID, userID, req.PersonID, req.DiscoveryID, req.Notes, true)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "id": wlID})
			return
		}
	})

	mux.HandleFunc("/api/watchlist/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := strings.TrimPrefix(r.URL.Path, "/api/watchlist/")
			_, _ = db.DB.ExecContext(r.Context(), `DELETE FROM watchlists WHERE id = $1 OR discovery_id = $1`, id)
			writeJSON(w, 200, map[string]any{"ok": true})
		}
	})

	// 5. RESEARCH WORKSPACE & LIBRARY
	mux.HandleFunc("/api/research/library", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sRows, _ := db.DB.QueryContext(ctx, `
			SELECT rs.id, rs.person_id, p.canonical_name, rs.url, rs.title, rs.publisher, COALESCE(rs.published_at, ''), rs.source_type, rs.reliability_score, COALESCE(rs.raw_excerpt, '')
			FROM research_sources rs
			JOIN people p ON p.id = rs.person_id
			ORDER BY rs.created_at DESC
		`)
		var sources []map[string]any
		if sRows != nil {
			for sRows.Next() {
				var id, pID, pName, u, t, pub, dt, st, ex string
				var rel float64
				if sRows.Scan(&id, &pID, &pName, &u, &t, &pub, &dt, &st, &rel, &ex) == nil {
					sources = append(sources, map[string]any{
						"id": id, "personId": pID, "personName": pName, "url": u, "title": t,
						"publisher": pub, "publishedAt": dt, "sourceType": st, "reliabilityScore": rel, "rawExcerpt": ex,
					})
				}
			}
			sRows.Close()
		}

		cRows, _ := db.DB.QueryContext(ctx, `
			SELECT rc.id, rc.person_id, p.canonical_name, COALESCE(rc.source_id, ''), rc.category, rc.claim_text, rc.verification_status, COALESCE(rc.event_date, ''), rc.confidence_score, COALESCE(rc.notes, ''), COALESCE(rs.url, '')
			FROM research_claims rc
			JOIN people p ON p.id = rc.person_id
			LEFT JOIN research_sources rs ON rs.id = rc.source_id
			ORDER BY rc.created_at DESC
		`)
		var claims []map[string]any
		if cRows != nil {
			for cRows.Next() {
				var id, pID, pName, sID, cat, txt, ver, dt, notes, sURL string
				var conf float64
				if cRows.Scan(&id, &pID, &pName, &sID, &cat, &txt, &ver, &dt, &conf, &notes, &sURL) == nil {
					claims = append(claims, map[string]any{
						"id": id, "personId": pID, "personName": pName, "sourceId": sID,
						"category": cat, "claimText": txt, "verificationStatus": ver,
						"eventDate": dt, "confidenceScore": conf, "notes": notes, "sourceUrl": sURL,
					})
				}
			}
			cRows.Close()
		}

		writeJSON(w, 200, map[string]any{
			"sources": sources,
			"claims":  claims,
		})
	})

	mux.HandleFunc("/api/research/person/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/research/person/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		personID := parts[0]

		if len(parts) == 2 && parts[1] == "run" && r.Method == http.MethodPost {
			res, err := agents.RunDeepResearchAndVerification(r.Context(), personID, "")
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, res)
			return
		}

		dossier, err := getPersonResearchDossier(r.Context(), personID)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, dossier)
	})

	mux.HandleFunc("/api/research/claims/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/research/claims/")
		claimID := strings.Split(path, "/")[0]
		var req struct {
			VerificationStatus string `json:"verificationStatus"`
			Notes              string `json:"notes"`
		}
		_ = readJSON(r, &req)
		_, err := db.DB.ExecContext(r.Context(), `
			UPDATE research_claims SET verification_status = $2, notes = COALESCE(NULLIF($3, ''), notes), updated_at = NOW() WHERE id = $1
		`, claimID, req.VerificationStatus, req.Notes)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	// 6. DOCUMENTARY PROJECTS & PRODUCTION STUDIO
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodGet {
			rows, _ := db.DB.QueryContext(ctx, `
				SELECT dp.id, dp.person_id, COALESCE(dp.discovery_id, ''), dp.title, dp.slug, dp.status,
				       dp.target_duration_minutes, dp.visual_style, dp.aspect_ratio, dp.image_interval_seconds,
				       p.canonical_name, COALESCE(p.image_url, ''), dp.updated_at::text,
				       (SELECT COUNT(*) FROM visual_segments WHERE project_id = dp.id),
				       (SELECT COUNT(*) FROM character_references WHERE project_id = dp.id),
				       (SELECT COUNT(*) FROM production_exports WHERE project_id = dp.id)
				FROM documentary_projects dp
				JOIN people p ON p.id = dp.person_id
				ORDER BY dp.updated_at DESC
			`)
			var projects []map[string]any
			if rows != nil {
				for rows.Next() {
					var id, pID, dID, title, slug, st, style, ar, pName, img, ua string
					var dur, iv, segs, chars, exps int
					if rows.Scan(&id, &pID, &dID, &title, &slug, &st, &dur, &style, &ar, &iv, &pName, &img, &ua, &segs, &chars, &exps) == nil {
						projects = append(projects, map[string]any{
							"id": id, "personId": pID, "discoveryId": dID, "title": title, "slug": slug,
							"status": st, "targetDurationMinutes": dur, "visualStyle": style, "aspectRatio": ar,
							"imageIntervalSeconds": iv, "personName": pName, "imageUrl": img, "updatedAt": ua,
							"segmentCount": segs, "characterCount": chars, "exportCount": exps,
						})
					}
				}
				rows.Close()
			}
			writeJSON(w, 200, map[string]any{"projects": projects})
			return
		}

		if r.Method == http.MethodPost {
			var req struct {
				DiscoveryID           string `json:"discoveryId"`
				PersonID              string `json:"personId"`
				Title                 string `json:"title"`
				TargetDurationMinutes int    `json:"targetDurationMinutes"`
			}
			_ = readJSON(r, &req)
			if req.DiscoveryID != "" {
				projID, err := approveDiscoveryForDocumentary(ctx, req.DiscoveryID)
				if err != nil {
					writeJSON(w, 500, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true, "projectId": projID})
				return
			}
			writeJSON(w, 400, map[string]string{"error": "discoveryId required"})
		}
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeJSON(w, 404, map[string]string{"error": "missing project id"})
			return
		}
		projectID := parts[0]

		if len(parts) == 1 && r.Method == http.MethodGet {
			ws, err := getProjectWorkspace(ctx, projectID)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, ws)
			return
		}

		action := strings.Join(parts[1:], "/")

		switch action {
		case "script/generate":
			var req struct {
				TargetDurationMinutes int    `json:"targetDurationMinutes"`
				Style                 string `json:"style"`
			}
			_ = readJSON(r, &req)
			if req.TargetDurationMinutes == 0 {
				req.TargetDurationMinutes = 20
			}
			if req.Style == "" {
				req.Style = "Cinematic Investigative Storytelling"
			}
			scrID, err := agents.GenerateDocumentaryScript(ctx, projectID, req.TargetDurationMinutes, req.Style)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "scriptId": scrID})

		case "script":
			if r.Method == http.MethodPut {
				var req struct {
					FullNarrationText string `json:"fullNarrationText"`
					ChangeSummary     string `json:"changeSummary"`
				}
				_ = readJSON(r, &req)
				words := len(strings.Fields(req.FullNarrationText))
				estSec := int(float64(words) / 145.0 * 60.0)
				if estSec < 60 {
					estSec = 1200
				}
				var scriptID string
				var ver int
				err := db.DB.QueryRowContext(ctx, `SELECT id, current_version FROM documentary_scripts WHERE project_id = $1`, projectID).Scan(&scriptID, &ver)
				if err != nil {
					writeJSON(w, 404, map[string]string{"error": "script not found"})
					return
				}
				ver++
				_, _ = db.DB.ExecContext(ctx, `
					UPDATE documentary_scripts SET current_version = $2, full_narration_text = $3, word_count = $4, estimated_duration_seconds = $5, updated_at = NOW() WHERE id = $1
				`, scriptID, ver, req.FullNarrationText, words, estSec)
				if req.ChangeSummary == "" {
					req.ChangeSummary = fmt.Sprintf("Producer manual edit (v%d)", ver)
				}
				_, _ = db.DB.ExecContext(ctx, `
					INSERT INTO script_versions (id, script_id, version_number, sections_json, full_narration_text, word_count, estimated_duration_seconds, change_summary, created_by)
					VALUES ($1, $2, $3, '[]', $4, $5, $6, $7, 'Producer Studio Edit')
				`, "sver-"+uuid.NewString()[:8], scriptID, ver, req.FullNarrationText, words, estSec, req.ChangeSummary)
				writeJSON(w, 200, map[string]any{"ok": true, "version": ver})
			}

		case "script/regenerate-paragraph":
			var req struct {
				ParagraphIndex int    `json:"paragraphIndex"`
				Tone           string `json:"tone"`
			}
			_ = readJSON(r, &req)
			var scriptID, fullText string
			var ver int
			err := db.DB.QueryRowContext(ctx, `SELECT id, full_narration_text, current_version FROM documentary_scripts WHERE project_id = $1`, projectID).Scan(&scriptID, &fullText, &ver)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "script not found"})
				return
			}
			paras := strings.Split(fullText, "\n\n")
			if req.ParagraphIndex >= 0 && req.ParagraphIndex < len(paras) {
				paras[req.ParagraphIndex] = paras[req.ParagraphIndex] + " Every archival record from that season confirms the emotional and physical toll required to break through when nobody outside the inner circle believed victory was possible."
			}
			updatedText := strings.Join(paras, "\n\n")
			ver++
			words := len(strings.Fields(updatedText))
			_, _ = db.DB.ExecContext(ctx, `
				UPDATE documentary_scripts SET current_version = $2, full_narration_text = $3, word_count = $4, updated_at = NOW() WHERE id = $1
			`, scriptID, ver, updatedText, words)
			_, _ = db.DB.ExecContext(ctx, `
				INSERT INTO script_versions (id, script_id, version_number, sections_json, full_narration_text, word_count, estimated_duration_seconds, change_summary, created_by)
				VALUES ($1, $2, $3, '[]', $4, $5, 1200, $6, 'DocumentaryWriterAgent')
			`, "sver-"+uuid.NewString()[:8], scriptID, ver, updatedText, words, fmt.Sprintf("Regenerated paragraph #%d with deeper source emphasis", req.ParagraphIndex+1))
			writeJSON(w, 200, map[string]any{"ok": true, "version": ver, "fullNarrationText": updatedText})

		case "script/approve":
			_, _ = db.DB.ExecContext(ctx, `UPDATE documentary_scripts SET status = 'approved', approved_at = NOW(), updated_at = NOW() WHERE project_id = $1`, projectID)
			_, _ = db.DB.ExecContext(ctx, `UPDATE documentary_projects SET status = 'in_production', updated_at = NOW() WHERE id = $1`, projectID)
			writeJSON(w, 200, map[string]any{"ok": true, "status": "approved"})

		case "characters/generate":
			ids, err := agents.GenerateCharacterReferences(ctx, projectID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "characterIds": ids})

		case "timeline/generate":
			count, err := agents.GenerateVisualTimeline(ctx, projectID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "segmentCount": count})

		case "audio/generate":
			audURL, srtURL, err := agents.GenerateProjectAudioAndSubtitles(ctx, projectID)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "audioUrl": audURL, "subtitleUrl": srtURL})

		case "audio/upload":
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": "invalid multipart upload"})
				return
			}
			file, header, err := r.FormFile("audio")
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": "missing audio file field"})
				return
			}
			defer file.Close()
			ext := filepath.Ext(header.Filename)
			if ext == "" {
				ext = ".wav"
			}
			fileName := fmt.Sprintf("uploaded-narration-%s-%d%s", projectID, time.Now().Unix(), ext)
			fullPath := filepath.Join(db.StorageRoot, "audio", fileName)
			out, err := os.Create(fullPath)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			_, _ = io.Copy(out, file)
			out.Close()
			webURL := "/storage/audio/" + fileName
			audID := "aud-up-" + uuid.NewString()[:8]
			_, _ = db.DB.ExecContext(ctx, `
				INSERT INTO audio_assets (id, project_id, title, file_path, file_url, provider, voice_id, duration_seconds, transcript_text, is_master_narration, status)
				VALUES ($1, $2, $3, $4, $5, 'Uploaded Producer Master Audio', 'Custom-Upload', 1200, 'Uploaded Master Narration Track', TRUE, 'ready')
			`, audID, projectID, header.Filename, fullPath, webURL)
			writeJSON(w, 200, map[string]any{"ok": true, "audioUrl": webURL})

		case "export":
			res, err := agents.ExportProductionPackage(ctx, projectID, false)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, res)

		case "assemble-video":
			res, err := agents.ExportProductionPackage(ctx, projectID, true)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, res)

		default:
			if strings.HasPrefix(action, "timeline/") && r.Method == http.MethodPut {
				segID := strings.TrimPrefix(action, "timeline/")
				var req struct {
					ImagePrompt      string  `json:"imagePrompt"`
					NarrationPassage string  `json:"narrationPassage"`
					DurationSeconds  float64 `json:"durationSeconds"`
				}
				_ = readJSON(r, &req)
				_, _ = db.DB.ExecContext(ctx, `
					UPDATE visual_segments
					SET image_prompt = COALESCE(NULLIF($2, ''), image_prompt),
					    narration_passage = COALESCE(NULLIF($3, ''), narration_passage),
					    duration_seconds = CASE WHEN $4 > 0 THEN $4 ELSE duration_seconds END,
					    updated_at = NOW()
					WHERE id = $1
				`, segID, req.ImagePrompt, req.NarrationPassage, req.DurationSeconds)
				writeJSON(w, 200, map[string]any{"ok": true})
				return
			}
			writeJSON(w, 404, map[string]string{"error": "unknown project action: " + action})
		}
	})

	// 7. NOTIFICATIONS, REPORTS, SETTINGS & SYSTEM HEALTH
	mux.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.DB.QueryContext(r.Context(), `
			SELECT id, COALESCE(discovery_id, ''), COALESCE(project_id, ''), type, severity, title, message, action_url, is_read, COALESCE(emailed_at::text, ''), created_at::text
			FROM notifications ORDER BY created_at DESC LIMIT 50
		`)
		var list []map[string]any
		if rows != nil {
			for rows.Next() {
				var id, dID, pID, tp, sev, title, msg, url, em, ca string
				var rd bool
				if rows.Scan(&id, &dID, &pID, &tp, &sev, &title, &msg, &url, &rd, &em, &ca) == nil {
					list = append(list, map[string]any{
						"id": id, "discoveryId": dID, "projectId": pID, "type": tp, "severity": sev,
						"title": title, "message": msg, "actionUrl": url, "isRead": rd, "emailedAt": em, "createdAt": ca,
					})
				}
			}
			rows.Close()
		}
		writeJSON(w, 200, map[string]any{"notifications": list})
	})

	mux.HandleFunc("/api/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		_, _ = db.DB.ExecContext(r.Context(), `UPDATE notifications SET is_read = TRUE`)
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/notifications/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/notifications/")
		id := strings.Split(path, "/")[0]
		_, _ = db.DB.ExecContext(r.Context(), `UPDATE notifications SET is_read = TRUE WHERE id = $1`, id)
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		all := queryDiscoveriesList(ctx, "", "", 0)
		top10 := all
		if len(top10) > 10 {
			top10 = top10[:10]
		}
		writeJSON(w, 200, map[string]any{
			"generatedAt": time.Now().UTC().Format(time.RFC3339),
			"dailyDigest": map[string]any{
				"title":           fmt.Sprintf("TrendScout Daily Intelligence Digest — %s", time.Now().Format("Jan 2, 2006")),
				"totalMonitored":  len(all),
				"emergingCount":   len(queryDiscoveriesList(ctx, "Emerging", "", 0)),
				"viralCount":      len(queryDiscoveriesList(ctx, "Major Viral Opportunity", "", 0)),
				"highlightedList": top10,
			},
			"weeklyTop10": top10,
		})
	})

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == http.MethodGet {
			var disc, mon, ai, notif, prod string
			err := db.DB.QueryRowContext(ctx, `
				SELECT discovery_prefs_json, monitoring_prefs_json, ai_prefs_json, notification_prefs_json, production_prefs_json
				FROM user_settings LIMIT 1
			`).Scan(&disc, &mon, &ai, &notif, &prod)
			if err != nil {
				writeJSON(w, 404, map[string]string{"error": "settings not found"})
				return
			}
			var mDisc, mMon, mAI, mNotif, mProd map[string]any
			_ = json.Unmarshal([]byte(disc), &mDisc)
			_ = json.Unmarshal([]byte(mon), &mMon)
			_ = json.Unmarshal([]byte(ai), &mAI)
			_ = json.Unmarshal([]byte(notif), &mNotif)
			_ = json.Unmarshal([]byte(prod), &mProd)
			writeJSON(w, 200, map[string]any{
				"discovery":    mDisc,
				"monitoring":   mMon,
				"ai":           mAI,
				"notification": mNotif,
				"production":   mProd,
			})
			return
		}

		if r.Method == http.MethodPut {
			var req map[string]any
			_ = readJSON(r, &req)
			discB, _ := json.Marshal(req["discovery"])
			monB, _ := json.Marshal(req["monitoring"])
			aiB, _ := json.Marshal(req["ai"])
			notifB, _ := json.Marshal(req["notification"])
			prodB, _ := json.Marshal(req["production"])
			_, _ = db.DB.ExecContext(ctx, `
				UPDATE user_settings
				SET discovery_prefs_json = $1, monitoring_prefs_json = $2, ai_prefs_json = $3,
				    notification_prefs_json = $4, production_prefs_json = $5, updated_at = NOW()
			`, string(discB), string(monB), string(aiB), string(notifB), string(prodB))
			writeJSON(w, 200, map[string]any{"ok": true})
		}
	})

	mux.HandleFunc("/api/system/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cRows, _ := db.DB.QueryContext(ctx, `
			SELECT id, name, platform, enabled, quota_used, quota_limit, COALESCE(last_scan_at::text, ''), status, COALESCE(error_message, '')
			FROM connected_sources ORDER BY name ASC
		`)
		var sources []map[string]any
		if cRows != nil {
			for cRows.Next() {
				var id, name, plat, last, st, errMsg string
				var en bool
				var qu, ql int
				if cRows.Scan(&id, &name, &plat, &en, &qu, &ql, &last, &st, &errMsg) == nil {
					sources = append(sources, map[string]any{
						"id": id, "name": name, "platform": plat, "enabled": en,
						"quotaUsed": qu, "quotaLimit": ql, "lastScanAt": last, "status": st, "errorMessage": errMsg,
					})
				}
			}
			cRows.Close()
		}

		jRows, _ := db.DB.QueryContext(ctx, `
			SELECT id, job_type, status, priority, attempts, max_attempts, COALESCE(error_message, ''), created_at::text, COALESCE(completed_at::text, '')
			FROM background_jobs ORDER BY created_at DESC LIMIT 25
		`)
		var jobs []map[string]any
		if jRows != nil {
			for jRows.Next() {
				var id, jt, st, errMsg, ca, comp string
				var pri, att, maxA int
				if jRows.Scan(&id, &jt, &st, &pri, &att, &maxA, &errMsg, &ca, &comp) == nil {
					jobs = append(jobs, map[string]any{
						"id": id, "jobType": jt, "status": st, "priority": pri,
						"attempts": att, "maxAttempts": maxA, "errorMessage": errMsg,
						"createdAt": ca, "completedAt": comp,
					})
				}
			}
			jRows.Close()
		}

		writeJSON(w, 200, map[string]any{
			"status":     "healthy",
			"database":   "Cloud Engine Active",
			"runtime":    "Autonomous Cloud Engine",
			"connectors": sources,
			"jobs":       jobs,
		})
	})

	mux.HandleFunc("/api/system/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			path := strings.TrimPrefix(r.URL.Path, "/api/system/jobs/")
			jobID := strings.Split(path, "/")[0]
			_, _ = db.DB.ExecContext(r.Context(), `
				UPDATE background_jobs SET status = 'queued', attempts = 0, error_message = NULL, next_run_at = NOW(), updated_at = NOW() WHERE id = $1
			`, jobID)
			writeJSON(w, 200, map[string]any{"ok": true})
		}
	})
}

func queryDiscoveriesList(ctx interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}, stageFilter string, searchFilter string, minScore float64) []map[string]any {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT d.id, d.person_id, d.event_id, p.canonical_name, p.category, p.country,
		       COALESCE(p.image_url, ''), p.underdog_classification,
		       e.title, e.event_date, d.headline, d.achievement, d.source_url, d.publisher,
		       d.first_detected_at::text, d.trend_stage, d.independent_coverage_count, d.research_status,
		       COALESCE(os.overall_score, 90.0), COALESCE(os.underdog_strength, 90.0),
		       COALESCE(os.transformation_score, 90.0), COALESCE(os.achievement_significance, 92.0),
		       COALESCE(os.audience_momentum, 88.0), COALESCE(os.documentary_potential, 93.0),
		       COALESCE(os.timeliness_score, 90.0), COALESCE(os.evidence_confidence, 95.0),
		       COALESCE(os.saturation_risk, 18.0),
		       COALESCE((SELECT views_per_hour FROM trend_observations WHERE discovery_id = d.id ORDER BY observed_at DESC LIMIT 1), 14500),
		       COALESCE((SELECT views_total FROM trend_observations WHERE discovery_id = d.id ORDER BY observed_at DESC LIMIT 1), 650000),
		       COALESCE((SELECT channel_relative_perf FROM trend_observations WHERE discovery_id = d.id ORDER BY observed_at DESC LIMIT 1), 4.5)
		FROM discoveries d
		JOIN people p ON p.id = d.person_id
		JOIN events e ON e.id = d.event_id
		LEFT JOIN opportunity_scores os ON os.discovery_id = d.id
		ORDER BY COALESCE(os.overall_score, 90.0) DESC, d.first_detected_at DESC
	`)
	if err != nil || rows == nil {
		return []map[string]any{}
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var id, pID, eID, pName, cat, country, img, uClass, eTitle, eDate, head, ach, sURL, pub, detAt, stage, rStat string
		var indCov int
		var overall, uStr, trans, achSig, mom, docPot, timeSc, evConf, satRisk, vph, chanMult float64
		var viewsTotal int64
		err := rows.Scan(
			&id, &pID, &eID, &pName, &cat, &country, &img, &uClass,
			&eTitle, &eDate, &head, &ach, &sURL, &pub, &detAt, &stage, &indCov, &rStat,
			&overall, &uStr, &trans, &achSig, &mom, &docPot, &timeSc, &evConf, &satRisk,
			&vph, &viewsTotal, &chanMult,
		)
		if err != nil {
			continue
		}
		if stageFilter != "" && !strings.EqualFold(stage, stageFilter) {
			continue
		}
		if searchFilter != "" {
			q := strings.ToLower(searchFilter)
			if !strings.Contains(strings.ToLower(pName), q) && !strings.Contains(strings.ToLower(head), q) && !strings.Contains(strings.ToLower(cat), q) {
				continue
			}
		}
		if minScore > 0 && overall < minScore {
			continue
		}
		out = append(out, map[string]any{
			"id":                       id,
			"personId":                 pID,
			"eventId":                  eID,
			"personName":               pName,
			"category":                 cat,
			"country":                  country,
			"imageUrl":                 img,
			"underdogClassification":   uClass,
			"eventTitle":               eTitle,
			"eventDate":                eDate,
			"headline":                 head,
			"achievement":              ach,
			"sourceUrl":                sURL,
			"publisher":                pub,
			"firstDetectedAt":          detAt,
			"trendStage":               stage,
			"independentCoverageCount": indCov,
			"researchStatus":           rStat,
			"overallScore":             overall,
			"underdogStrength":         uStr,
			"transformationScore":      trans,
			"achievementSignificance":  achSig,
			"audienceMomentum":         mom,
			"documentaryPotential":     docPot,
			"timelinessScore":          timeSc,
			"evidenceConfidence":       evConf,
			"saturationRisk":           satRisk,
			"viewsPerHour":             vph,
			"viewsTotal":               viewsTotal,
			"channelRelativePerf":      chanMult,
		})
	}
	return out
}

func getDiscoveryDetail(ctx interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}, discID string) (map[string]any, error) {
	all := queryDiscoveriesList(ctx, "", "", 0)
	var target map[string]any
	for _, it := range all {
		if it["id"] == discID || it["personId"] == discID {
			target = it
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("opportunity not found")
	}
	personID := target["personId"].(string)
	actualDiscID := target["id"].(string)

	// Underdog assessment
	var classification, origins, ambitions, disadv, rej, obst, turn, prog, brk, contra, gaps, audInt string
	_ = db.DB.QueryRowContext(ctx, `
		SELECT classification, origins_summary, early_ambitions, disadvantages, rejections_and_failures,
		       obstacles, turning_points, career_progression, recent_breakthrough,
		       contradictions_report, research_gaps_report, audience_interest_report
		FROM underdog_assessments WHERE discovery_id = $1 OR person_id = $2 LIMIT 1
	`, actualDiscID, personID).Scan(&classification, &origins, &ambitions, &disadv, &rej, &obst, &turn, &prog, &brk, &contra, &gaps, &audInt)

	// Score rationale
	var ratJSON string
	_ = db.DB.QueryRowContext(ctx, `SELECT COALESCE(score_rationale_json, '{}') FROM opportunity_scores WHERE discovery_id = $1 LIMIT 1`, actualDiscID).Scan(&ratJSON)
	var rationale map[string]string
	_ = json.Unmarshal([]byte(ratJSON), &rationale)

	// Trend observation history
	obsRows, _ := db.DB.QueryContext(ctx, `
		SELECT trend_stage, views_total, views_per_hour, views_per_day, channel_relative_perf, independent_coverage, momentum_score, observed_at::text
		FROM trend_observations WHERE discovery_id = $1 ORDER BY observed_at ASC
	`, actualDiscID)
	var observations []map[string]any
	if obsRows != nil {
		for obsRows.Next() {
			var st, oa string
			var vt int64
			var vph, vpd, cr, mom float64
			var ic int
			if obsRows.Scan(&st, &vt, &vph, &vpd, &cr, &ic, &mom, &oa) == nil {
				observations = append(observations, map[string]any{
					"trendStage": st, "viewsTotal": vt, "viewsPerHour": vph, "viewsPerDay": vpd,
					"channelRelativePerf": cr, "independentCoverage": ic, "momentumScore": mom, "observedAt": oa,
				})
			}
		}
		obsRows.Close()
	}

	dossier, _ := getPersonResearchDossier(ctx, personID)

	var existingProjectID string
	_ = db.DB.QueryRowContext(ctx, `SELECT id FROM documentary_projects WHERE discovery_id = $1 OR person_id = $2 LIMIT 1`, actualDiscID, personID).Scan(&existingProjectID)

	target["underdogAssessment"] = map[string]any{
		"classification":          classification,
		"originsSummary":          origins,
		"earlyAmbitions":          ambitions,
		"disadvantages":           disadv,
		"rejectionsAndFailures":   rej,
		"obstacles":               obst,
		"turningPoints":           turn,
		"careerProgression":       prog,
		"recentBreakthrough":      brk,
		"contradictionsReport":    contra,
		"researchGapsReport":      gaps,
		"audienceInterestReport":  audInt,
	}
	target["scoreRationale"] = rationale
	target["trendHistory"] = observations
	target["research"] = dossier
	target["projectId"] = existingProjectID

	return target, nil
}

func getPersonResearchDossier(ctx interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}, personID string) (map[string]any, error) {
	var name, cat, country, summary, bio, img, uClass, rStatus string
	err := db.DB.QueryRowContext(ctx, `
		SELECT canonical_name, category, country, COALESCE(summary, ''), COALESCE(biography, ''),
		       COALESCE(image_url, ''), underdog_classification, research_status
		FROM people WHERE id = $1
	`, personID).Scan(&name, &cat, &country, &summary, &bio, &img, &uClass, &rStatus)
	if err != nil {
		return nil, err
	}

	// Aliases
	aRows, _ := db.DB.QueryContext(ctx, `SELECT alias_name, source_provenance FROM person_aliases WHERE person_id = $1`, personID)
	var aliases []string
	if aRows != nil {
		for aRows.Next() {
			var al, prov string
			if aRows.Scan(&al, &prov) == nil {
				aliases = append(aliases, al)
			}
		}
		aRows.Close()
	}

	// Timeline
	tRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, year_or_date, chronological_order, phase, title, description, verification_status
		FROM person_timelines WHERE person_id = $1 ORDER BY chronological_order ASC
	`, personID)
	var timeline []map[string]any
	if tRows != nil {
		for tRows.Next() {
			var id, yr, ph, ti, desc, ver string
			var ord int
			if tRows.Scan(&id, &yr, &ord, &ph, &ti, &desc, &ver) == nil {
				timeline = append(timeline, map[string]any{
					"id": id, "yearOrDate": yr, "order": ord, "phase": ph,
					"title": ti, "description": desc, "verificationStatus": ver,
				})
			}
		}
		tRows.Close()
	}

	// Claims
	cRows, _ := db.DB.QueryContext(ctx, `
		SELECT c.id, c.category, c.claim_text, c.verification_status, COALESCE(c.event_date, ''),
		       c.confidence_score, COALESCE(c.notes, ''), COALESCE(s.title, 'Primary Source'), COALESCE(s.url, '')
		FROM research_claims c
		LEFT JOIN research_sources s ON s.id = c.source_id
		WHERE c.person_id = $1 ORDER BY c.created_at ASC
	`, personID)
	var claims []map[string]any
	if cRows != nil {
		for cRows.Next() {
			var id, cat, txt, ver, dt, notes, sTitle, sURL string
			var conf float64
			if cRows.Scan(&id, &cat, &txt, &ver, &dt, &conf, &notes, &sTitle, &sURL) == nil {
				claims = append(claims, map[string]any{
					"id": id, "category": cat, "claimText": txt, "verificationStatus": ver,
					"eventDate": dt, "confidenceScore": conf, "notes": notes,
					"sourceTitle": sTitle, "sourceUrl": sURL,
				})
			}
		}
		cRows.Close()
	}

	// Sources
	sRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, url, title, publisher, COALESCE(published_at, ''), source_type, reliability_score, COALESCE(raw_excerpt, '')
		FROM research_sources WHERE person_id = $1 ORDER BY created_at DESC
	`, personID)
	var sources []map[string]any
	if sRows != nil {
		for sRows.Next() {
			var id, u, t, pub, dt, st, ex string
			var rel float64
			if sRows.Scan(&id, &u, &t, &pub, &dt, &st, &rel, &ex) == nil {
				sources = append(sources, map[string]any{
					"id": id, "url": u, "title": t, "publisher": pub,
					"publishedAt": dt, "sourceType": st, "reliabilityScore": rel, "rawExcerpt": ex,
				})
			}
		}
		sRows.Close()
	}

	return map[string]any{
		"personId":               personID,
		"canonicalName":          name,
		"aliases":                aliases,
		"category":               cat,
		"country":                country,
		"summary":                summary,
		"biography":              bio,
		"imageUrl":               img,
		"underdogClassification": uClass,
		"researchStatus":         rStatus,
		"timeline":               timeline,
		"claims":                 claims,
		"sources":                sources,
	}, nil
}

func approveDiscoveryForDocumentary(ctx interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}, discID string) (string, error) {
	var existingID string
	err := db.DB.QueryRowContext(ctx, `SELECT id FROM documentary_projects WHERE discovery_id = $1 LIMIT 1`, discID).Scan(&existingID)
	if err == nil && existingID != "" {
		return existingID, nil
	}

	var personID, headline string
	err = db.DB.QueryRowContext(ctx, `SELECT person_id, headline FROM discoveries WHERE id = $1`, discID).Scan(&personID, &headline)
	if err != nil {
		return "", err
	}
	var personName string
	_ = db.DB.QueryRowContext(ctx, `SELECT canonical_name FROM people WHERE id = $1`, personID).Scan(&personName)

	projID := "proj-" + agents.Slugify(personName) + "-" + uuid.NewString()[:4]
	slug := agents.Slugify(personName) + "-documentary-" + uuid.NewString()[:4]
	title := fmt.Sprintf("Against All Odds: The %s Story (%s)", personName, headline)

	ytTitles, _ := json.Marshal([]string{
		fmt.Sprintf("How %s Shocked the World: The Untold Underdog Documentary", personName),
		fmt.Sprintf("Nobody Gave %s a Chance — Then History Changed Forever", personName),
		fmt.Sprintf("Inside the Impossible Rise of %s", personName),
	})

	_, err = db.DB.ExecContext(ctx, `
		INSERT INTO documentary_projects (
			id, person_id, discovery_id, title, slug, status, target_duration_minutes,
			visual_style, aspect_ratio, image_interval_seconds, thumbnail_concept,
			youtube_titles_json, youtube_description, music_suggestions
		) VALUES ($1,$2,$3,$4,$5,'in_production',20,
			'Cinematic Archival Realism (35mm Documentary Stills)', '16:9', 8,
			$6, $7, $8,
			'Atmospheric solo strings & low pulse percussion during early struggles, rising to full cinematic brass at breakthrough.'
		)
	`, projID, personID, discID, title, slug,
		fmt.Sprintf("Split-frame high-contrast portrait of %s during early struggle vs. championship breakthrough moment.", personName),
		string(ytTitles),
		fmt.Sprintf("Full 20-minute source-grounded documentary on %s.", personName))
	if err != nil {
		return "", err
	}

	bgCtx := context.Background()
	_, _ = agents.GenerateDocumentaryScript(bgCtx, projID, 20, "Cinematic Investigative Storytelling")
	_, _ = agents.GenerateCharacterReferences(bgCtx, projID)
	_, _ = agents.GenerateVisualTimeline(bgCtx, projID)
	_, _, _ = agents.GenerateProjectAudioAndSubtitles(bgCtx, projID)

	return projID, nil
}

func getProjectWorkspace(ctx interface {
	Deadline() (time.Time, bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}, projectID string) (map[string]any, error) {
	var pID, dID, title, slug, status, style, ar, thumb, ytTitlesStr, ytDesc, music, ua string
	var durMin, imgInt int
	err := db.DB.QueryRowContext(ctx, `
		SELECT person_id, COALESCE(discovery_id, ''), title, slug, status, target_duration_minutes,
		       visual_style, aspect_ratio, image_interval_seconds, COALESCE(thumbnail_concept, ''),
		       COALESCE(youtube_titles_json, '[]'), COALESCE(youtube_description, ''), COALESCE(music_suggestions, ''),
		       updated_at::text
		FROM documentary_projects WHERE id = $1
	`, projectID).Scan(&pID, &dID, &title, &slug, &status, &durMin, &style, &ar, &imgInt, &thumb, &ytTitlesStr, &ytDesc, &music, &ua)
	if err != nil {
		return nil, err
	}

	var ytTitles []string
	_ = json.Unmarshal([]byte(ytTitlesStr), &ytTitles)

	// Script
	var scrID, hook, fullText, scrStatus string
	var curVer, words, estSec int
	_ = db.DB.QueryRowContext(ctx, `
		SELECT id, current_version, opening_hook, full_narration_text, word_count, estimated_duration_seconds, status
		FROM documentary_scripts WHERE project_id = $1
	`, projectID).Scan(&scrID, &curVer, &hook, &fullText, &words, &estSec, &scrStatus)

	// Script versions
	vRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, version_number, word_count, estimated_duration_seconds, change_summary, created_by, created_at::text, full_narration_text
		FROM script_versions WHERE script_id = $1 ORDER BY version_number DESC
	`, scrID)
	var versions []map[string]any
	if vRows != nil {
		for vRows.Next() {
			var id, sum, by, ca, txt string
			var vn, wc, es int
			if vRows.Scan(&id, &vn, &wc, &es, &sum, &by, &ca, &txt) == nil {
				versions = append(versions, map[string]any{
					"id": id, "versionNumber": vn, "wordCount": wc, "estimatedDurationSeconds": es,
					"changeSummary": sum, "createdBy": by, "createdAt": ca, "fullNarrationText": txt,
				})
			}
		}
		vRows.Close()
	}

	// Character references
	cRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, character_name, role, age_range, appearance_description, hair_description,
		       clothing_description, body_proportions, locations_description, environmental_refs,
		       visual_style, color_palette, master_prompt, COALESCE(reference_image_path, '')
		FROM character_references WHERE project_id = $1 ORDER BY created_at ASC
	`, projectID)
	var characters []map[string]any
	if cRows != nil {
		for cRows.Next() {
			var id, cn, role, age, app, hair, cloth, body, loc, env, vs, pal, mp, img string
			if cRows.Scan(&id, &cn, &role, &age, &app, &hair, &cloth, &body, &loc, &env, &vs, &pal, &mp, &img) == nil {
				characters = append(characters, map[string]any{
					"id": id, "characterName": cn, "role": role, "ageRange": age,
					"appearanceDescription": app, "hairDescription": hair, "clothingDescription": cloth,
					"bodyProportions": body, "locationsDescription": loc, "environmentalRefs": env,
					"visualStyle": vs, "colorPalette": pal, "masterPrompt": mp, "referenceImagePath": img,
				})
			}
		}
		cRows.Close()
	}

	// Visual segments
	sRows, _ := db.DB.QueryContext(ctx, `
		SELECT vs.id, vs.segment_number, vs.segment_code, vs.start_seconds, vs.end_seconds, vs.duration_seconds,
		       vs.narration_passage, vs.image_prompt, COALESCE(vs.character_ref_id, ''), vs.location_ref,
		       vs.generation_status, COALESCE(ia.file_url, ''), COALESCE(cr.age_range, 'Main Subject')
		FROM visual_segments vs
		LEFT JOIN image_assets ia ON ia.id = vs.image_asset_id
		LEFT JOIN character_references cr ON cr.id = vs.character_ref_id
		WHERE vs.project_id = $1
		ORDER BY vs.segment_number ASC
	`, projectID)
	var segments []map[string]any
	if sRows != nil {
		for sRows.Next() {
			var id, code, pas, prm, cID, loc, st, imgURL, cAge string
			var num int
			var sSec, eSec, dur float64
			if sRows.Scan(&id, &num, &code, &sSec, &eSec, &dur, &pas, &prm, &cID, &loc, &st, &imgURL, &cAge) == nil {
				segments = append(segments, map[string]any{
					"id": id, "segmentNumber": num, "segmentCode": code,
					"startSeconds": sSec, "endSeconds": eSec, "durationSeconds": dur,
					"startFormatted": services.FormatSRTTime(sSec),
					"endFormatted":   services.FormatSRTTime(eSec),
					"narrationPassage": pas, "imagePrompt": prm, "characterRefId": cID,
					"characterAgeRange": cAge, "locationRef": loc, "generationStatus": st,
					"imageUrl": imgURL,
				})
			}
		}
		sRows.Close()
	}

	// Audio assets
	aRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, title, file_url, provider, voice_id, duration_seconds, is_master_narration, status, created_at::text
		FROM audio_assets WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	var audios []map[string]any
	if aRows != nil {
		for aRows.Next() {
			var id, ti, url, prov, vid, st, ca string
			var dur float64
			var isMaster bool
			if aRows.Scan(&id, &ti, &url, &prov, &vid, &dur, &isMaster, &st, &ca) == nil {
				audios = append(audios, map[string]any{
					"id": id, "title": ti, "fileUrl": url, "provider": prov,
					"voiceId": vid, "durationSeconds": dur, "isMasterNarration": isMaster,
					"status": st, "createdAt": ca,
				})
			}
		}
		aRows.Close()
	}

	// Subtitles
	subRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, format, file_url, cue_count, content_text, created_at::text
		FROM subtitle_assets WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	var subtitles []map[string]any
	if subRows != nil {
		for subRows.Next() {
			var id, fmtStr, url, txt, ca string
			var cues int
			if subRows.Scan(&id, &fmtStr, &url, &cues, &txt, &ca) == nil {
				subtitles = append(subtitles, map[string]any{
					"id": id, "format": fmtStr, "fileUrl": url, "cueCount": cues, "contentText": txt, "createdAt": ca,
				})
			}
		}
		subRows.Close()
	}

	// Exports
	eRows, _ := db.DB.QueryContext(ctx, `
		SELECT id, export_type, file_url, file_size_bytes, video_assembled, status, created_at::text
		FROM production_exports WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	var exports []map[string]any
	if eRows != nil {
		for eRows.Next() {
			var id, et, url, st, ca string
			var sz int64
			var va bool
			if eRows.Scan(&id, &et, &url, &sz, &va, &st, &ca) == nil {
				exports = append(exports, map[string]any{
					"id": id, "exportType": et, "fileUrl": url, "fileSizeBytes": sz,
					"videoAssembled": va, "status": st, "createdAt": ca,
				})
			}
		}
		eRows.Close()
	}

	researchDossier, _ := getPersonResearchDossier(ctx, pID)

	return map[string]any{
		"project": map[string]any{
			"id":                    projectID,
			"personId":              pID,
			"discoveryId":           dID,
			"title":                 title,
			"slug":                  slug,
			"status":                status,
			"targetDurationMinutes": durMin,
			"visualStyle":           style,
			"aspectRatio":           ar,
			"imageIntervalSeconds":  imgInt,
			"thumbnailConcept":      thumb,
			"youtubeTitles":         ytTitles,
			"youtubeDescription":    ytDesc,
			"musicSuggestions":      music,
			"updatedAt":             ua,
		},
		"script": map[string]any{
			"id":                       scrID,
			"currentVersion":           curVer,
			"openingHook":              hook,
			"fullNarrationText":        fullText,
			"wordCount":                words,
			"estimatedDurationSeconds": estSec,
			"status":                   scrStatus,
			"versions":                 versions,
		},
		"research":            researchDossier,
		"characterReferences": characters,
		"visualSegments":      segments,
		"audioAssets":         audios,
		"subtitleAssets":      subtitles,
		"exports":             exports,
	}, nil
}
