package handler

import (
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PostHandler struct {
	Posts *service.PostService
}

func NewPostHandler(posts *service.PostService) *PostHandler { return &PostHandler{Posts: posts} }

func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := appmiddleware.CurrentUser(r.Context())
	query := r.URL.Query()
	page, err := h.Posts.List(r.Context(), actor, service.ListManagedPostsInput{
		Limit: query.Get("limit"), Cursor: query.Get("cursor"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewManagedPostPageResponse(page.Items, page.NextCursor))
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := postID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	post, err := h.Posts.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreatePostRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	post, err := h.Posts.Create(r.Context(), actor, service.CreatePostInput{
		CategoryID: request.CategoryID, Slug: request.Slug, Title: request.Title,
		Summary: request.Summary, Content: request.Content, Tags: request.Tags,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := postID(w, r)
	if !ok {
		return
	}
	var request dto.UpdatePostRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	post, err := h.Posts.Update(r.Context(), actor, id, service.UpdatePostInput{
		CategoryID: request.CategoryID, Slug: request.Slug, Title: request.Title,
		Summary: request.Summary, Content: request.Content, Tags: request.Tags,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := postID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	if err := h.Posts.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.SuccessWithMsg[any](w, "post deleted")
}

func (h *PostHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id, ok := postID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	post, err := h.Posts.Publish(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (h *PostHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	id, ok := postID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	post, err := h.Posts.Withdraw(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func postID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid post id")
		return uuid.Nil, false
	}
	return id, true
}
