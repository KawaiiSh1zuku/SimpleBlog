package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
)

type routeUserStore struct {
	current *model.User
	users   []model.User
}

func (store *routeUserStore) GetByUsername(context.Context, string) (*model.User, error) {
	return store.current, nil
}
func (store *routeUserStore) GetByID(context.Context, uuid.UUID) (*model.User, error) {
	return store.current, nil
}
func (store *routeUserStore) List(context.Context) ([]model.User, error) { return store.users, nil }
func (store *routeUserStore) Create(_ context.Context, user *model.User) (*model.User, error) {
	return user, nil
}
func (store *routeUserStore) CreateBootstrapAdmin(_ context.Context, user *model.User) (*model.User, error) {
	return user, nil
}
func (store *routeUserStore) Update(context.Context, uuid.UUID, repository.UpdateUserParams) (*model.User, error) {
	return store.current, nil
}
func (store *routeUserStore) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (store *routeUserStore) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }

func TestM1RoutesHaveNoAPIPrefixAndUseDatabaseRole(t *testing.T) {
	admin := &model.User{
		ID: uuid.New(), Username: "admin", PasswordHash: "must-not-leak",
		Nickname: "Administrator", Role: model.RoleAdmin,
	}
	store := &routeUserStore{current: admin, users: []model.User{*admin}}
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	authHandler := handler.NewAuthHandler(service.NewAuthService(store, tokenAuth))
	userHandler := handler.NewUserHandler(service.NewUserService(store))
	r := New(
		chi.NewRouter(),
		AuthRouter(authHandler, tokenAuth, store),
		AdminRouter(userHandler, nil, nil, nil, tokenAuth, store),
		chi.NewRouter(),
		true,
	)
	// The role claim is deliberately stale. Authorization must use the database role.
	_, token, err := tokenAuth.Encode(map[string]any{"sub": admin.ID.String(), "role": "contributor"})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/auth/me", "/admin/users"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(recorder, request)
		if code := decodeRouteBusinessCode(t, recorder); code != http.StatusOK {
			t.Fatalf("GET %s business code = %d", path, code)
		}
		if strings.Contains(recorder.Body.String(), "must-not-leak") || strings.Contains(recorder.Body.String(), "password_hash") {
			t.Fatalf("GET %s leaked password data: %s", path, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if code := decodeRouteBusinessCode(t, recorder); code != http.StatusNotFound {
		t.Fatalf("legacy /api path business code = %d, want 404", code)
	}
}

func decodeRouteBusinessCode(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d", recorder.Code)
	}
	var body response.Response[any]
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Code
}
