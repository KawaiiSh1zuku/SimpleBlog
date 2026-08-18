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
	categoryHandler *handler.CategoryHandler,
	postHandler *handler.PostHandler,
	commentHandler *handler.CommentHandler,
	uploadHandler *handler.UploadHandler,
	linkHandler *handler.LinkHandler,
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

		admin.Get("/categories", categoryHandler.List)
		admin.Post("/categories", categoryHandler.Create)
		admin.Get("/categories/{categoryID}", categoryHandler.Get)
		admin.Patch("/categories/{categoryID}", categoryHandler.Update)
		admin.Delete("/categories/{categoryID}", categoryHandler.Delete)

		admin.Get("/links", linkHandler.ListManaged)
		admin.Post("/links", linkHandler.Create)
		admin.Get("/links/{linkID}", linkHandler.Get)
		admin.Patch("/links/{linkID}", linkHandler.Update)
		admin.Delete("/links/{linkID}", linkHandler.Delete)
	})

	r.Get("/posts", postHandler.List)
	r.Post("/posts", postHandler.Create)
	r.Get("/posts/{postID}", postHandler.Get)
	r.Patch("/posts/{postID}", postHandler.Update)
	r.Delete("/posts/{postID}", postHandler.Delete)
	r.Post("/posts/{postID}/publish", postHandler.Publish)
	r.Post("/posts/{postID}/withdraw", postHandler.Withdraw)

	r.Get("/comments", commentHandler.ListManaged)
	r.Post("/comments/{commentID}/hide", commentHandler.Hide)
	r.Post("/comments/{commentID}/show", commentHandler.Show)
	r.Delete("/comments/{commentID}", commentHandler.Delete)
	r.Post("/uploads/presign", uploadHandler.Presign)
	return r
}
