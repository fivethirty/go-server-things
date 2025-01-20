package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	slogFields       contextKey = "slog_fields"
)

var DefaultLogger *slog.Logger = NewLogger(os.Stdout)

func (m *Middleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped, ok := w.(*responseWriter)
		if !ok {
			wrapped = wrapResponseWriter(w)
		}

		ctx := appendCtx(r.Context(), slog.String("method", r.Method))
		ctx = appendCtx(ctx, slog.String("path", r.URL.EscapedPath()))
		ctx = appendCtx(ctx, slog.Any("params", r.URL.Query()))
		ctx = appendCtx(ctx, slog.String("request_id", uuid.NewString()))

		userID := ""
		ctx = context.WithValue(ctx, userIDContextKey, &userID)
		ctx = appendCtx(ctx, slog.Any("user_id", &userID))

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

func SetUserID(ctx context.Context, userID string) error {
	ptr, err := userIDPtr(ctx)
	if err != nil {
		return fmt.Errorf("SetContextUserID: %w", err)
	}
	*ptr = userID
	return nil
}

func UserID(ctx context.Context) (string, error) {
	ptr, err := userIDPtr(ctx)
	if err != nil || *ptr == "" {
		return "", fmt.Errorf("GetContextUserID: no user id in context %w", err)
	}
	return *ptr, nil
}

func userIDPtr(ctx context.Context) (*string, error) {
	ptr, ok := ctx.Value(userIDContextKey).(*string)
	if !ok || ptr == nil {
		return nil, errors.New("unexpected nil pointer")
	}
	return ptr, nil
}

type contextHandler struct {
	slog.Handler
}

func (ch *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}
	return ch.Handler.Handle(ctx, r)
}

func NewLogger(w io.Writer) *slog.Logger {
	handler := &contextHandler{
		Handler: slog.NewJSONHandler(w, nil),
	}
	return slog.New(handler)
}

func appendCtx(ctx context.Context, attr slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if v, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		v = append(v, attr)
		return context.WithValue(ctx, slogFields, v)
	}

	v := []slog.Attr{}
	v = append(v, attr)
	return context.WithValue(ctx, slogFields, v)
}
