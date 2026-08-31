package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxTags             = 10
	MaxPostContentBytes = 1024 * 1024
	maxPostSummaryRunes = 1000
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type PostStore interface {
	List(context.Context, repository.ListPostsParams) ([]model.PostSummary, error)
	GetByID(context.Context, uuid.UUID) (*model.Post, error)
	Create(context.Context, *model.Post) (*model.Post, error)
	Update(context.Context, uuid.UUID, repository.UpdatePostParams) (*model.Post, error)
	Delete(context.Context, uuid.UUID) error
	Publish(context.Context, uuid.UUID) (*model.Post, error)
	Withdraw(context.Context, uuid.UUID) (*model.Post, error)
}

type PostService struct {
	Posts PostStore
}

type ListManagedPostsInput struct {
	Limit  string
	Cursor string
}

type ManagedPostPage struct {
	Items      []model.PostSummary
	NextCursor string
}

type CreatePostInput struct {
	CategoryID uuid.UUID
	Slug       string
	Title      string
	Summary    string
	CoverImg   string
	Content    string
	Tags       []string
}

type UpdatePostInput struct {
	CategoryID *uuid.UUID
	Slug       *string
	Title      *string
	Summary    *string
	CoverImg   *string
	Content    *string
	Tags       *[]string
}

func NewPostService(posts PostStore) *PostService { return &PostService{Posts: posts} }

func (s *PostService) List(ctx context.Context, actor *model.User, input ListManagedPostsInput) (*ManagedPostPage, error) {
	if !canManagePosts(actor) {
		return nil, ErrForbidden
	}
	limit, cursor, err := parseManagedPage(input.Limit, input.Cursor, "posts")
	if err != nil {
		return nil, err
	}
	var authorID *uuid.UUID
	if actor.Role == model.RoleContributor {
		authorID = &actor.ID
	}
	posts, err := s.Posts.List(ctx, repository.ListPostsParams{AuthorID: authorID, Limit: limit + 1, Cursor: cursor})
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	page := &ManagedPostPage{Items: posts}
	if len(posts) <= limit {
		return page, nil
	}
	page.Items = posts[:limit]
	last := page.Items[len(page.Items)-1]
	page.NextCursor, err = encodeManagedCursor("posts", repository.TimeCursor{At: last.UpdatedAt, ID: last.ID})
	return page, err
}

func (s *PostService) Get(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Post, error) {
	post, err := s.authorizePost(ctx, actor, id)
	return post, err
}

func (s *PostService) Create(ctx context.Context, actor *model.User, input CreatePostInput) (*model.Post, error) {
	if !canManagePosts(actor) || actor.ID == uuid.Nil {
		return nil, ErrForbidden
	}
	if input.CategoryID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	slug, title, summary, coverImg, tags, err := validatePost(
		input.Slug, input.Title, input.Summary, input.CoverImg, input.Content, input.Tags,
	)
	if err != nil {
		return nil, err
	}
	if coverImg == nil {
		coverImg = firstPostImageURL(input.Content)
	}
	post := &model.Post{
		ID: uuid.New(), AuthorID: actor.ID, CategoryID: input.CategoryID,
		Slug: slug, Title: title, Summary: summary, CoverImg: coverImg, Content: input.Content,
		Tags: tags, Status: model.PostStatusDraft,
	}
	created, err := s.Posts.Create(ctx, post)
	return created, mapPostStoreError(err)
}

