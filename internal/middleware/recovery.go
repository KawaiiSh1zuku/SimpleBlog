package middleware

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer 将未处理的 panic 转换为统一的业务响应，避免泄露内部细节。
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.ErrorContext(
					r.Context(),
					"panic recovered",
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				response.Error(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
