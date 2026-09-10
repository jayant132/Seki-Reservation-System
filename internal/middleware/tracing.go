package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/jayant132/seki/internal/tracing"
)

// Tracing starts a server span for every request, tags it with the route
// and status code, marks it as an error span on 5xx, and surfaces the
// trace ID in the response so a client (or the Streamlit demo UI) can
// link straight to the matching trace in Jaeger.
func Tracing(serviceName string) func(http.Handler) http.Handler {
	tracer := tracing.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			traceID := tracing.TraceIDFromContext(ctx)
			if traceID != "" {
				w.Header().Set("X-Trace-Id", traceID)
			}

			ww := &statusCapturingWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r.WithContext(ctx))

			route := chi.RouteContext(ctx).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", route),
				attribute.Int("http.status_code", ww.status),
			)
			if ww.status >= 500 {
				span.SetStatus(codes.Error, http.StatusText(ww.status))
			}
		})
	}
}

type statusCapturingWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
