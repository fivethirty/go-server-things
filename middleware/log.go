package middleware

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

type contextHandler struct {
	slog.Handler
	ctxKey contextKey
}

func (ch *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ch.ctxKey).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}
	return ch.Handler.Handle(ctx, r)
}

type Logger struct {
	*slog.Logger
	ctxKey contextKey
}

func NewLogger(w io.Writer) *Logger {
	handler := &contextHandler{
		Handler: slog.NewJSONHandler(w, nil),
		ctxKey:  contextKey(uuid.NewString()),
	}
	return &Logger{
		Logger: slog.New(handler),
		ctxKey: handler.ctxKey,
	}
}

func (l *Logger) AppendCtx(ctx context.Context, attr slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if v, ok := ctx.Value(l.ctxKey).([]slog.Attr); ok {
		v = append(v, attr)
		return context.WithValue(ctx, l.ctxKey, v)
	}

	v := []slog.Attr{}
	v = append(v, attr)
	return context.WithValue(ctx, l.ctxKey, v)
}

func (m *Middleware) Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped, ok := w.(*responseWriter)
		if !ok {
			wrapped = wrapResponseWriter(w)
		}

		ctx := m.logger.AppendCtx(r.Context(), slog.String("method", r.Method))
		ctx = m.logger.AppendCtx(ctx, slog.String("path", r.URL.EscapedPath()))
		ctx = m.logger.AppendCtx(ctx, slog.Any("params", r.URL.Query()))
		ctx = m.logger.AppendCtx(ctx, slog.String("request_id", uuid.NewString()))

		userID := ""
		ctx = context.WithValue(ctx, m.userIDContextKey, &userID)
		ctx = m.logger.AppendCtx(ctx, slog.Any("user_id", &userID))

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
