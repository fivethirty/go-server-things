package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		status int
		size   int
		level  slog.Level
		userID string
	}{
		{
			name:   "OK should log INFO",
			method: http.MethodPost,
			path:   "/foo?bar=baz",
			status: http.StatusOK,
			size:   10,
			level:  slog.LevelInfo,
		},
		{
			name:   "redirect should log INFO",
			method: http.MethodPost,
			path:   "/foo?bar=baz",
			status: http.StatusFound,
			size:   0,
			level:  slog.LevelInfo,
		},
		{
			name:   "error should log ERROR",
			method: http.MethodPost,
			path:   "/foo?bar=baz",
			status: http.StatusNotFound,
			size:   5,
			level:  slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := newTestMiddleware()
			wrapped := tm.Log(handler(t, tt.status, tt.size))
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			if tt.userID != "" {
				req.Context()
			}
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			entries := tm.logs(t)
			if len(entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(entries))
			}
			entry := entries[0]
			if entry.Level != tt.level.String() {
				t.Errorf("expected log level %s, got %s", tt.level.String(), entry.Level)
			}
			if entry.Method != tt.method {
				t.Errorf("expected method %s, got %s", tt.method, entry.Method)
			}
			u, err := url.Parse(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if entry.Path != u.Path {
				t.Errorf("expected path %s, got %s", tt.path, entry.Path)
			}
			if len(entry.Params) != len(u.Query()) {
				t.Errorf("expected %d params, got %d", len(u.Query()), len(entry.Params))
			}
			for k, v := range entry.Params {
				if v[0] != u.Query().Get(k) {
					t.Errorf("expected param %s=%s, got %s", k, u.Query().Get(k), v[0])
				}
			}
			if entry.RequestID == "" {
				t.Error("expected request ID to be set")
			}
			if entry.Duration == 0 {
				t.Error("expected duration to > 0")
			}
			if entry.ContentLength != tt.size {
				t.Errorf("expected content length %d, got %d", tt.size, entry.ContentLength)
			}
		})
	}
}

func TestLogWithUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		userID string
	}{
		{
			name:   "should set user ID ",
			userID: "user_1",
		},
		{
			name:   "should handle empty user ID",
			userID: "user_1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := newTestMiddleware()
			ctxUserID := ""
			wrapped := tm.Log(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if tt.userID != "" {
					tm.SetContextUserID(r.Context(), tt.userID)
					ctxUserID, _ = tm.GetContextUserID(r.Context())
				}
			}))
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if ctxUserID != tt.userID {
				t.Errorf("expected user ID in context %s, got %s", tt.userID, ctxUserID)
			}
			entries := tm.logs(t)
			if len(entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(entries))
			}
			entry := entries[0]
			if entry.UserID != tt.userID {
				t.Errorf("expected user ID %s in logs, got %s", tt.userID, entry.UserID)
			}
		})
	}
}
