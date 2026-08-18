package handler

import (
	"context"
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CommentHandler struct {
	Comments        *service.CommentService
	GravatarBaseURL string
}

func NewCommentHandler(comments *service.CommentService, gravatarBaseURL string) *CommentHandler {
	return &CommentHandler{Comments: comments, GravatarBaseURL: gravatarBaseURL}
}

func (h *CommentHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	comments, err := h.Comments.ListPublic(r.Context(), chi.URLParam(r, "slug"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicCommentTree(comments, h.GravatarBaseURL))
}

func (h *CommentHandler) CreatePublic(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateCommentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	comment, err := h.Comments.Create(r.Context(), chi.URLParam(r, "slug"), service.CreateCommentInput{
		ParentID: request.ParentID, Nickname: request.Nickname,
		Email: request.Email, Content: request.Content,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewPublicCommentResponse(comment, h.GravatarBaseURL))
}

func (h *CommentHandler) ListManaged(w http.ResponseWriter, r *http.Request) {
	actor, _ := appmiddleware.CurrentUser(r.Context())
	query := r.URL.Query()
	page, err := h.Comments.ListManaged(r.Context(), actor, service.ListManagedCommentsInput{
		Limit: query.Get("limit"), Cursor: query.Get("cursor"),
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewManagedCommentPageResponse(page.Items, page.NextCursor))
}

func (h *CommentHandler) Hide(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, h.Comments.Hide)
}

func (h *CommentHandler) Show(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, h.Comments.Show)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, h.Comments.Delete)
}

func (h *CommentHandler) moderate(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, *model.User, uuid.UUID) (*model.Comment, error),
) {
	id, ok := commentID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	comment, err := action(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewCommentModerationResponse(comment))
}

func commentID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "commentID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid comment id")
		return uuid.Nil, false
	}
	return id, true
}
