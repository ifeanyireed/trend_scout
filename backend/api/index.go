package handler

import (
	"context"
	"net/http"
	"sync"

	"trendscout-backend/internal/agents"
	"trendscout-backend/internal/api"
	"trendscout-backend/internal/db"
)

var (
	appMux   http.Handler
	initOnce sync.Once
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func initServerless() {
	_, _ = db.ConnectAndMigrate()
	ctx := context.Background()
	agents.RegisterQueueHandlers()
	_ = agents.EnsureInitialOperationalData(ctx)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Built-in Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"online","service":"trendscout-studio"}`))
	})

	appMux = corsMiddleware(mux)
}

// Handler is the entrypoint for Vercel Serverless Functions
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(initServerless)
	if appMux != nil {
		// Restore original URL path if Vercel internal rewrite changed it to /api/index.go
		if r.URL.Path == "/api/index.go" || r.URL.Path == "/api/index" {
			if orig := r.Header.Get("x-forwarded-uri"); orig != "" {
				r.URL.Path = orig
			} else if matched := r.Header.Get("x-matched-path"); matched != "" {
				r.URL.Path = matched
			}
		}
		appMux.ServeHTTP(w, r)
	} else {
		http.Error(w, `{"error":"Backend initialization failed"}`, http.StatusInternalServerError)
	}
}
