package http

import (
	"log"
	"net/http"
	"runtime/debug"
)

func RecoverMiddleware(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Printf(
						"panic recovered: request_id=%s panic=%v\n%s",
						RequestIDFromContext(r.Context()),
						rec,
						debug.Stack(),
					)

					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":      "internal server error",
						"request_id": RequestIDFromContext(r.Context()),
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
