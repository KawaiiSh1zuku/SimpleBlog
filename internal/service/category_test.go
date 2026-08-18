package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type categoryStoreStub struct {
	created   *model.Category
	updated   repository.UpdateCategoryParams
	deleteErr error
}

func TestCategoryServiceRejectsOversizedDescription(t *testing.T) {
	service := NewCategoryService(&categoryStoreStub{})
	_, err := service.Create(context.Background(), &model.User{Role: model.RoleAdmin}, CreateCategoryInput{
		Name: "General", Slug: "general", Description: strings.Repeat("类", maxCategoryDescriptionRunes+1),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("oversized description error = %v", err)
	}
}

func (stub *categoryStoreStub) List(context.Context) ([]model.Category, error) { return nil, nil }
func (stub *categoryStoreStub) GetByID(context.Context, uuid.UUID) (*model.Category, error) {
	return &model.Category{}, nil
}
func (stub *categoryStoreStub) Create(_ context.Context, category *model.Category) (*model.Category, error) {
	stub.created = category
	return category, nil
}
func (stub *categoryStoreStub) Update(_ context.Context, _ uuid.UUID, params repository.UpdateCategoryParams) (*model.Category, error) {
	stub.updated = params
	return &model.Category{}, nil
}
func (stub *categoryStoreStub) Delete(context.Context, uuid.UUID) error { return stub.deleteErr }

func TestCategoryServiceRequiresAdminAndNormalizesInput(t *testing.T) {
	store := &categoryStoreStub{}
	service := NewCategoryService(store)
	contributor := &model.User{Role: model.RoleContributor}
	if _, err := service.Create(context.Background(), contributor, CreateCategoryInput{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("contributor Create() error = %v", err)
	}
	admin := &model.User{Role: model.RoleAdmin}
	created, err := service.Create(context.Background(), admin, CreateCategoryInput{
		Name: " General ", Slug: "General-NEWS", Description: " News ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "General" || created.Slug != "general-news" || created.Description != "News" {
		t.Fatalf("Create() result = %+v", created)
	}
	if _, err := service.Update(context.Background(), admin, uuid.New(), UpdateCategoryInput{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty Update() error = %v", err)
	}
}

func TestCategoryServiceMapsReferencedDeleteToConflict(t *testing.T) {
	store := &categoryStoreStub{deleteErr: &pgconn.PgError{Code: "23503"}}
	service := NewCategoryService(store)
	err := service.Delete(context.Background(), &model.User{Role: model.RoleAdmin}, uuid.New())
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Delete() error = %v", err)
	}
}
