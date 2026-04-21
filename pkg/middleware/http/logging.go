package http

import (
	"net/http"
	"time"

	"github.com/akgate/platform/pkg/logging"
)

func LoggingMiddleware(log logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := newStatusResponseWriter(w)

			reqLogger := log.With(
				logging.String("request_id", RequestIDFromContext(r.Context())),
			)

			next.ServeHTTP(sw, r)

			reqLogger.Info("http request",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.String("query", r.URL.RawQuery),
				logging.Int("status", sw.status),
				logging.Int("bytes", sw.size),
				logging.Duration("duration", time.Since(start)),
				logging.String("remote_addr", r.RemoteAddr),
				logging.String("user_agent", r.UserAgent()),
			)
		})
	}
}
