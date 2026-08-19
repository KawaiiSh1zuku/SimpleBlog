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

type LinkHandler struct{ Links *service.LinkService }

func NewLinkHandler(links *service.LinkService) *LinkHandler { return &LinkHandler{Links: links} }

func (h *LinkHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	links, err := h.Links.ListPublic(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewLinkResponses(links))
}

func (h *LinkHandler) ListManaged(w http.ResponseWriter, r *http.Request) {
	actor, _ := appmiddleware.CurrentUser(r.Context())
	links, err := h.Links.ListManaged(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewLinkResponses(links))
}

func (h *LinkHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := linkID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	link, err := h.Links.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewLinkResponse(link))
}

func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateLinkRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	link, err := h.Links.Create(r.Context(), actor, service.CreateLinkInput{Name: request.Name, Description: request.Description, URL: request.URL, FaviconURL: request.FaviconURL})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewLinkResponse(link))
}

func (h *LinkHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := linkID(w, r)
	if !ok {
		return
	}
	var request dto.UpdateLinkRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	link, err := h.Links.Update(r.Context(), actor, id, service.UpdateLinkInput{Name: request.Name, Description: request.Description, URL: request.URL, FaviconURL: request.FaviconURL})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewLinkResponse(link))
}

func (h *LinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := linkID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	if err := h.Links.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.SuccessWithMsg[any](w, "删除成功")
}

func linkID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "友链 ID 错误")
		return uuid.Nil, false
	}
	return id, true
}
