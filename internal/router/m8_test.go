package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
)

type routeM8LinkStore struct{ links []model.FriendLink }

func (s routeM8LinkStore) List(context.Context) ([]model.FriendLink, error) { return s.links, nil }
func (routeM8LinkStore) GetByID(context.Context, uuid.UUID) (*model.FriendLink, error) {
	return nil, nil
}
func (routeM8LinkStore) Create(_ context.Context, link *model.FriendLink) (*model.FriendLink, error) {
	return link, nil
}
func (routeM8LinkStore) Update(context.Context, uuid.UUID, repository.UpdateLinkParams) (*model.FriendLink, error) {
	return nil, nil
}
func (routeM8LinkStore) Delete(context.Context, uuid.UUID) error { return nil }

type routeM8FeedStore struct{ err error }

func (s routeM8FeedStore) ListArticles(context.Context) ([]model.FeedArticle, error) {
	return []model.FeedArticle{{ID: uuid.New(), Slug: "post", Title: "Post", Summary: "Summary", PublishedAt: time.Unix(1, 0)}}, s.err
}
func (s routeM8FeedStore) ListComments(context.Context) ([]model.FeedComment, error) {
	return nil, s.err
}

func TestM8PublicLinkAndRSSRoutes(t *testing.T) {
	linkHandler := handler.NewLinkHandler(service.NewLinkService(routeM8LinkStore{links: []model.FriendLink{{ID: uuid.New(), Name: "Example", URL: "https://example.com"}}}))
	publicRoutes := newRouter()
	publicRoutes.Get("/links", linkHandler.ListPublic)
	rssHandler := handler.NewRSSHandler(service.NewRSSService(routeM8FeedStore{}, service.RSSSite{Name: "Blog", Description: "Description", BaseURL: "https://blog.example.com"}))
	root := New(newRouter(), newRouter(), newRouter(), RSSRouter(rssHandler), publicRoutes, false)

	links := httptest.NewRecorder()
	root.ServeHTTP(links, httptest.NewRequest(http.MethodGet, "/links", nil))
	if links.Code != http.StatusOK || !strings.Contains(links.Body.String(), "https://example.com") {
		t.Fatalf("GET /links = %d %s", links.Code, links.Body.String())
	}

	feed := httptest.NewRecorder()
	root.ServeHTTP(feed, httptest.NewRequest(http.MethodGet, "/rss/article", nil))
	if feed.Code != http.StatusOK || feed.Header().Get("Content-Type") != "application/rss+xml; charset=utf-8" || !strings.HasPrefix(feed.Body.String(), "<?xml") {
		t.Fatalf("GET /rss/article = %d %q %s", feed.Code, feed.Header().Get("Content-Type"), feed.Body.String())
	}
	if feed.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header: %#v", feed.Header())
	}
}

func TestRSSFailureUsesRealHTTP500(t *testing.T) {
	rssHandler := handler.NewRSSHandler(service.NewRSSService(routeM8FeedStore{err: errors.New("database unavailable")}, service.RSSSite{}))
	recorder := httptest.NewRecorder()
	RSSRouter(rssHandler).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/article", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("RSS HTTP status = %d, want 500", recorder.Code)
	}
}

func TestAdminLinkRoutesUseDatabaseRole(t *testing.T) {
	linkHandler := handler.NewLinkHandler(service.NewLinkService(routeM8LinkStore{}))
	for _, test := range []struct {
		name     string
		role     model.Role
		wantCode int
	}{
		{name: "admin", role: model.RoleAdmin, wantCode: http.StatusOK},
		{name: "contributor", role: model.RoleContributor, wantCode: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			user := &model.User{ID: uuid.New(), Role: test.role}
			users := &routeUserStore{current: user}
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			_, token, err := tokenAuth.Encode(map[string]any{"sub": user.ID.String()})
			if err != nil {
				t.Fatal(err)
			}
			routes := AdminRouter(nil, nil, nil, nil, nil, linkHandler, tokenAuth, users)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/links", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			routes.ServeHTTP(recorder, request)
			if code := decodeRouteBusinessCode(t, recorder); code != test.wantCode {
				t.Fatalf("business code = %d, want %d", code, test.wantCode)
			}
		})
	}
}
