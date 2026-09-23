package rest

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/swiftahul20/expense-tracker/internal/auth"
)

func StructuredLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			userID, hasUser := auth.UserIDFromContext(r.Context())

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", float64(duration.Microseconds()) / 1000.0,
				"remote_addr", r.RemoteAddr,
			}

			if hasUser {
				attrs = append(attrs, "user_id", userID)
			}

			if ww.Status() >= 500 {
				log.Error("request completed", attrs...)
			} else if ww.Status() >= 400 {
				log.Warn("request completed", attrs...)
			} else {
				log.Info("request completed", attrs...)
			}
		})
	}
}
