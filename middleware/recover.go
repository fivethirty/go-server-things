package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func (m *Middleware) Recover(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := wrapResponseWriter(w)
		defer func() {
			if err := recover(); err != nil {
				if !wrapped.isHeaderWritten {
					m.writeError(wrapped, http.StatusInternalServerError)
				}
				m.logger.Log(
					r.Context(),
					slog.LevelError,
					"Panic",
					"status", wrapped.status,
					"error", err,
					"trace", string(debug.Stack()),
				)
			}
		}()
		h.ServeHTTP(wrapped, r)
	})
}
