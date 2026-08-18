package router

import (
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"net/http"
)

// New 创建应用根路由并安装所有全局中间件和兜底响应。
func New(serviceRoutes, authRoutes, adminRoutes http.Handler, debug bool) chi.Router {
	r := newRouter()
	r.Use(chimiddleware.RequestID)
	if debug {
		r.Use(chimiddleware.ClientIPFromRemoteAddr)
	} else {
		r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	}
	r.Use(appmiddleware.Recoverer)

	r.Mount("/service", serviceRoutes)
	r.Mount("/auth", authRoutes)
	r.Mount("/admin", adminRoutes)

	return r
}

func newRouter() chi.Router {
	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})
	return r
}
