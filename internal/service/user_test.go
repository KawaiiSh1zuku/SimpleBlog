package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type userStoreStub struct {
	created        *model.User
	softDeleteErr  error
	softDeleteCall bool
	passwordHash   string
}

func TestUserServiceRejectsOversizedAvatarURL(t *testing.T) {
	service := NewUserService(&userStoreStub{})
	_, err := service.Create(context.Background(), &model.User{Role: model.RoleAdmin}, CreateUserInput{
		Username: "writer", Password: "strong-password", Nickname: "Writer",
		AvatarURL: strings.Repeat("x", maxAvatarURLRunes+1), Role: model.RoleContributor,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("oversized avatar URL error = %v", err)
	}
}

func (stub *userStoreStub) GetByID(context.Context, uuid.UUID) (*model.User, error) { return nil, nil }
func (stub *userStoreStub) List(context.Context) ([]model.User, error)              { return nil, nil }
func (stub *userStoreStub) Create(_ context.Context, user *model.User) (*model.User, error) {
	stub.created = user
	return user, nil
}
func (stub *userStoreStub) CreateBootstrapAdmin(_ context.Context, user *model.User) (*model.User, error) {
	stub.created = user
	return user, nil
}
func (stub *userStoreStub) Update(context.Context, uuid.UUID, repository.UpdateUserParams) (*model.User, error) {
	return nil, nil
}
func (stub *userStoreStub) SoftDelete(context.Context, uuid.UUID) error {
	stub.softDeleteCall = true
	return stub.softDeleteErr
}
func (stub *userStoreStub) UpdatePassword(_ context.Context, _ uuid.UUID, hash string) error {
	stub.passwordHash = hash
	return nil
}

func TestUserServiceCreateValidatesAndHashesPassword(t *testing.T) {
	store := &userStoreStub{}
	service := NewUserService(store)
	actor := &model.User{ID: uuid.New(), Role: model.RoleAdmin}
	created, err := service.Create(context.Background(), actor, CreateUserInput{
		Username: " contributor ", Password: "strong-password", Nickname: " Writer ", Role: model.RoleContributor,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Username != "contributor" || created.Nickname != "Writer" {
		t.Fatalf("values were not normalized: %+v", created)
	}
	if created.PasswordHash == "strong-password" {
		t.Fatal("password was stored as plain text")
	}
	if bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("strong-password")) != nil {
		t.Fatal("stored password hash does not match")
	}
	if _, err := service.Create(context.Background(), actor, CreateUserInput{Username: "ab"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("invalid Create() error = %v", err)
	}
}

func TestUserServiceProtectsCurrentAndLastAdmin(t *testing.T) {
	actor := &model.User{ID: uuid.New(), Role: model.RoleAdmin}
	store := &userStoreStub{}
	service := NewUserService(store)
	if err := service.Delete(context.Background(), actor, actor.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("self Delete() error = %v", err)
	}
	if store.softDeleteCall {
		t.Fatal("self deletion reached the repository")
	}

	store.softDeleteErr = repository.ErrLastAdmin
	if err := service.Delete(context.Background(), actor, uuid.New()); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("last-admin Delete() error = %v", err)
	}
}

func TestUserServiceUpdatePasswordHashesPassword(t *testing.T) {
	store := &userStoreStub{}
	service := NewUserService(store)
	actor := &model.User{Role: model.RoleAdmin}
	if err := service.UpdatePassword(context.Background(), actor, uuid.New(), "new-password"); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(store.passwordHash), []byte("new-password")) != nil {
		t.Fatal("password was not hashed")
	}
}
