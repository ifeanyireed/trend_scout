package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"trendscout-backend/internal/db"
)

type JobHandler func(ctx context.Context, payload map[string]any) (map[string]any, error)

type WorkerManager struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
	workerID string
	stopCh   chan struct{}
}

var GlobalQueue = &WorkerManager{
	handlers: make(map[string]JobHandler),
	workerID: "worker-" + uuid.NewString()[:8],
	stopCh:   make(chan struct{}),
}

func (wm *WorkerManager) Register(jobType string, handler JobHandler) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.handlers[jobType] = handler
}

func EnqueueJob(ctx context.Context, jobType string, priority int, idempotencyKey string, payload map[string]any) (string, error) {
	if db.DB == nil {
		return "", fmt.Errorf("db not initialized")
	}
	payloadBytes, _ := json.Marshal(payload)
	jobID := "job-" + uuid.NewString()

	if idempotencyKey != "" {
		var existingID string
		err := db.DB.QueryRowContext(ctx, `SELECT id FROM background_jobs WHERE idempotency_key = $1`, idempotencyKey).Scan(&existingID)
		if err == nil && existingID != "" {
			return existingID, nil
		}
	}

	var idemPtr *string
	if idempotencyKey != "" {
		idemPtr = &idempotencyKey
	}

	_, err := db.DB.ExecContext(ctx, `
		INSERT INTO background_jobs (id, job_type, queue_name, priority, payload_json, idempotency_key, status, attempts, max_attempts, next_run_at)
		VALUES ($1, $2, 'default', $3, $4, $5, 'queued', 0, 3, NOW())
		ON CONFLICT (idempotency_key) DO NOTHING
	`, jobID, jobType, priority, string(payloadBytes), idemPtr)
	if err != nil {
		return "", err
	}
	return jobID, nil
}

func (wm *WorkerManager) RecoverStalledJobs(ctx context.Context) {
	if db.DB == nil {
		return
	}
	_, _ = db.DB.ExecContext(ctx, `
		UPDATE background_jobs
		SET status = 'queued', locked_by = NULL, locked_at = NULL, updated_at = NOW()
		WHERE status = 'running' AND heartbeat_at < NOW() - INTERVAL '3 minutes'
	`)
}

func (wm *WorkerManager) ProcessNextJob(ctx context.Context) bool {
	if db.DB == nil {
		return false
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	var jobID, jobType, payloadStr string
	var attempts, maxAttempts int

	err = tx.QueryRowContext(ctx, `
		SELECT id, job_type, payload_json, attempts, max_attempts
		FROM background_jobs
		WHERE status = 'queued' AND next_run_at <= NOW()
		ORDER BY priority DESC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&jobID, &jobType, &payloadStr, &attempts, &maxAttempts)

	if err == sql.ErrNoRows || err != nil {
		return false
	}

	attempts++
	_, err = tx.ExecContext(ctx, `
		UPDATE background_jobs
		SET status = 'running', attempts = $2, locked_by = $3, locked_at = NOW(), heartbeat_at = NOW(), started_at = COALESCE(started_at, NOW()), updated_at = NOW()
		WHERE id = $1
	`, jobID, attempts, wm.workerID)
	if err != nil {
		return false
	}

	if err := tx.Commit(); err != nil {
		return false
	}

	wm.mu.RLock()
	handler, exists := wm.handlers[jobType]
	wm.mu.RUnlock()

	if !exists {
		_, _ = db.DB.ExecContext(ctx, `
			UPDATE background_jobs SET status = 'dead_letter', error_message = $2, completed_at = NOW(), updated_at = NOW() WHERE id = $1
		`, jobID, "no registered handler for job_type: "+jobType)
		return true
	}

	var payload map[string]any
	_ = json.Unmarshal([]byte(payloadStr), &payload)

	jobCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Heartbeat goroutine
	hbDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-hbDone:
				return
			case <-ticker.C:
				_, _ = db.DB.ExecContext(context.Background(), `UPDATE background_jobs SET heartbeat_at = NOW() WHERE id = $1`, jobID)
			}
		}
	}()

	res, execErr := handler(jobCtx, payload)
	close(hbDone)

	if execErr != nil {
		nextStatus := "queued"
		if attempts >= maxAttempts {
			nextStatus = "dead_letter"
		}
		backoffSec := int(math.Pow(2, float64(attempts)) * 5)
		_, _ = db.DB.ExecContext(context.Background(), `
			UPDATE background_jobs
			SET status = $2, error_message = $3, next_run_at = NOW() + ($4 || ' seconds')::interval, updated_at = NOW()
			WHERE id = $1
		`, jobID, nextStatus, execErr.Error(), fmt.Sprintf("%d", backoffSec))
		return true
	}

	resBytes, _ := json.Marshal(res)
	_, _ = db.DB.ExecContext(context.Background(), `
		UPDATE background_jobs
		SET status = 'completed', result_json = $2, error_message = NULL, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, jobID, string(resBytes))
	return true
}

func (wm *WorkerManager) StartBackgroundLoop(ctx context.Context) {
	wm.RecoverStalledJobs(ctx)
	go func() {
		workerTicker := time.NewTicker(3 * time.Second)
		scheduleTicker := time.NewTicker(5 * time.Minute)
		discoveryTicker := time.NewTicker(15 * time.Minute)
		defer workerTicker.Stop()
		defer scheduleTicker.Stop()
		defer discoveryTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-wm.stopCh:
				return
			case <-workerTicker.C:
				for i := 0; i < 3; i++ {
					if !wm.ProcessNextJob(ctx) {
						break
					}
				}
			case <-scheduleTicker.C:
				wm.RecoverStalledJobs(ctx)
				windowKey := fmt.Sprintf("scheduled-monitoring-%s", time.Now().UTC().Format("2006-01-02T15:04"))
				_, err := EnqueueJob(ctx, "trend_monitoring_cycle", 6, windowKey, map[string]any{"trigger": "scheduler"})
				if err != nil {
					log.Printf("scheduler enqueue error: %v", err)
				}
			case <-discoveryTicker.C:
				wm.RecoverStalledJobs(ctx)
				discKey := fmt.Sprintf("scheduled-discovery-%s", time.Now().UTC().Format("2006-01-02T15:04"))
				_, err := EnqueueJob(ctx, "discovery_scan", 8, discKey, map[string]any{"trigger": "scheduler"})
				if err != nil {
					log.Printf("discovery scheduler enqueue error: %v", err)
				}
			}
		}
	}()
}
