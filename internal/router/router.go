package router

import (
	"io"
	"net/http"

	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// New 创建应用根路由并安装所有全局中间件和兜底响应。
func New(serviceRoutes, authRoutes, adminRoutes, rssRoutes, publicRoutes http.Handler, accessWriter io.Writer, trustProxyHeaders bool, corsAllowedOrigins []string) chi.Router {
	r := newRouter()
	r.Use(chimiddleware.RequestID)
	if trustProxyHeaders {
		r.Use(chimiddleware.ClientIPFromHeader("X-Real-IP"))
	} else {
		r.Use(chimiddleware.ClientIPFromRemoteAddr)
	}
	r.Use(appmiddleware.AccessLog(accessWriter))
	r.Use(appmiddleware.CORS(corsAllowedOrigins))
	r.Use(appmiddleware.LimitRequestBody(appmiddleware.MaxRequestBodyBytes))
	r.Use(appmiddleware.Recoverer)
	r.Use(appmiddleware.SecurityHeaders)

	r.Mount("/service", serviceRoutes)
	r.Mount("/auth", authRoutes)
	r.Mount("/admin", adminRoutes)
	r.Mount("/rss", rssRoutes)
	r.Mount("/", publicRoutes)

	return r
}

func newRouter() chi.Router {
	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "找不到资源")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "请求方法错误")
	})
	return r
}
