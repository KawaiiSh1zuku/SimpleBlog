package routers

import (
    "github.com/KawaiiSh1zuku/SimpleBlog/handlers"
    "github.com/go-chi/chi/v5"
)

// ServiceRouter 返回一个包含所有服务相关路由的 chi.Router
func ServiceRouter(serviceHandler *handlers.ServiceHandler) chi.Router {
    r := chi.NewRouter()

    // 可以在这里添加仅对用户路由生效的中间件
    // r.Use(SomeUserMiddleware)

    r.Get("/ping", serviceHandler.Ping) // GET /service/ping

    return r
}
