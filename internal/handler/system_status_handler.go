package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// SystemStatusHandler exposes the two checks an orchestrator (Docker,
// Kubernetes, an internal load balancer) actually needs: is the process
// alive, and is it safe to route traffic to. Kept as its own concern
// rather than folded into a generic "health" catch-all, since liveness
// and readiness answer genuinely different operational questions.
type SystemStatusHandler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewSystemStatusHandler(pool *pgxpool.Pool, redis *redis.Client) *SystemStatusHandler {
	return &SystemStatusHandler{pool: pool, redis: redis}
}

// Live reports only that the process is up, without checking any
// dependency. An orchestrator should restart the container if and only if
// this fails — restarting a healthy process because a downstream
// dependency hiccuped just adds churn.
func (h *SystemStatusHandler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

// Ready checks that every dependency this instance needs (Postgres,
// Redis) is actually reachable. An orchestrator uses this to decide
// whether to route traffic here — a failing dependency should pull the
// instance out of the load balancer, not kill it.
func (h *SystemStatusHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := map[string]string{"postgres": "connected", "redis": "connected"}
	allReady := true

	if err := h.pool.Ping(ctx); err != nil {
		status["postgres"] = "unreachable"
		allReady = false
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = "unreachable"
		allReady = false
	}

	if !allReady {
		writeJSON(w, http.StatusServiceUnavailable, status)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
