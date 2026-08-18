package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/google/uuid"
)

type routePublicPostStore struct {
	post model.PublicPost
}

func (store routePublicPostStore) ListPublished(context.Context, repository.ListPublishedParams) ([]model.PublicPost, error) {
	return []model.PublicPost{store.post}, nil
}
func (store routePublicPostStore) GetPublishedBySlug(context.Context, string) (*model.PublicPost, error) {
	return &store.post, nil
}
func (routePublicPostStore) ListPublishedTags(context.Context) ([]string, error) {
	return []string{"go"}, nil
}

func TestPublicRoutesNeedNoAuthenticationAndHideListContent(t *testing.T) {
	publishedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	store := routePublicPostStore{post: model.PublicPost{
		Post: model.Post{
			ID: uuid.New(), AuthorID: uuid.New(), CategoryID: uuid.New(),
			Slug: "public-post", Title: "Public Post", Content: "private-list-payload",
			Tags: []string{"go"}, PublishedAt: &publishedAt,
		},
		AuthorNickname: "Writer", CategoryName: "Tech", CategorySlug: "tech",
	}}
	publicHandler := handler.NewPublicHandler(service.NewPublicService(store, routeCategoryStore{}))
	r := PublicRouter(publicHandler, nil, nil)

	listRecorder := httptest.NewRecorder()
	r.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/posts", nil))
	if code := decodeRouteBusinessCode(t, listRecorder); code != http.StatusOK {
		t.Fatalf("GET /posts business code = %d", code)
	}
	if strings.Contains(listRecorder.Body.String(), "private-list-payload") {
		t.Fatalf("list leaked content: %s", listRecorder.Body.String())
	}

	detailRecorder := httptest.NewRecorder()
	r.ServeHTTP(detailRecorder, httptest.NewRequest(http.MethodGet, "/posts/public-post", nil))
	if code := decodeRouteBusinessCode(t, detailRecorder); code != http.StatusOK {
		t.Fatalf("GET /posts/public-post business code = %d", code)
	}
	if !strings.Contains(detailRecorder.Body.String(), "private-list-payload") {
		t.Fatalf("detail omitted content: %s", detailRecorder.Body.String())
	}

	invalidRecorder := httptest.NewRecorder()
	r.ServeHTTP(invalidRecorder, httptest.NewRequest(http.MethodGet, "/posts?cursor=invalid", nil))
	if code := decodeRouteBusinessCode(t, invalidRecorder); code != http.StatusBadRequest {
		t.Fatalf("invalid cursor business code = %d, want 400", code)
	}
}
