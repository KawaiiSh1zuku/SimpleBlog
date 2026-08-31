package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultPublicPageSize = 20
	maxPublicPageSize     = 100
	publicCursorVersion   = 1
)

type PublicPostStore interface {
	ListPublished(context.Context, repository.ListPublishedParams) ([]model.PublicPost, error)
	SearchPublished(context.Context, repository.SearchPublishedParams) ([]model.PublicPost, error)
	GetPublishedBySlug(context.Context, string) (*model.PublicPost, error)
	ListPublishedTags(context.Context) ([]string, error)
}

type PublicCategoryStore interface {
	List(context.Context) ([]model.Category, error)
}

type PublicService struct {
	Posts      PublicPostStore
	Categories PublicCategoryStore
}

type ListPublicPostsInput struct {
	Limit        string
	Cursor       string
	CategorySlug string
	Tag          string
}

type SearchPublicPostsInput struct {
	Query  string
	Limit  string
	Cursor string
}

type PublicPostPage struct {
	Items      []model.PublicPost
	NextCursor string
}

type publicCursorPayload struct {
	Version     int    `json:"v"`
	PublishedAt string `json:"published_at"`
	ID          string `json:"id"`
}

func NewPublicService(posts PublicPostStore, categories PublicCategoryStore) *PublicService {
	return &PublicService{Posts: posts, Categories: categories}
}

func (s *PublicService) ListPosts(ctx context.Context, input ListPublicPostsInput) (*PublicPostPage, error) {
	limit, err := parsePublicLimit(input.Limit)
	if err != nil {
		return nil, err
	}
	categorySlug := ""
	if strings.TrimSpace(input.CategorySlug) != "" {
		var ok bool
		categorySlug, ok = normalizeSlug(input.CategorySlug, 100)
		if !ok {
			return nil, ErrInvalidInput
		}
	}
	tag := strings.ToLower(strings.TrimSpace(input.Tag))
	if utf8.RuneCountInString(tag) > 30 {
		return nil, ErrInvalidInput
	}
	cursor, err := decodePublicCursor(input.Cursor)
	if err != nil {
		return nil, err
	}

	posts, err := s.Posts.ListPublished(ctx, repository.ListPublishedParams{
		Limit: limit + 1, CategorySlug: categorySlug, Tag: tag, Cursor: cursor,
	})
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	return buildPublicPostPage(posts, limit)
}

func (s *PublicService) SearchPosts(ctx context.Context, input SearchPublicPostsInput) (*PublicPostPage, error) {
	query, ok := normalizePlainText(input.Query, 1, 100, false)
	if !ok {
		return nil, ErrInvalidInput
	}
	limit, err := parsePublicLimit(input.Limit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodePublicCursor(input.Cursor)
	if err != nil {
		return nil, err
	}
	posts, err := s.Posts.SearchPublished(ctx, repository.SearchPublishedParams{
		Query: query, Limit: limit + 1, Cursor: cursor,
	})
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	return buildPublicPostPage(posts, limit)
}

func buildPublicPostPage(posts []model.PublicPost, limit int) (*PublicPostPage, error) {
	page := &PublicPostPage{Items: posts}
	if len(posts) <= limit {
		return page, nil
	}
	page.Items = posts[:limit]
	last := page.Items[len(page.Items)-1]
	if last.PublishedAt == nil {
		return nil, errors.New("published post has no published_at")
	}
	nextCursor, err := encodePublicCursor(repository.PublishedCursor{
		PublishedAt: *last.PublishedAt,
		ID:          last.ID,
	})
	if err != nil {
		return nil, err
	}
	page.NextCursor = nextCursor
	return page, nil
}

func (s *PublicService) GetPost(ctx context.Context, slug string) (*model.PublicPost, error) {
	slug, ok := normalizeSlug(slug, 160)
	if !ok {
		return nil, ErrInvalidInput
	}
	post, err := s.Posts.GetPublishedBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return post, mapStoreAvailability(err)
}

func (s *PublicService) ListCategories(ctx context.Context) ([]model.Category, error) {
	categories, err := s.Categories.List(ctx)
	return categories, mapStoreAvailability(err)
}

func (s *PublicService) ListTags(ctx context.Context) ([]string, error) {
	tags, err := s.Posts.ListPublishedTags(ctx)
	return tags, mapStoreAvailability(err)
}

func parsePublicLimit(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultPublicPageSize, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxPublicPageSize {
		return 0, ErrInvalidInput
	}
	return limit, nil
}

func encodePublicCursor(cursor repository.PublishedCursor) (string, error) {
	payload, err := json.Marshal(publicCursorPayload{
		Version: publicCursorVersion, PublishedAt: cursor.PublishedAt.UTC().Format(time.RFC3339Nano), ID: cursor.ID.String(),
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodePublicCursor(value string) (*repository.PublishedCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if len(value) > 512 {
		return nil, ErrInvalidInput
	}
	encoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var payload publicCursorPayload
	if err := decoder.Decode(&payload); err != nil {
		return nil, ErrInvalidInput
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrInvalidInput
	}
	if payload.Version != publicCursorVersion {
		return nil, ErrInvalidInput
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, payload.PublishedAt)
	if err != nil {
		return nil, ErrInvalidInput
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return nil, ErrInvalidInput
	}
	return &repository.PublishedCursor{PublishedAt: publishedAt, ID: id}, nil
}
