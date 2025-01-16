package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

type errorWriter func(w http.ResponseWriter, code int)

type userIDContextKey string

type Middleware struct {
	userIDContextKey userIDContextKey
	logger           *Logger
	writeError       errorWriter
}

func New(writeErrorBody errorWriter, logger *Logger) *Middleware {
	return &Middleware{
		userIDContextKey: userIDContextKey(uuid.NewString()),
		writeError: func(w http.ResponseWriter, code int) {
			w.WriteHeader(code)
			if writeErrorBody != nil {
				writeErrorBody(w, code)
			}
		},
		logger: logger,
	}
}

type responseWriter struct {
	http.ResponseWriter
	status          int
	bytesWritten    int64
	isHeaderWritten bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.isHeaderWritten {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.isHeaderWritten = true
}

func (rw *responseWriter) Write(body []byte) (int, error) {
	bytesWritten, err := rw.ResponseWriter.Write(body)
	rw.bytesWritten += int64(bytesWritten)
	return bytesWritten, err
}
