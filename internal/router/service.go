package router

import (
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/chi/v5"
)

// ServiceRouter 返回一个包含所有服务相关路由的 chi.Router
func ServiceRouter(serviceHandler *handler.ServiceHandler, turnstileService *service.TurnstileService) chi.Router {
	r := newRouter()

	// 可以在这里添加仅对服务路由生效的中间件
	// r.Use(SomeUserMiddleware)

	r.Get("/ping", serviceHandler.Ping) // GET /service/ping

	r.With(appmiddleware.TurnstileMiddleware(turnstileService)).
		Get("/test_turnstile", serviceHandler.TestTurnstile)

	return r
}
