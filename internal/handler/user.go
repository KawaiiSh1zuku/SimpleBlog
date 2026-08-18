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

type UserHandler struct {
	Users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler { return &UserHandler{Users: users} }

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	actor, _ := appmiddleware.CurrentUser(r.Context())
	users, err := h.Users.List(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewUserResponses(users))
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	user, err := h.Users.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewUserResponse(user))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateUserRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	user, err := h.Users.Create(r.Context(), actor, service.CreateUserInput{
		Username: request.Username, Password: request.Password, Nickname: request.Nickname,
		AvatarURL: request.AvatarURL, Role: request.Role,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewUserResponse(user))
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	var request dto.UpdateUserRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	user, err := h.Users.Update(r.Context(), actor, id, service.UpdateUserInput{
		Username: request.Username, Nickname: request.Nickname,
		AvatarURL: request.AvatarURL, Role: request.Role,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewUserResponse(user))
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	if err := h.Users.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.SuccessWithMsg[any](w, "user deleted")
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}
	var request dto.UpdatePasswordRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	if err := h.Users.UpdatePassword(r.Context(), actor, id, request.Password); err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.SuccessWithMsg[any](w, "password updated")
}

func userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return uuid.Nil, false
	}
	return id, true
}
