package logs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/fivethirty/go-server-things/logs"
)

func TestLogsWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		attrs []slog.Attr
	}{
		{
			name: "no attributes",
		},
		{
			name: "attributes",
			attrs: []slog.Attr{
				slog.String("key", "value"),
				slog.Int("int", 101),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(nil)
			ctx := context.Background()
			for _, attr := range tt.attrs {
				ctx = logs.AppendCtx(ctx, attr)
			}

			logger := logs.New(buffer)
			logger.Log(ctx, slog.LevelInfo, "test")
			var entry map[string]any
			dec := json.NewDecoder(buffer)
			if err := dec.Decode(&entry); err != nil {
				t.Fatal(err)
			}
			for _, attr := range tt.attrs {
				value, ok := entry[attr.Key]
				if !ok {
					t.Errorf("value for key %s not found in logs", attr.Key)
				}
				strValue := fmt.Sprintf("%v", value)
				if strValue != attr.Value.String() {
					t.Errorf("expected %v but got %v", attr.Value.Any(), strValue)
				}
			}
		})
	}
}
