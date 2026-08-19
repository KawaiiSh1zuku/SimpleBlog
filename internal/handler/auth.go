package handler

import (
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

type AuthHandler struct {
	Auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler { return &AuthHandler{Auth: auth} }

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request dto.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	result, err := h.Auth.Login(r.Context(), request.Username, request.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.LoginResponse{Token: result.Token, ExpiresAt: result.ExpiresAt})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := appmiddleware.CurrentUser(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "未登录")
		return
	}
	response.Success(w, dto.NewUserResponse(user))
}
