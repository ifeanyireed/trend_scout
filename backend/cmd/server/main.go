package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trendscout-backend/internal/agents"
	"trendscout-backend/internal/api"
	"trendscout-backend/internal/db"
	"trendscout-backend/internal/queue"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("[TrendScout Studio] Connecting to PostgreSQL & running migrations...")
	conn, err := db.ConnectAndMigrate()
	if err != nil {
		log.Fatalf("Failed to connect/migrate database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	agents.RegisterQueueHandlers()

	log.Println("[TrendScout Studio] Verifying baseline operational data & underdog dossiers...")
	if err := agents.EnsureInitialOperationalData(ctx); err != nil {
		log.Printf("Seed warning: %v", err)
	}

	queue.GlobalQueue.StartBackgroundLoop(ctx)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	// Serve /storage/* media files (images, audio, subtitles, exports, characters)
	fsStorage := http.StripPrefix("/storage/", http.FileServer(http.Dir(db.StorageRoot)))
	mux.Handle("/storage/", fsStorage)

	// Serve built React Web Dashboard if present
	webDist := filepath.Join(filepath.Dir(db.StorageRoot), "web", "dist")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/storage/") {
			http.NotFound(w, r)
			return
		}
		targetFile := filepath.Join(webDist, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(targetFile); err == nil && !info.IsDir() {
			http.ServeFile(w, r, targetFile)
			return
		}
		indexFile := filepath.Join(webDist, "index.html")
		if _, err := os.Stat(indexFile); err == nil {
			http.ServeFile(w, r, indexFile)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"TrendScout Studio Go + PostgreSQL Cloud Backend","status":"online"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	log.Printf("[TrendScout Studio] Go + PostgreSQL Server listening on port :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server exited: %v", err)
	}
}
