package handler

import (
	"encoding/json"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/KawaiiSh1zuku/SimpleBlog/pkg/response"
	"net/http"
)

// AdminHandler 结构体，持有依赖（如数据库连接池、服务层）
type AdminHandler struct {
	AuthService *service.AuthService
}

// NewAdminHandler 构造函数，用于注入依赖
func NewAdminHandler(authService *service.AuthService) *AdminHandler {
	return &AdminHandler{
		AuthService: authService,
	}
}

// LoginRequest 用于登录请求的结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ------------------- 方法即 Handler -------------------

// Login POST /admin/login
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}

	token, err := h.AuthService.Login(
		r.Context(),
		req.Username,
		req.Password,
	)

	if err != nil {
		response.Error(
			w,
			http.StatusUnauthorized,
			"invalid username or password",
		)
		return
	}

	response.Success(w, map[string]string{
		"token": token,
	})
}
