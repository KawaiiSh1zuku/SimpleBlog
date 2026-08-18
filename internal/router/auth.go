package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"time"
)

func AuthRouter(
	authHandler *handler.AuthHandler,
	tokenAuth *jwtauth.JWTAuth,
	users appmiddleware.CurrentUserStore,
) chi.Router {
	r := newRouter()
	loginLimiter := appmiddleware.NewIPRateLimiter(10, time.Minute)
	r.With(loginLimiter.Limit).Post("/login", authHandler.Login)
	r.Group(func(protected chi.Router) {
		protected.Use(jwtauth.Verifier(tokenAuth))
		protected.Use(appmiddleware.Authenticate(users))
		protected.Get("/me", authHandler.Me)
	})
	return r
}
