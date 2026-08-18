package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const MaxCommentContentRunes = 10000

type CommentStore interface {
	CreateForPublishedPost(context.Context, string, *model.Comment) (*model.Comment, error)
	ListPublicByPostSlug(context.Context, string) ([]model.Comment, bool, error)
	ListManaged(context.Context, repository.ListCommentsParams) ([]model.ManagedComment, error)
	GetManagedByID(context.Context, uuid.UUID) (*model.ManagedComment, error)
	Hide(context.Context, uuid.UUID) (*model.Comment, error)
	Show(context.Context, uuid.UUID) (*model.Comment, error)
	Delete(context.Context, uuid.UUID) (*model.Comment, error)
}

type CommentService struct {
	Comments CommentStore
}

type CreateCommentInput struct {
	ParentID *uuid.UUID
	Nickname string
	Email    string
	Content  string
}

type ListManagedCommentsInput struct {
	Limit  string
	Cursor string
}

type ManagedCommentPage struct {
	Items      []model.ManagedComment
	NextCursor string
}

func NewCommentService(comments CommentStore) *CommentService {
	return &CommentService{Comments: comments}
}

func (s *CommentService) Create(ctx context.Context, postSlug string, input CreateCommentInput) (*model.Comment, error) {
	postSlug, ok := normalizeSlug(postSlug, 160)
	if !ok {
		return nil, ErrInvalidInput
	}
	nickname, nicknameOK := normalizePlainText(input.Nickname, 1, 80, false)
	email := strings.TrimSpace(input.Email)
	content, contentOK := normalizePlainText(input.Content, 1, MaxCommentContentRunes, true)
	if !nicknameOK || !validEmail(email) || !contentOK {
		return nil, ErrInvalidInput
	}
	if input.ParentID != nil && *input.ParentID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	comment := &model.Comment{
		ID: uuid.New(), ParentID: input.ParentID, Nickname: nickname,
		Email: email, Content: content, Status: model.CommentStatusVisible,
	}
	created, err := s.Comments.CreateForPublishedPost(ctx, postSlug, comment)
	return created, mapCreateCommentError(err)
}

func (s *CommentService) ListPublic(ctx context.Context, postSlug string) ([]model.Comment, error) {
	postSlug, ok := normalizeSlug(postSlug, 160)
	if !ok {
		return nil, ErrInvalidInput
	}
	comments, exists, err := s.Comments.ListPublicByPostSlug(ctx, postSlug)
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	return comments, nil
}

func (s *CommentService) ListManaged(ctx context.Context, actor *model.User, input ListManagedCommentsInput) (*ManagedCommentPage, error) {
	if !canManagePosts(actor) {
		return nil, ErrForbidden
	}
	limit, cursor, err := parseManagedPage(input.Limit, input.Cursor, "comments")
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	var authorID *uuid.UUID
	if actor.Role == model.RoleContributor {
		authorID = &actor.ID
	}
	comments, err := s.Comments.ListManaged(ctx, repository.ListCommentsParams{AuthorID: authorID, Limit: limit + 1, Cursor: cursor})
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	page := &ManagedCommentPage{Items: comments}
	if len(comments) <= limit {
		return page, nil
	}
	page.Items = comments[:limit]
	last := page.Items[len(page.Items)-1]
	page.NextCursor, err = encodeManagedCursor("comments", repository.TimeCursor{At: last.CreatedAt, ID: last.ID})
	return page, err
}

func (s *CommentService) Hide(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Comment, error) {
	comment, err := s.authorizeManaged(ctx, actor, id)
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	if comment.Status != model.CommentStatusVisible {
		return nil, ErrConflict
	}
	updated, err := s.Comments.Hide(ctx, id)
	return updated, mapModerationError(err)
}

func (s *CommentService) Show(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Comment, error) {
	comment, err := s.authorizeManaged(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if comment.Status != model.CommentStatusHidden {
		return nil, ErrConflict
	}
	updated, err := s.Comments.Show(ctx, id)
	return updated, mapModerationError(err)
}

func (s *CommentService) Delete(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Comment, error) {
	comment, err := s.authorizeManaged(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if comment.Status == model.CommentStatusDeleted {
		return nil, ErrConflict
	}
	updated, err := s.Comments.Delete(ctx, id)
	return updated, mapModerationError(err)
}

func (s *CommentService) authorizeManaged(ctx context.Context, actor *model.User, id uuid.UUID) (*model.ManagedComment, error) {
	if !canManagePosts(actor) {
		return nil, ErrForbidden
	}
	comment, err := s.Comments.GetManagedByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, mapStoreAvailability(err)
	}
	if actor.Role == model.RoleContributor && comment.PostAuthorID != actor.ID {
		return nil, ErrForbidden
	}
	return comment, nil
}

func validEmail(value string) bool {
	if value == "" || utf8.RuneCountInString(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func mapCreateCommentError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case repository.IsForeignKeyViolation(err):
		return ErrInvalidInput
	case repository.IsInputViolation(err):
		return ErrInvalidInput
	case repository.IsRetryable(err):
		return ErrUnavailable
	default:
		return err
	}
}

func mapModerationError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConflict
	}
	if repository.IsRetryable(err) {
		return ErrUnavailable
	}
	return err
}
