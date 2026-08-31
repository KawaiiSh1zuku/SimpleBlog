package handler

import (
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/chi/v5"
)

type PublicHandler struct {
	Public *service.PublicService
}

func NewPublicHandler(public *service.PublicService) *PublicHandler {
	return &PublicHandler{Public: public}
}

func (h *PublicHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := h.Public.ListPosts(r.Context(), service.ListPublicPostsInput{
		Limit: query.Get("limit"), Cursor: query.Get("cursor"),
		CategorySlug: query.Get("category"), Tag: query.Get("tag"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicPostPageResponse(page.Items, page.NextCursor))
}

func (h *PublicHandler) SearchPosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := h.Public.SearchPosts(r.Context(), service.SearchPublicPostsInput{
		Query: query.Get("q"), Limit: query.Get("limit"), Cursor: query.Get("cursor"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicPostPageResponse(page.Items, page.NextCursor))
}

func (h *PublicHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	post, err := h.Public.GetPost(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicPostDetail(post))
}

func (h *PublicHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Public.ListCategories(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicCategoryResponses(categories))
}

func (h *PublicHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.Public.ListTags(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, tags)
}
