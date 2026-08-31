package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const (
	adminToken       = "mock-admin-token"
	contributorToken = "mock-contributor-token"
	maxBodyBytes     = 11 << 20
)

type actorContextKey struct{}

type mockServer struct {
	store  *mockStore
	router http.Handler
}

func newMockServer() http.Handler { return newMockApp() }

func newMockApp() *mockServer {
	store := newSeedStore()
	app := &mockServer{store: store}
	app.router = app.routes()
	return app
}

func (app *mockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.router.ServeHTTP(w, r)
}

func (app *mockServer) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(app.cors)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/service/ping", app.ping)
	r.With(app.requireTurnstile).Get("/service/test_turnstile", app.testTurnstile)
	r.With(app.requireTurnstile).Post("/auth/login", app.login)
	r.With(app.authenticate).Get("/auth/me", app.me)

	r.Get("/rss/article", app.articleRSS)
	r.Get("/rss/comment", app.commentRSS)
	r.Post("/mock-upload", app.mockUpload)
	r.Get("/mock-assets/{asset}", app.mockAsset)

	r.Get("/search", app.searchPosts)
	r.Get("/posts", app.listPublicPosts)
	r.Get("/posts/{slug}/comments", app.listPublicComments)
	r.With(app.requireTurnstile).Post("/posts/{slug}/comments", app.createPublicComment)
	r.Get("/posts/{slug}", app.getPublicPost)
	r.Get("/categories", app.listPublicCategories)
	r.Get("/tags", app.listPublicTags)
	r.Get("/links", app.listPublicLinks)

	r.Route("/admin", func(admin chi.Router) {
		admin.Use(app.authenticate)
		admin.With(app.requireAdmin).Get("/users", app.listUsers)
		admin.With(app.requireAdmin).Post("/users", app.createUser)
		admin.With(app.requireAdmin).Get("/users/{userID}", app.getUser)
		admin.With(app.requireAdmin).Patch("/users/{userID}", app.updateUser)
		admin.With(app.requireAdmin).Delete("/users/{userID}", app.deleteUser)
		admin.With(app.requireAdmin).Put("/users/{userID}/password", app.updateUserPassword)

		admin.With(app.requireAdmin).Get("/categories", app.listCategories)
		admin.With(app.requireAdmin).Post("/categories", app.createCategory)
		admin.With(app.requireAdmin).Get("/categories/{categoryID}", app.getCategory)
		admin.With(app.requireAdmin).Patch("/categories/{categoryID}", app.updateCategory)
		admin.With(app.requireAdmin).Delete("/categories/{categoryID}", app.deleteCategory)

		admin.With(app.requireAdmin).Get("/links", app.listLinks)
		admin.With(app.requireAdmin).Post("/links", app.createLink)
		admin.With(app.requireAdmin).Get("/links/{linkID}", app.getLink)
		admin.With(app.requireAdmin).Patch("/links/{linkID}", app.updateLink)
		admin.With(app.requireAdmin).Delete("/links/{linkID}", app.deleteLink)

		admin.Get("/posts", app.listPosts)
		admin.Post("/posts", app.createPost)
		admin.Get("/posts/{postID}", app.getPost)
		admin.Patch("/posts/{postID}", app.updatePost)
		admin.Delete("/posts/{postID}", app.deletePost)
		admin.Post("/posts/{postID}/publish", app.publishPost)
		admin.Post("/posts/{postID}/withdraw", app.withdrawPost)

		admin.Get("/comments", app.listComments)
		admin.Post("/comments/{commentID}/hide", app.hideComment)
		admin.Post("/comments/{commentID}/show", app.showComment)
		admin.Delete("/comments/{commentID}", app.deleteComment)
		admin.Post("/uploads/presign", app.presignUpload)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusNotFound, "找不到资源")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusMethodNotAllowed, "请求方法错误")
	})
	return r
}

func (app *mockServer) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Turnstile-Response")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *mockServer) requireTurnstile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("X-Turnstile-Response")) == "" {
			response.Error(w, http.StatusBadRequest, "Turnstile 验证失败")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *mockServer) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(w, http.StatusUnauthorized, "未登录")
			return
		}
		userID, ok := userIDForToken(parts[1])
		if !ok {
			response.Error(w, http.StatusUnauthorized, "未登录")
			return
		}
		app.store.mu.RLock()
		user, ok := app.store.userByID(userID)
		if ok {
			copy := *user
			user = &copy
		}
		app.store.mu.RUnlock()
		if !ok {
			response.Error(w, http.StatusUnauthorized, "未登录")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey{}, user)))
	})
}

func userIDForToken(token string) (uuid.UUID, bool) {
	switch token {
	case adminToken:
		return seedID("user-admin"), true
	case contributorToken:
		return seedID("user-writer"), true
	}
	if !strings.HasPrefix(token, "mock-user-") {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(strings.TrimPrefix(token, "mock-user-"))
	return id, err == nil
}

func (app *mockServer) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := currentActor(r)
		if actor == nil || actor.Role != model.RoleAdmin {
			response.Error(w, http.StatusForbidden, "无权访问")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func currentActor(r *http.Request) *model.User {
	actor, _ := r.Context().Value(actorContextKey{}).(*model.User)
	return actor
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeDecodeError(w, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
		return false
	}
	return true
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		response.Error(w, http.StatusRequestEntityTooLarge, "请求体大小异常")
		return
	}
	response.Error(w, http.StatusBadRequest, "错误请求")
}

func pathID(w http.ResponseWriter, r *http.Request, name, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		response.Error(w, http.StatusBadRequest, message)
		return uuid.Nil, false
	}
	return id, true
}

func parsePage(r *http.Request, defaultLimit int) (int, int, bool) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return 0, 0, false
		}
		limit = value
	}
	offset := 0
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || !strings.HasPrefix(string(decoded), "mock:") {
			return 0, 0, false
		}
		value, err := strconv.Atoi(strings.TrimPrefix(string(decoded), "mock:"))
		if err != nil || value < 0 {
			return 0, 0, false
		}
		offset = value
	}
	return limit, offset, true
}

func paginate[T any](values []T, limit, offset int) ([]T, string) {
	if offset >= len(values) {
		return []T{}, ""
	}
	end := min(offset+limit, len(values))
	page := values[offset:end]
	if end == len(values) {
		return page, ""
	}
	return page, base64.RawURLEncoding.EncodeToString([]byte("mock:" + strconv.Itoa(end)))
}

func validNonEmpty(value string, maxRunes int) bool {
	value = strings.TrimSpace(value)
	return value != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validSlug(value string) bool {
	if value == "" || len(value) > 160 {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || (char == '-' && index > 0 && index < len(value)-1) {
			continue
		}
		return false
	}
	return !strings.Contains(value, "--")
}

func conflict(w http.ResponseWriter)   { response.Error(w, http.StatusConflict, "资源冲突") }
func notFound(w http.ResponseWriter)   { response.Error(w, http.StatusNotFound, "找不到资源") }
func badRequest(w http.ResponseWriter) { response.Error(w, http.StatusBadRequest, "错误请求") }

func (app *mockServer) ping(w http.ResponseWriter, r *http.Request) {
	host := r.RemoteAddr
	if separator := strings.LastIndex(host, ":"); separator > 0 {
		host = host[:separator]
	}
	response.Success(w, "Hello, "+host)
}

func (app *mockServer) testTurnstile(w http.ResponseWriter, _ *http.Request) {
	response.Success(w, "Turnstile verification passed!")
}

func nowUTC() time.Time { return time.Now().UTC().Truncate(time.Second) }
