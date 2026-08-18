package service

import (
	"context"
	"errors"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type commentStoreStub struct {
	created      *model.Comment
	postSlug     string
	createErr    error
	publicExists bool
	managed      *model.ManagedComment
	listAuthor   *uuid.UUID
	hideCalled   bool
}

func (stub *commentStoreStub) CreateForPublishedPost(_ context.Context, slug string, comment *model.Comment) (*model.Comment, error) {
	stub.postSlug = slug
	stub.created = comment
	return comment, stub.createErr
}
func (stub *commentStoreStub) ListPublicByPostSlug(context.Context, string) ([]model.Comment, bool, error) {
	return []model.Comment{}, stub.publicExists, nil
}
func (stub *commentStoreStub) ListManaged(_ context.Context, authorID *uuid.UUID) ([]model.ManagedComment, error) {
	stub.listAuthor = authorID
	return nil, nil
}
func (stub *commentStoreStub) GetManagedByID(context.Context, uuid.UUID) (*model.ManagedComment, error) {
	return stub.managed, nil
}
func (stub *commentStoreStub) Hide(context.Context, uuid.UUID) (*model.Comment, error) {
	stub.hideCalled = true
	return &model.Comment{Status: model.CommentStatusHidden}, nil
}
func (stub *commentStoreStub) Show(context.Context, uuid.UUID) (*model.Comment, error) {
	return &model.Comment{Status: model.CommentStatusVisible}, nil
}
func (stub *commentStoreStub) Delete(context.Context, uuid.UUID) (*model.Comment, error) {
	return &model.Comment{Status: model.CommentStatusDeleted}, nil
}

func TestCommentServiceCreatesValidatedAnonymousComment(t *testing.T) {
	parentID := uuid.New()
	store := &commentStoreStub{}
	service := NewCommentService(store)
	created, err := service.Create(context.Background(), " Post-Slug ", CreateCommentInput{
		ParentID: &parentID, Nickname: " Visitor ", Email: "visitor@example.com", Content: "reply",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.postSlug != "post-slug" || created.Nickname != "Visitor" || created.Email != "visitor@example.com" {
		t.Fatalf("Create() result = %+v, slug = %q", created, store.postSlug)
	}
	if created.ParentID == nil || *created.ParentID != parentID || created.Status != model.CommentStatusVisible {
		t.Fatalf("Create() relation/status = %+v", created)
	}
	for _, input := range []CreateCommentInput{
		{Nickname: "", Email: "visitor@example.com", Content: "body"},
		{Nickname: "Visitor", Email: "Name <visitor@example.com>", Content: "body"},
		{Nickname: "Visitor", Email: "visitor@example.com", Content: "  "},
	} {
		if _, err := service.Create(context.Background(), "post", input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Create(%+v) error = %v", input, err)
		}
	}
}

func TestCommentServiceMapsPostAndParentFailures(t *testing.T) {
	store := &commentStoreStub{createErr: pgx.ErrNoRows}
	service := NewCommentService(store)
	input := CreateCommentInput{Nickname: "Visitor", Email: "visitor@example.com", Content: "body"}
	if _, err := service.Create(context.Background(), "post", input); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing post Create() error = %v", err)
	}
	store.createErr = &pgconn.PgError{Code: "23503"}
	if _, err := service.Create(context.Background(), "post", input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid parent Create() error = %v", err)
	}
}

func TestCommentServiceEnforcesModerationOwnershipAndState(t *testing.T) {
	ownerID := uuid.New()
	store := &commentStoreStub{managed: &model.ManagedComment{
		Comment: model.Comment{Status: model.CommentStatusVisible}, PostAuthorID: ownerID,
	}}
	service := NewCommentService(store)
	other := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	if _, err := service.Hide(context.Background(), other, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other contributor Hide() error = %v", err)
	}
	if store.hideCalled {
		t.Fatal("unauthorized Hide() reached repository")
	}
	owner := &model.User{ID: ownerID, Role: model.RoleContributor}
	if _, err := service.Hide(context.Background(), owner, uuid.New()); err != nil {
		t.Fatalf("owner Hide() error = %v", err)
	}
	store.managed.Status = model.CommentStatusDeleted
	if _, err := service.Show(context.Background(), &model.User{Role: model.RoleAdmin}, uuid.New()); !errors.Is(err, ErrConflict) {
		t.Fatalf("Show(deleted) error = %v", err)
	}
	if _, err := service.ListManaged(context.Background(), owner); err != nil {
		t.Fatalf("ListManaged() error = %v", err)
	}
	if store.listAuthor == nil || *store.listAuthor != ownerID {
		t.Fatalf("contributor list author = %v", store.listAuthor)
	}
}

func TestCommentServiceReturnsNotFoundForUnavailablePublicPost(t *testing.T) {
	service := NewCommentService(&commentStoreStub{publicExists: false})
	if _, err := service.ListPublic(context.Background(), "post"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListPublic() error = %v", err)
	}
}
