package handler

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

// ServiceHandler 结构体，持有依赖（如数据库连接池、服务层）
type ServiceHandler struct {
	//
}

// NewServiceHandler 构造函数，用于注入依赖
func NewServiceHandler() *ServiceHandler {
	return &ServiceHandler{
		// Turnstile: turnstileService,
	}
}

// ------------------- 方法即 Handler -------------------

// Ping GET /service/ping
func (h *ServiceHandler) Ping(w http.ResponseWriter, r *http.Request) {
	clientIP := middleware.GetClientIP(r.Context())
	response.Success(w, "Hello, "+clientIP)
}

// TestTurnstile GET /service/test_turnstile
func (h *ServiceHandler) TestTurnstile(w http.ResponseWriter, r *http.Request) {
	response.Success(w, "Turnstile verification passed!")
}
