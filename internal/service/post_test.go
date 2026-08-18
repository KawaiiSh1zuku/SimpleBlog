package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
)

type postStoreStub struct {
	post       *model.Post
	created    *model.Post
	updated    repository.UpdatePostParams
	listAuthor *uuid.UUID
	listLimit  int
	published  bool
	withdrawn  bool
}

func TestPostServiceRejectsOversizedContentAndSummary(t *testing.T) {
	service := NewPostService(&postStoreStub{})
	actor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	base := CreatePostInput{CategoryID: uuid.New(), Slug: "post", Title: "Post", Content: "body"}
	oversizedContent := base
	oversizedContent.Content = strings.Repeat("x", MaxPostContentBytes+1)
	if _, err := service.Create(context.Background(), actor, oversizedContent); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("oversized content error = %v", err)
	}
	oversizedSummary := base
	oversizedSummary.Summary = strings.Repeat("界", maxPostSummaryRunes+1)
	if _, err := service.Create(context.Background(), actor, oversizedSummary); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("oversized summary error = %v", err)
	}
}

func (stub *postStoreStub) List(_ context.Context, params repository.ListPostsParams) ([]model.PostSummary, error) {
	stub.listAuthor = params.AuthorID
	stub.listLimit = params.Limit
	return nil, nil
}
func (stub *postStoreStub) GetByID(context.Context, uuid.UUID) (*model.Post, error) {
	return stub.post, nil
}
func (stub *postStoreStub) Create(_ context.Context, post *model.Post) (*model.Post, error) {
	stub.created = post
	return post, nil
}
func (stub *postStoreStub) Update(_ context.Context, _ uuid.UUID, params repository.UpdatePostParams) (*model.Post, error) {
	stub.updated = params
	return stub.post, nil
}
func (stub *postStoreStub) Delete(context.Context, uuid.UUID) error { return nil }
func (stub *postStoreStub) Publish(context.Context, uuid.UUID) (*model.Post, error) {
	stub.published = true
	result := *stub.post
	result.Status = model.PostStatusPublished
	return &result, nil
}
func (stub *postStoreStub) Withdraw(context.Context, uuid.UUID) (*model.Post, error) {
	stub.withdrawn = true
	result := *stub.post
	result.Status = model.PostStatusDraft
	return &result, nil
}

func TestPostServiceCreateFixesAuthorAndNormalizesTags(t *testing.T) {
	actor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	store := &postStoreStub{}
	service := NewPostService(store)
	created, err := service.Create(context.Background(), actor, CreatePostInput{
		CategoryID: uuid.New(), Slug: " First-Post ", Title: " First ",
		Content: "Markdown $x^2$", Tags: []string{" Go ", "go", "LaTeX", ""},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.AuthorID != actor.ID || created.Status != model.PostStatusDraft {
		t.Fatalf("Create() identity/status = %+v", created)
	}
	if created.Slug != "first-post" || !reflect.DeepEqual(created.Tags, []string{"go", "latex"}) {
		t.Fatalf("Create() normalized values = %+v", created)
	}
	if _, err := service.Create(context.Background(), actor, CreatePostInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid Create() error = %v", err)
	}
}

func TestPostServiceScopesContributorAndAdmin(t *testing.T) {
	ownerID := uuid.New()
	post := &model.Post{ID: uuid.New(), AuthorID: ownerID, Status: model.PostStatusDraft}
	store := &postStoreStub{post: post}
	service := NewPostService(store)
	contributor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	if _, err := service.Get(context.Background(), contributor, post.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other contributor Get() error = %v", err)
	}
	if _, err := service.Get(context.Background(), &model.User{Role: model.RoleAdmin}, post.ID); err != nil {
		t.Fatalf("admin Get() error = %v", err)
	}
	if _, err := service.List(context.Background(), contributor, ListManagedPostsInput{}); err != nil {
		t.Fatalf("contributor List() error = %v", err)
	}
	if store.listAuthor == nil || *store.listAuthor != contributor.ID {
		t.Fatalf("contributor list author = %v", store.listAuthor)
	}
	if _, err := service.List(context.Background(), &model.User{Role: model.RoleAdmin}, ListManagedPostsInput{}); err != nil {
		t.Fatalf("admin List() error = %v", err)
	}
	if store.listAuthor != nil {
		t.Fatalf("admin list author = %v, want nil", store.listAuthor)
	}
	if store.listLimit != defaultManagedPageSize+1 {
		t.Fatalf("managed post repository limit = %d", store.listLimit)
	}
}

func TestPostServiceEnforcesLifecycle(t *testing.T) {
	actor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	post := &model.Post{ID: uuid.New(), AuthorID: actor.ID, Status: model.PostStatusDraft}
	store := &postStoreStub{post: post}
	service := NewPostService(store)
	published, err := service.Publish(context.Background(), actor, post.ID)
	if err != nil || published.Status != model.PostStatusPublished || !store.published {
		t.Fatalf("Publish() = %+v, %v", published, err)
	}
	if _, err := service.Withdraw(context.Background(), actor, post.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("Withdraw(draft) error = %v", err)
	}
	post.Status = model.PostStatusPublished
	withdrawn, err := service.Withdraw(context.Background(), actor, post.ID)
	if err != nil || withdrawn.Status != model.PostStatusDraft || !store.withdrawn {
		t.Fatalf("Withdraw() = %+v, %v", withdrawn, err)
	}
	if _, err := service.Publish(context.Background(), actor, post.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("Publish(published) error = %v", err)
	}
}

func TestNormalizeTagsRejectsLimitAfterDeduplication(t *testing.T) {
	tags := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven"}
	if _, err := normalizeTags(tags); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("normalizeTags() error = %v", err)
	}
}
