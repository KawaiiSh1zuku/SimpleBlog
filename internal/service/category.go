package service

import (
	"context"
	"errors"
	"strings"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CategoryStore interface {
	List(context.Context) ([]model.Category, error)
	GetByID(context.Context, uuid.UUID) (*model.Category, error)
	Create(context.Context, *model.Category) (*model.Category, error)
	Update(context.Context, uuid.UUID, repository.UpdateCategoryParams) (*model.Category, error)
	Delete(context.Context, uuid.UUID) error
}

type CategoryService struct {
	Categories CategoryStore
}

type CreateCategoryInput struct {
	Name        string
	Slug        string
	Description string
}

type UpdateCategoryInput struct {
	Name        *string
	Slug        *string
	Description *string
}

func NewCategoryService(categories CategoryStore) *CategoryService {
	return &CategoryService{Categories: categories}
}

func (s *CategoryService) List(ctx context.Context, actor *model.User) ([]model.Category, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	return s.Categories.List(ctx)
}

func (s *CategoryService) Get(ctx context.Context, actor *model.User, id uuid.UUID) (*model.Category, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	category, err := s.Categories.GetByID(ctx, id)
	return category, mapCategoryStoreError(err)
}

func (s *CategoryService) Create(ctx context.Context, actor *model.User, input CreateCategoryInput) (*model.Category, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	name, slug, description, err := validateCategory(input.Name, input.Slug, input.Description)
	if err != nil {
		return nil, err
	}
	category := &model.Category{ID: uuid.New(), Name: name, Slug: slug, Description: description}
	created, err := s.Categories.Create(ctx, category)
	return created, mapCategoryStoreError(err)
}

func (s *CategoryService) Update(ctx context.Context, actor *model.User, id uuid.UUID, input UpdateCategoryInput) (*model.Category, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Slug == nil && input.Description == nil {
		return nil, ErrInvalidInput
	}
	params := repository.UpdateCategoryParams{}
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if !validText(value, 1, 80) {
			return nil, ErrInvalidInput
		}
		params.Name = &value
	}
	if input.Slug != nil {
		value, ok := normalizeSlug(*input.Slug, 100)
		if !ok {
			return nil, ErrInvalidInput
		}
		params.Slug = &value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		params.Description = &value
	}
	updated, err := s.Categories.Update(ctx, id, params)
	return updated, mapCategoryStoreError(err)
}

func (s *CategoryService) Delete(ctx context.Context, actor *model.User, id uuid.UUID) error {
	if !isAdmin(actor) {
		return ErrForbidden
	}
	return mapCategoryStoreError(s.Categories.Delete(ctx, id))
}

func validateCategory(name, slug, description string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	slug, ok := normalizeSlug(slug, 100)
	if !validText(name, 1, 80) || !ok {
		return "", "", "", ErrInvalidInput
	}
	return name, slug, strings.TrimSpace(description), nil
}

func mapCategoryStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case repository.IsUniqueViolation(err), repository.IsForeignKeyViolation(err):
		return ErrConflict
	default:
		return err
	}
}
