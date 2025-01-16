package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fivethirty/go-server-things/logs"
	"github.com/fivethirty/go-server-things/middleware"
)

type logEntry struct {
	Level         string        `json:"level"`
	Message       string        `json:"msg"`
	Status        int           `json:"status"`
	Method        string        `json:"method"`
	Path          string        `json:"path"`
	Params        url.Values    `json:"params"`
	RequestID     string        `json:"request_id"`
	Duration      time.Duration `json:"duration"`
	ContentLength int           `json:"content_length"`
	UserID        string        `json:"user_id"`
}

type testMiddleware struct {
	*middleware.Middleware
	ErrorBodyCalled bool
	logBuffer       bytes.Buffer
}

func newTestMiddleware() *testMiddleware {
	tm := testMiddleware{
		ErrorBodyCalled: false,
	}
	m := middleware.New(
		func(w http.ResponseWriter, code int) {
			tm.ErrorBodyCalled = true
		},
		logs.New(&tm.logBuffer),
	)
	tm.Middleware = m
	return &tm
}

func (tm *testMiddleware) validateErrorBodyWriterCalled(t *testing.T, code int) {
	t.Helper()
	if code < 400 && tm.ErrorBodyCalled {
		t.Errorf("status code was %d but error body writer was called", code)
	} else if code >= 400 && !tm.ErrorBodyCalled {
		t.Errorf("status code was %d but error body writer was not called", code)
	}
}

func (tm *testMiddleware) logs(t *testing.T) []logEntry {
	t.Helper()
	var entries []logEntry
	dec := json.NewDecoder(&tm.logBuffer)
	for {
		var entry logEntry
		if err := dec.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func handler(t *testing.T, status int, responseSize int) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if responseSize > 0 {
			_, err := w.Write([]byte(strings.Repeat("a", responseSize)))
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}
