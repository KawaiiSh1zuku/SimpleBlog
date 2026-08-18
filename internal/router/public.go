package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/go-chi/chi/v5"
)

func PublicRouter(publicHandler *handler.PublicHandler) chi.Router {
	r := newRouter()
	r.Get("/posts", publicHandler.ListPosts)
	r.Get("/posts/{slug}", publicHandler.GetPost)
	r.Get("/categories", publicHandler.ListCategories)
	r.Get("/tags", publicHandler.ListTags)
	return r
}
