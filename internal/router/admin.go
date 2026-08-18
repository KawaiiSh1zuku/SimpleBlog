package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func AdminRouter(
	userHandler *handler.UserHandler,
	tokenAuth *jwtauth.JWTAuth,
	users appmiddleware.CurrentUserStore,
) chi.Router {
	r := newRouter()
	r.Use(jwtauth.Verifier(tokenAuth))
	r.Use(appmiddleware.Authenticate(users))

	r.Group(func(admin chi.Router) {
		admin.Use(appmiddleware.RequireRole(model.RoleAdmin))
		admin.Get("/users", userHandler.List)
		admin.Post("/users", userHandler.Create)
		admin.Get("/users/{userID}", userHandler.Get)
		admin.Patch("/users/{userID}", userHandler.Update)
		admin.Delete("/users/{userID}", userHandler.Delete)
		admin.Put("/users/{userID}/password", userHandler.UpdatePassword)
	})
	return r
}
