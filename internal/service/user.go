package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	GetByID(context.Context, uuid.UUID) (*model.User, error)
	List(context.Context) ([]model.User, error)
	Create(context.Context, *model.User) (*model.User, error)
	CreateBootstrapAdmin(context.Context, *model.User) (*model.User, error)
	Update(context.Context, uuid.UUID, repository.UpdateUserParams) (*model.User, error)
	SoftDelete(context.Context, uuid.UUID) error
	UpdatePassword(context.Context, uuid.UUID, string) error
}

type UserService struct {
	Users UserStore
}

type CreateUserInput struct {
	Username  string
	Password  string
	Nickname  string
	AvatarURL string
	Role      model.Role
}

type UpdateUserInput struct {
	Username  *string
	Nickname  *string
	AvatarURL *string
	Role      *model.Role
}

func NewUserService(users UserStore) *UserService { return &UserService{Users: users} }

func (s *UserService) List(ctx context.Context, actor *model.User) ([]model.User, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	return s.Users.List(ctx)
}

func (s *UserService) Get(ctx context.Context, actor *model.User, id uuid.UUID) (*model.User, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	user, err := s.Users.GetByID(ctx, id)
	return user, mapUserStoreError(err)
}

func (s *UserService) Create(ctx context.Context, actor *model.User, input CreateUserInput) (*model.User, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	user, err := buildUser(input)
	if err != nil {
		return nil, err
	}
	created, err := s.Users.Create(ctx, user)
	return created, mapUserStoreError(err)
}

func (s *UserService) BootstrapAdmin(ctx context.Context, input CreateUserInput) (*model.User, error) {
	input.Role = model.RoleAdmin
	user, err := buildUser(input)
	if err != nil {
		return nil, err
	}
	created, err := s.Users.CreateBootstrapAdmin(ctx, user)
	return created, mapUserStoreError(err)
}

func (s *UserService) Update(ctx context.Context, actor *model.User, id uuid.UUID, input UpdateUserInput) (*model.User, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	params, err := validateUpdate(input)
	if err != nil {
		return nil, err
	}
	updated, err := s.Users.Update(ctx, id, params)
	return updated, mapUserStoreError(err)
}

func (s *UserService) Delete(ctx context.Context, actor *model.User, id uuid.UUID) error {
	if !isAdmin(actor) {
		return ErrForbidden
	}
	if actor.ID == id {
		return ErrForbidden
	}
	return mapUserStoreError(s.Users.SoftDelete(ctx, id))
}

func (s *UserService) UpdatePassword(ctx context.Context, actor *model.User, id uuid.UUID, password string) error {
	if !isAdmin(actor) {
		return ErrForbidden
	}
	if !validPassword(password) {
		return ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return mapUserStoreError(s.Users.UpdatePassword(ctx, id, string(hash)))
}

func buildUser(input CreateUserInput) (*model.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Nickname = strings.TrimSpace(input.Nickname)
	input.AvatarURL = strings.TrimSpace(input.AvatarURL)
	if !validText(input.Username, 3, 64) || !validText(input.Nickname, 1, 80) ||
		!validPassword(input.Password) || !input.Role.Valid() {
		return nil, ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID: uuid.New(), Username: input.Username, PasswordHash: string(hash),
		Nickname: input.Nickname, AvatarURL: input.AvatarURL, Role: input.Role,
	}, nil
}

func validateUpdate(input UpdateUserInput) (repository.UpdateUserParams, error) {
	params := repository.UpdateUserParams{AvatarURL: input.AvatarURL, Role: input.Role}
	if input.Username == nil && input.Nickname == nil && input.AvatarURL == nil && input.Role == nil {
		return params, ErrInvalidInput
	}
	if input.Username != nil {
		value := strings.TrimSpace(*input.Username)
		if !validText(value, 3, 64) {
			return params, ErrInvalidInput
		}
		params.Username = &value
	}
	if input.Nickname != nil {
		value := strings.TrimSpace(*input.Nickname)
		if !validText(value, 1, 80) {
			return params, ErrInvalidInput
		}
		params.Nickname = &value
	}
	if input.AvatarURL != nil {
		value := strings.TrimSpace(*input.AvatarURL)
		params.AvatarURL = &value
	}
	if input.Role != nil && !input.Role.Valid() {
		return params, ErrInvalidInput
	}
	return params, nil
}

func validText(value string, minimum, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func validPassword(password string) bool { return len(password) >= 10 && len(password) <= 72 }

func isAdmin(user *model.User) bool { return user != nil && user.Role == model.RoleAdmin }

func mapUserStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case errors.Is(err, repository.ErrLastAdmin):
		return ErrLastAdmin
	case errors.Is(err, repository.ErrAdminExists), repository.IsUniqueViolation(err):
		return ErrConflict
	default:
		return err
	}
}