func (s *PostService) Update(ctx context.Context, actor *model.User, id uuid.UUID, input UpdatePostInput) (*model.Post, error) {
	current, err := s.authorizePost(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	shouldDeriveCover := current.CoverImg == nil
	if input.CoverImg != nil {
		shouldDeriveCover = strings.TrimSpace(*input.CoverImg) == ""
	}
	if shouldDeriveCover {
		content := current.Content
		if input.Content != nil {
			content = *input.Content
		}
		if coverImg := firstPostImageURL(content); coverImg != nil {
			input.CoverImg = coverImg
		}
	}
	params, err := validatePostUpdate(input)
	if err != nil {
		return nil, err
	}
	updated, err := s.Posts.Update(ctx, id, params)
	return updated, mapPostStoreError(err)
}

func (s *PostService) Delete(ctx context.Context, actor *model.User, id uuid.UUID) error {
	if _, err := s.authorizePost(ctx, actor, id); err != nil {
		return err
	}
	return mapPostStoreError(s.Posts.Delete(ctx, id))
}

func (s *PostService) Publish(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Post, error) {
	post, err := s.authorizePost(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if post.Status != model.PostStatusDraft {
		return nil, ErrConflict
	}
	published, err := s.Posts.Publish(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConflict
	}
	return published, mapPostStoreError(err)
}

func (s *PostService) Withdraw(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Post, error) {
	post, err := s.authorizePost(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if post.Status != model.PostStatusPublished {
		return nil, ErrConflict
	}
	withdrawn, err := s.Posts.Withdraw(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConflict
	}
	return withdrawn, mapPostStoreError(err)
}

func (s *PostService) authorizePost(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Post, error) {
	if !canManagePosts(actor) {
		return nil, ErrForbidden
	}
	post, err := s.Posts.GetByID(ctx, id)
	if err != nil {
		return nil, mapPostStoreError(err)
	}
	if actor.Role == model.RoleContributor && post.AuthorID != actor.ID {
		return nil, ErrForbidden
	}
	return post, nil
}

func validatePost(slug, title, summary, coverImg, content string, tags []string) (string, string, string, *string, []string, error) {
	slug, ok := normalizeSlug(slug, 160)
	title = strings.TrimSpace(title)
	if !ok || !validText(title, 1, 200) || !validText(summary, 0, maxPostSummaryRunes) ||
		strings.TrimSpace(content) == "" || len(content) > MaxPostContentBytes {
		return "", "", "", nil, nil, ErrInvalidInput
	}
	coverImg, ok = validExternalURL(coverImg, true)
	if !ok {
		return "", "", "", nil, nil, ErrInvalidInput
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return "", "", "", nil, nil, err
	}
	var normalizedCoverImg *string
	if coverImg != "" {
		normalizedCoverImg = &coverImg
	}
	return slug, title, strings.TrimSpace(summary), normalizedCoverImg, normalizedTags, nil
}

func validatePostUpdate(input UpdatePostInput) (repository.UpdatePostParams, error) {
	params := repository.UpdatePostParams{CategoryID: input.CategoryID, Summary: input.Summary}
	if input.CategoryID == nil && input.Slug == nil && input.Title == nil && input.Summary == nil && input.CoverImg == nil && input.Content == nil && input.Tags == nil {
		return params, ErrInvalidInput
	}
	if input.CategoryID != nil && *input.CategoryID == uuid.Nil {
		return params, ErrInvalidInput
	}
	if input.Slug != nil {
		value, ok := normalizeSlug(*input.Slug, 160)
		if !ok {
			return params, ErrInvalidInput
		}
		params.Slug = &value
	}
	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		if !validText(value, 1, 200) {
			return params, ErrInvalidInput
		}
		params.Title = &value
	}
	if input.Summary != nil {
		value := strings.TrimSpace(*input.Summary)
		if !validText(value, 0, maxPostSummaryRunes) {
			return params, ErrInvalidInput
		}
		params.Summary = &value
	}
	if input.CoverImg != nil {
		value, ok := validExternalURL(*input.CoverImg, true)
		if !ok {
			return params, ErrInvalidInput
		}
		params.CoverImg = &value
	}
	if input.Content != nil {
		if strings.TrimSpace(*input.Content) == "" || len(*input.Content) > MaxPostContentBytes {
			return params, ErrInvalidInput
		}
		params.Content = input.Content
	}
	if input.Tags != nil {
		values, err := normalizeTags(*input.Tags)
		if err != nil {
			return params, err
		}
		params.Tags = &values
	}
	return params, nil
}

func normalizeTags(tags []string) ([]string, error) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 30 {
			return nil, ErrInvalidInput
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	if len(result) > maxTags {
		return nil, ErrInvalidInput
	}
	return result, nil
}

func normalizeSlug(slug string, maximum int) (string, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	return slug, utf8.RuneCountInString(slug) <= maximum && slugPattern.MatchString(slug)
}

func canManagePosts(user *model.User) bool {
	return user != nil && (user.Role == model.RoleAdmin || user.Role == model.RoleContributor)
}

func mapPostStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case repository.IsUniqueViolation(err):
		return ErrConflict
	case repository.IsForeignKeyViolation(err):
		return ErrNotFound
	case repository.IsInputViolation(err):
		return ErrInvalidInput
	case repository.IsRetryable(err):
		return ErrUnavailable
	default:
		return err
	}
}
