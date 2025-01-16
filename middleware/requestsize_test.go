package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestRequestSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		maxBytes       int64
		requestSize    int
		contentLength  int
		expectedCode   int
		isBodyReadable bool
	}{
		{
			name:           "content length > max bytes, request size < max bytes should return 413",
			maxBytes:       10,
			requestSize:    5,
			contentLength:  15,
			expectedCode:   http.StatusRequestEntityTooLarge,
			isBodyReadable: true,
		},
		{
			name:           "content length and request size > bytes should return 413",
			maxBytes:       10,
			requestSize:    15,
			contentLength:  15,
			expectedCode:   http.StatusRequestEntityTooLarge,
			isBodyReadable: false,
		},
		{
			name:           "content length and request size < max bytes should return 200",
			maxBytes:       10,
			requestSize:    5,
			contentLength:  5,
			expectedCode:   http.StatusOK,
			isBodyReadable: true,
		},
		{
			name:           "content length < max bytes, request size > max bytes should not be readable",
			maxBytes:       10,
			requestSize:    15,
			contentLength:  5,
			expectedCode:   http.StatusOK,
			isBodyReadable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tm := newTestMiddleware()
			wrapped := tm.RequestSize(tt.maxBytes)(handler(t, http.StatusOK, 0))
			body := strings.NewReader(strings.Repeat("a", int(tt.requestSize)))
			req := httptest.NewRequest(http.MethodPost, "/", body)
			req.Header.Add("content-length", strconv.Itoa(tt.contentLength))
			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status code %d, got %d", tt.expectedCode, w.Code)
			}
			_, err := io.ReadAll(req.Body)
			_, ok := err.(*http.MaxBytesError)
			if !ok && err != nil {
				t.Fatal(err)
			}

			if ok && tt.isBodyReadable {
				t.Errorf("expected request body to be readable but got error: %v", err)
			} else if err == nil && !tt.isBodyReadable {
				t.Errorf("expected request body to be unreadable")
			}

			if len(tm.logs(t)) > 0 {
				t.Errorf("expected no logs, got %s", tm.logBuffer.String())
			}

			tm.validateErrorBodyWriterCalled(t, w.Code)
		})
	}
}
