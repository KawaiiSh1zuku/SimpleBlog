package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
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

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func AccessLog(accessWriter io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &accessResponseWriter{
				ResponseWriter: w,
				httpStatus:     http.StatusOK,
				businessCode:   http.StatusOK,
			}
			next.ServeHTTP(recorder, r)
			completedAt := time.Now()
			elapsed := completedAt.Sub(startedAt)
			remoteIP := accessLogIP(requestClientIP(r))

			attributes := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"remote_ip", remoteIP,
				"http_status", recorder.httpStatus,
				"business_code", recorder.businessCode,
				"duration_ms", elapsed.Milliseconds(),
				"response_bytes", recorder.bytes,
			}
			level := slog.LevelInfo
			switch {
			case recorder.businessCode >= 500:
				level = slog.LevelError
			case recorder.businessCode >= 400:
				level = slog.LevelWarn
			}
			slog.Log(r.Context(), level, "request completed", attributes...)

			if accessWriter != nil {
				requestLine := strconv.Quote(r.Method + " " + r.URL.Path + " " + r.Proto)
				_, err := fmt.Fprintf(
					accessWriter,
					"%s - - [%s] %s %d %d duration_ms=%d business_code=%d\n",
					remoteIP,
					completedAt.Format("02/Jan/2006:15:04:05 -0700"),
					requestLine,
					recorder.httpStatus,
					recorder.bytes,
					elapsed.Milliseconds(),
					recorder.businessCode,
				)
				if err != nil {
					slog.ErrorContext(r.Context(), "write access log", "error", err)
				}
			}
		})
	}
}

func accessLogIP(value string) string {
	value = strings.TrimSpace(value)
	if net.ParseIP(value) == nil {
		return "-"
	}
	return value
}

type accessResponseWriter struct {
	http.ResponseWriter
	httpStatus      int
	wroteHeader     bool
	businessCode    int
	businessCodeSet bool
	bytes           int
}

func (w *accessResponseWriter) SetBusinessCode(code int) {
	w.businessCode = code
	w.businessCodeSet = true
}

func (w *accessResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.httpStatus = code
	if !w.businessCodeSet {
		w.businessCode = code
		w.businessCodeSet = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *accessResponseWriter) Write(data []byte) (int, error) {
	written, err := w.ResponseWriter.Write(data)
	w.bytes += written
	return written, err
}

func (w *accessResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
