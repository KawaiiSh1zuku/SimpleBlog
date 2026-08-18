package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
)

type routeCategoryStore struct{}

func (routeCategoryStore) List(context.Context) ([]model.Category, error) {
	return []model.Category{}, nil
}
func (routeCategoryStore) GetByID(context.Context, uuid.UUID) (*model.Category, error) {
	return nil, nil
}
func (routeCategoryStore) Create(_ context.Context, category *model.Category) (*model.Category, error) {
	return category, nil
}
func (routeCategoryStore) Update(context.Context, uuid.UUID, repository.UpdateCategoryParams) (*model.Category, error) {
	return nil, nil
}
func (routeCategoryStore) Delete(context.Context, uuid.UUID) error { return nil }

type routePostStore struct {
	authorID *uuid.UUID
	created  *model.Post
}

func (store *routePostStore) List(_ context.Context, params repository.ListPostsParams) ([]model.PostSummary, error) {
	store.authorID = params.AuthorID
	return []model.PostSummary{}, nil
}
func (*routePostStore) GetByID(context.Context, uuid.UUID) (*model.Post, error) { return nil, nil }

func (store *routePostStore) Create(_ context.Context, post *model.Post) (*model.Post, error) {
	store.created = post
	return post, nil
}
func (*routePostStore) Update(context.Context, uuid.UUID, repository.UpdatePostParams) (*model.Post, error) {
	return nil, nil
}
func (*routePostStore) Delete(context.Context, uuid.UUID) error                  { return nil }
func (*routePostStore) Publish(context.Context, uuid.UUID) (*model.Post, error)  { return nil, nil }
func (*routePostStore) Withdraw(context.Context, uuid.UUID) (*model.Post, error) { return nil, nil }

func TestAdminRoutesAllowContributorPostsButNotCategories(t *testing.T) {
	contributor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	users := &routeUserStore{current: contributor}
	posts := &routePostStore{}
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	_, token, err := tokenAuth.Encode(map[string]any{"sub": contributor.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	r := AdminRouter(
		handler.NewUserHandler(service.NewUserService(users)),
		handler.NewCategoryHandler(service.NewCategoryService(routeCategoryStore{})),
		handler.NewPostHandler(service.NewPostService(posts)),
		nil,
		nil,
		tokenAuth,
		users,
	)

	postRecorder := httptest.NewRecorder()
	postRequest := httptest.NewRequest(http.MethodGet, "/posts", nil)
	postRequest.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(postRecorder, postRequest)
	if code := decodeRouteBusinessCode(t, postRecorder); code != http.StatusOK {
		t.Fatalf("GET /posts business code = %d", code)
	}
	if posts.authorID == nil || *posts.authorID != contributor.ID {
		t.Fatalf("post list author = %v", posts.authorID)
	}

	categoryRecorder := httptest.NewRecorder()
	categoryRequest := httptest.NewRequest(http.MethodGet, "/categories", nil)
	categoryRequest.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(categoryRecorder, categoryRequest)
	if code := decodeRouteBusinessCode(t, categoryRecorder); code != http.StatusForbidden {
		t.Fatalf("GET /categories business code = %d, want 403", code)
	}

	invalidRecorder := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(`{
		"category_id":"`+uuid.NewString()+`","slug":"post","title":"Post",
		"content":"body","status":"published"
	}`))
	invalidRequest.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(invalidRecorder, invalidRequest)
	if code := decodeRouteBusinessCode(t, invalidRecorder); code != http.StatusBadRequest {
		t.Fatalf("POST /posts with status business code = %d, want 400", code)
	}
}
