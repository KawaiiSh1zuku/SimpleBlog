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

type CategoryHandler struct {
	Categories *service.CategoryService
}

func NewCategoryHandler(categories *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{Categories: categories}
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := appmiddleware.CurrentUser(r.Context())
	categories, err := h.Categories.List(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewCategoryResponses(categories))
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := categoryID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	category, err := h.Categories.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewCategoryResponse(category))
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateCategoryRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	category, err := h.Categories.Create(r.Context(), actor, service.CreateCategoryInput{
		Name: request.Name, Slug: request.Slug, Description: request.Description,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewCategoryResponse(category))
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := categoryID(w, r)
	if !ok {
		return
	}
	var request dto.UpdateCategoryRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	category, err := h.Categories.Update(r.Context(), actor, id, service.UpdateCategoryInput{
		Name: request.Name, Slug: request.Slug, Description: request.Description,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewCategoryResponse(category))
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := categoryID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	if err := h.Categories.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.SuccessWithMsg[any](w, "category deleted")
}

func categoryID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "categoryID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid category id")
		return uuid.Nil, false
	}
	return id, true
}
