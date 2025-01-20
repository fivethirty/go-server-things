package logs

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const slogFields contextKey = "slog_fields"

var Default *slog.Logger = New(os.Stdout)

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

func New(w io.Writer) *slog.Logger {
	handler := &contextHandler{
		Handler: slog.NewJSONHandler(w, nil),
	}
	return slog.New(handler)
}

func AppendCtx(ctx context.Context, attr slog.Attr) context.Context {
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
