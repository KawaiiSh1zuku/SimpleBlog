package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type publicPostStoreStub struct {
	posts  []model.PublicPost
	post   *model.PublicPost
	tags   []string
	err    error
	params repository.ListPublishedParams
}

func (stub *publicPostStoreStub) ListPublished(_ context.Context, params repository.ListPublishedParams) ([]model.PublicPost, error) {
	stub.params = params
	return stub.posts, stub.err
}
func (stub *publicPostStoreStub) GetPublishedBySlug(context.Context, string) (*model.PublicPost, error) {
	return stub.post, stub.err
}
func (stub *publicPostStoreStub) ListPublishedTags(context.Context) ([]string, error) {
	return stub.tags, stub.err
}

type publicCategoryStoreStub struct {
	categories []model.Category
}

func (stub publicCategoryStoreStub) List(context.Context) ([]model.Category, error) {
	return stub.categories, nil
}

func TestPublicServiceUsesStableCursorAndLimit(t *testing.T) {
	base := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	posts := make([]model.PublicPost, 21)
	for index := range posts {
		publishedAt := base.Add(-time.Duration(index) * time.Minute)
		posts[index] = model.PublicPost{Post: model.Post{ID: uuid.New(), PublishedAt: &publishedAt}}
	}
	store := &publicPostStoreStub{posts: posts}
	service := NewPublicService(store, publicCategoryStoreStub{})
	page, err := service.ListPosts(context.Background(), ListPublicPostsInput{
		CategorySlug: " News ", Tag: " Go ",
	})
	if err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if len(page.Items) != 20 || page.NextCursor == "" {
		t.Fatalf("page size/cursor = %d, %q", len(page.Items), page.NextCursor)
	}
	if store.params.Limit != 21 || store.params.CategorySlug != "news" || store.params.Tag != "go" {
		t.Fatalf("repository params = %+v", store.params)
	}
	cursor, err := decodePublicCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("decodePublicCursor() error = %v", err)
	}
	last := page.Items[19]
	if cursor.ID != last.ID || !cursor.PublishedAt.Equal(*last.PublishedAt) {
		t.Fatalf("cursor = %+v, last = %+v", cursor, last)
	}
}

func TestPublicServiceValidatesPaginationAndFilters(t *testing.T) {
	service := NewPublicService(&publicPostStoreStub{}, publicCategoryStoreStub{})
	for _, input := range []ListPublicPostsInput{
		{Limit: "0"},
		{Limit: "101"},
		{Limit: "invalid"},
		{Cursor: "not-base64"},
		{CategorySlug: "contains/slash"},
		{Tag: "1234567890123456789012345678901"},
	} {
		if _, err := service.ListPosts(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("ListPosts(%+v) error = %v", input, err)
		}
	}
}

func TestPublicServiceOnlyMapsMissingPublishedDetail(t *testing.T) {
	store := &publicPostStoreStub{err: pgx.ErrNoRows}
	service := NewPublicService(store, publicCategoryStoreStub{})
	if _, err := service.GetPost(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetPost() error = %v", err)
	}
	if _, err := service.GetPost(context.Background(), "bad/slug"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("GetPost(invalid slug) error = %v", err)
	}
}

func TestPublicCursorRejectsUnknownFieldsAndVersion(t *testing.T) {
	for _, value := range []string{
		"eyJ2IjoyLCJwdWJsaXNoZWRfYXQiOiIyMDI2LTA4LTE4VDEyOjAwOjAwWiIsImlkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIn0",
		"eyJ2IjoxLCJwdWJsaXNoZWRfYXQiOiIyMDI2LTA4LTE4VDEyOjAwOjAwWiIsImlkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIiwiZXh0cmEiOnRydWV9",
	} {
		if _, err := decodePublicCursor(value); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("decodePublicCursor(%q) error = %v", value, err)
		}
	}
}
