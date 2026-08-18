package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func PublicRouter(
	publicHandler *handler.PublicHandler,
	commentHandler *handler.CommentHandler,
	turnstile appmiddleware.TurnstileVerifier,
) chi.Router {
	r := newRouter()
	r.Get("/posts", publicHandler.ListPosts)
	r.Get("/posts/{slug}/comments", commentHandler.ListPublic)
	r.With(appmiddleware.TurnstileMiddleware(turnstile)).Post("/posts/{slug}/comments", commentHandler.CreatePublic)
	r.Get("/posts/{slug}", publicHandler.GetPost)
	r.Get("/categories", publicHandler.ListCategories)
	r.Get("/tags", publicHandler.ListTags)
	return r
}
