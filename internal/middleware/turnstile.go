package middleware

import (
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
				http.Error(
					w,
					"请求需要 Turnstile 验证",
					http.StatusForbidden,
				)
				return
			}

			if err := ts.Verify(
				r.Context(),
				token,
				r.RemoteAddr,
			); err != nil {
				http.Error(
					w,
					"Turnstile 验证不通过",
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
