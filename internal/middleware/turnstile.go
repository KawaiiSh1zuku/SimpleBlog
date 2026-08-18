package middleware

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"net/http"
)

func TurnstileMiddleware(
	ts *service.TurnstileService,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Turnstile-Token")

			if token == "" {
				response.Error(w, 401, "请求需要 Turnstile 验证")
				return
			}

			if err := ts.Verify(
				r.Context(),
				token,
				r.RemoteAddr,
			); err != nil {
				response.Error(w, 403, "Turnstile 验证不通过")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
