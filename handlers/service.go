package handlers

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/pkg/response"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

// ServiceHandler 结构体，持有依赖（如数据库连接池、服务层）
type ServiceHandler struct{}

// NewServiceHandler 构造函数，用于注入依赖
func NewServiceHandler() *ServiceHandler {
	return nil
}

// ------------------- 方法即 Handler -------------------

// Ping GET /service/ping
func (h *ServiceHandler) Ping(w http.ResponseWriter, r *http.Request) {
	clientIP := middleware.GetClientIP(r.Context())
	response.Success(w, "Hello, "+clientIP)
}
