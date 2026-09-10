package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jayant132/seki/internal/metrics"
	"github.com/jayant132/seki/internal/tracing"
)

// RequestLogger emits one structured log line per request and records
// latency histograms per route+status, so p50/p95/p99 are queryable in
// Grafana without grepping logs.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			logger.Info("http_request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("request_id", middleware.GetReqID(r.Context())),
				slog.String("trace_id", tracing.TraceIDFromContext(r.Context())),
			)

			metrics.HTTPRequestDuration.WithLabelValues(
				r.Method, route, strconv.Itoa(ww.Status()),
			).Observe(duration.Seconds())
		})
	}
}
