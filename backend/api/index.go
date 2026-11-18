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
		_, _ = w.Write([]byte(`{"status":"online","platform":"vercel-serverless","service":"trendscout-backend"}`))
	})

	appMux = corsMiddleware(mux)
}

// Handler is the entrypoint for Vercel Serverless Functions
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(initServerless)
	if appMux != nil {
		appMux.ServeHTTP(w, r)
	} else {
		http.Error(w, `{"error":"Backend initialization failed"}`, http.StatusInternalServerError)
	}
}
