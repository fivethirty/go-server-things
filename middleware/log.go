package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fivethirty/go-server-things/logs"
	"github.com/google/uuid"
)

func (m *Middleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped, ok := w.(*responseWriter)
		if !ok {
			wrapped = wrapResponseWriter(w)
		}

		ctx := logs.AppendCtx(r.Context(), slog.String("method", r.Method))
		ctx = logs.AppendCtx(ctx, slog.String("path", r.URL.EscapedPath()))
		ctx = logs.AppendCtx(ctx, slog.Any("params", r.URL.Query()))
		ctx = logs.AppendCtx(ctx, slog.String("request_id", uuid.NewString()))

		userID := ""
		ctx = context.WithValue(ctx, m.userIDContextKey, &userID)
		ctx = logs.AppendCtx(ctx, slog.Any("user_id", &userID))

		r = r.WithContext(ctx)

		next.ServeHTTP(wrapped, r)

		level := slog.LevelInfo
		if wrapped.status >= 400 {
			level = slog.LevelError
		}
		m.logger.Log(
			ctx,
			level,
			"Request",
			"status", wrapped.status,
			"duration", time.Since(start),
			"content_length", wrapped.bytesWritten,
		)
	})
}

func (m *Middleware) SetContextUserID(ctx context.Context, userID string) error {
	ptr, err := m.userIDPtr(ctx)
	if err != nil {
		return fmt.Errorf("SetContextUserID: %w", err)
	}
	*ptr = userID
	return nil
}

func (m *Middleware) GetContextUserID(ctx context.Context) (string, error) {
	ptr, err := m.userIDPtr(ctx)
	if err != nil || *ptr == "" {
		return "", fmt.Errorf("GetContextUserID: no user id in context %w", err)
	}
	return *ptr, nil
}

func (m *Middleware) userIDPtr(ctx context.Context) (*string, error) {
	ptr, ok := ctx.Value(m.userIDContextKey).(*string)
	if !ok || ptr == nil {
		return nil, errors.New("unexpected nil pointer")
	}
	return ptr, nil
}
