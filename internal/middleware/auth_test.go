package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type currentUserStoreStub struct {
	user *model.User
	err  error
}

func (stub currentUserStoreStub) GetByID(context.Context, uuid.UUID) (*model.User, error) {
	return stub.user, stub.err
}

func TestAuthenticateReloadsCurrentUserAndChecksRole(t *testing.T) {
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	user := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	_, token, err := tokenAuth.Encode(map[string]any{"sub": user.ID.String(), "role": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Use(jwtauth.Verifier(tokenAuth))
	r.Use(Authenticate(currentUserStoreStub{user: user}))
	r.Use(RequireRole(model.RoleAdmin))
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { response.Success(w, "ok") })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	assertBusinessCode(t, recorder, http.StatusForbidden)
}

func TestAuthenticateRejectsMissingOrDeletedUser(t *testing.T) {
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	_, token, err := tokenAuth.Encode(map[string]any{"sub": uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name  string
		token string
		store currentUserStoreStub
	}{
		{name: "missing token", store: currentUserStoreStub{}},
		{name: "deleted user", token: token, store: currentUserStoreStub{err: pgx.ErrNoRows}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Use(jwtauth.Verifier(tokenAuth))
			r.Use(Authenticate(test.store))
			r.Get("/", func(w http.ResponseWriter, _ *http.Request) { t.Fatal("protected handler was called") })
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)
			assertBusinessCode(t, recorder, http.StatusUnauthorized)
		})
	}
}

func TestAuthenticateReturnsInternalErrorForStoreFailure(t *testing.T) {
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	_, token, _ := tokenAuth.Encode(map[string]any{"sub": uuid.NewString()})
	r := chi.NewRouter()
	r.Use(jwtauth.Verifier(tokenAuth))
	r.Use(Authenticate(currentUserStoreStub{err: errors.New("database unavailable")}))
	r.Get("/", func(http.ResponseWriter, *http.Request) {})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	assertBusinessCode(t, recorder, http.StatusInternalServerError)
}

func assertBusinessCode(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d", recorder.Code)
	}
	var body response.Response[any]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != want {
		t.Fatalf("business code = %d, want %d", body.Code, want)
	}
}
