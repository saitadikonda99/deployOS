package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDFromContext returns the request ID stored by Middleware, if any.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// Middleware assigns a request ID to every incoming request, makes it
// available via RequestIDFromContext, and logs the request's method,
// path, status code, and duration once it completes.
func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := newRequestID()

			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			ww := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r.WithContext(ctx))

			logger.Info("http request",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// statusRecorder captures the status code written by the wrapped
// http.ResponseWriter so it can be logged after the handler returns.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
