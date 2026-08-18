package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const MaxRequestBodyBytes int64 = 2 * 1024 * 1024

func LimitRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &accessResponseWriter{ResponseWriter: w, businessCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		attributes := []any{
			"request_id", chimiddleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"remote_ip", requestClientIP(r),
			"business_code", recorder.businessCode,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"response_bytes", recorder.bytes,
		}
		switch {
		case recorder.businessCode >= 500:
			slog.ErrorContext(r.Context(), "request completed", attributes...)
		case recorder.businessCode >= 400:
			slog.WarnContext(r.Context(), "request completed", attributes...)
		default:
			slog.InfoContext(r.Context(), "request completed", attributes...)
		}
	})
}

type accessResponseWriter struct {
	http.ResponseWriter
	businessCode int
	bytes        int
}

func (w *accessResponseWriter) SetBusinessCode(code int) {
	w.businessCode = code
}

func (w *accessResponseWriter) Write(data []byte) (int, error) {
	written, err := w.ResponseWriter.Write(data)
	w.bytes += written
	return written, err
}

func (w *accessResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
