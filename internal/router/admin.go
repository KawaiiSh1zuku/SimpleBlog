package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/go-chi/chi/v5"
)

// AdminRouter 返回一个包含所有用户（管理员）相关路由的 chi.Router
func AdminRouter(adminHandler *handler.AdminHandler) chi.Router {
	r := chi.NewRouter()

	// 登录不需要 JWT
	r.Post("/login", adminHandler.Login) // POST /admin/login

	return r
}
