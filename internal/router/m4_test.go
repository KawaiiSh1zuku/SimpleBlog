package router

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/google/uuid"
)

type routeCommentStore struct {
	created *model.Comment
}

func (store *routeCommentStore) CreateForPublishedPost(_ context.Context, _ string, comment *model.Comment) (*model.Comment, error) {
	store.created = comment
	return comment, nil
}
func (*routeCommentStore) ListPublicByPostSlug(context.Context, string) ([]model.Comment, bool, error) {
	return []model.Comment{}, true, nil
}
func (*routeCommentStore) ListManaged(context.Context, repository.ListCommentsParams) ([]model.ManagedComment, error) {
	return nil, nil
}
func (*routeCommentStore) GetManagedByID(context.Context, uuid.UUID) (*model.ManagedComment, error) {
	return nil, nil
}
func (*routeCommentStore) Hide(context.Context, uuid.UUID) (*model.Comment, error) {
	return nil, nil
}
func (*routeCommentStore) Show(context.Context, uuid.UUID) (*model.Comment, error) {
	return nil, nil
}
func (*routeCommentStore) Delete(context.Context, uuid.UUID) (*model.Comment, error) {
	return nil, nil
}

type turnstileVerifierStub struct {
	token    string
	remoteIP string
	err      error
}

func (stub *turnstileVerifierStub) Verify(_ context.Context, token, remoteIP string) error {
	stub.token = token
	stub.remoteIP = remoteIP
	return stub.err
}

func TestPublicCommentCreateRequiresTurnstileAndNeverReturnsEmail(t *testing.T) {
	commentStore := &routeCommentStore{}
	turnstile := &turnstileVerifierStub{}
	publicHandler := handler.NewPublicHandler(service.NewPublicService(routePublicPostStore{}, routeCategoryStore{}))
	commentHandler := handler.NewCommentHandler(service.NewCommentService(commentStore), "https://avatar.example/avatar")
	r := PublicRouter(publicHandler, commentHandler, turnstile)
	body := []byte(`{"nickname":"Visitor","email":"visitor@example.com","content":"Hello"}`)

	missingRecorder := httptest.NewRecorder()
	r.ServeHTTP(missingRecorder, httptest.NewRequest(http.MethodPost, "/posts/post/comments", bytes.NewReader(body)))
	if code := decodeRouteBusinessCode(t, missingRecorder); code != http.StatusUnauthorized {
		t.Fatalf("missing Turnstile business code = %d", code)
	}

	request := httptest.NewRequest(http.MethodPost, "/posts/post/comments", bytes.NewReader(body))
	request.Header.Set("X-Turnstile-Token", "verified-token")
	request.RemoteAddr = "203.0.113.8:4321"
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	if code := decodeRouteBusinessCode(t, recorder); code != http.StatusOK {
		t.Fatalf("comment create business code = %d", code)
	}
	if turnstile.token != "verified-token" || turnstile.remoteIP != "203.0.113.8" {
		t.Fatalf("Turnstile arguments = %q, %q", turnstile.token, turnstile.remoteIP)
	}
	if strings.Contains(recorder.Body.String(), "visitor@example.com") || strings.Contains(recorder.Body.String(), "email") {
		t.Fatalf("public create leaked email: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "https://avatar.example/avatar/") {
		t.Fatalf("public create did not return avatar URL: %s", recorder.Body.String())
	}
	if commentStore.created == nil || commentStore.created.Email != "visitor@example.com" {
		t.Fatalf("repository did not receive email: %+v", commentStore.created)
	}
}

func TestPublicCommentCreateMapsFailedTurnstileToForbidden(t *testing.T) {
	turnstile := &turnstileVerifierStub{err: errors.New("invalid token")}
	r := PublicRouter(
		handler.NewPublicHandler(service.NewPublicService(routePublicPostStore{}, routeCategoryStore{})),
		handler.NewCommentHandler(service.NewCommentService(&routeCommentStore{}), "https://avatar.example/avatar"),
		turnstile,
	)
	request := httptest.NewRequest(http.MethodPost, "/posts/post/comments", strings.NewReader(`{}`))
	request.Header.Set("X-Turnstile-Token", "invalid")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	if code := decodeRouteBusinessCode(t, recorder); code != http.StatusForbidden {
		t.Fatalf("failed Turnstile business code = %d", code)
	}
}
