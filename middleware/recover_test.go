package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		handler      http.Handler
		expectedCode int
	}{
		{
			name:         "no panic should return 200",
			handler:      handler(t, http.StatusOK, 0),
			expectedCode: http.StatusOK,
		},
		{
			name: "panic should return 500",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("oops!")
			}),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := newTestMiddleware()
			wrapped := tm.Recover(tt.handler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if w.Code == http.StatusOK && len(tm.logs(t)) != 0 {
				t.Errorf("expected no logs, got %s", tm.logBuffer.String())
			} else if w.Code != http.StatusOK {
				entries := tm.logs(t)
				if len(entries) != 1 {
					t.Fatalf("expected 1 log entry, got %d", len(entries))
				}
				entry := entries[0]
				if entry.Level != "ERROR" {
					t.Errorf("expected log level ERROR, got %s", entry.Level)
				}
				if entry.Message != "Panic" {
					t.Errorf("expected log message Panic, got %s", entry.Message)
				}
				if entry.Status != http.StatusInternalServerError {
					t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, entry.Status)
				}
			}
			tm.validateErrorBodyWriterCalled(t, w.Code)
		})
	}
}
