package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/go-chi/chi/v5"
)

func RSSRouter(rssHandler *handler.RSSHandler) chi.Router {
	r := newRouter()
	r.Get("/article", rssHandler.Articles)
	r.Get("/comment", rssHandler.Comments)
	return r
}
