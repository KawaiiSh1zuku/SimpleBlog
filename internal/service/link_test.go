package service

import (
	"context"
	"errors"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
)

type linkStoreStub struct{ created *model.FriendLink }

func (*linkStoreStub) List(context.Context) ([]model.FriendLink, error)              { return nil, nil }
func (*linkStoreStub) GetByID(context.Context, uuid.UUID) (*model.FriendLink, error) { return nil, nil }
func (s *linkStoreStub) Create(_ context.Context, link *model.FriendLink) (*model.FriendLink, error) {
	s.created = link
	return link, nil
}
func (*linkStoreStub) Update(context.Context, uuid.UUID, repository.UpdateLinkParams) (*model.FriendLink, error) {
	return nil, nil
}
func (*linkStoreStub) Delete(context.Context, uuid.UUID) error { return nil }

func TestLinkServiceRequiresAdminAndValidatesPlainHTTPLinks(t *testing.T) {
	store := &linkStoreStub{}
	links := NewLinkService(store)
	admin := &model.User{Role: model.RoleAdmin}
	input := CreateLinkInput{Name: " Example ", Description: " friend ", URL: "https://example.com/path?q=1#top", FaviconURL: "https://example.com/favicon.ico"}
	created, err := links.Create(context.Background(), admin, input)
	if err != nil || created.Name != "Example" || created.Description != "friend" {
		t.Fatalf("Create() = %+v, %v", created, err)
	}
	if _, err := links.Create(context.Background(), &model.User{Role: model.RoleContributor}, input); !errors.Is(err, ErrForbidden) {
		t.Fatalf("contributor Create() error = %v", err)
	}
	for _, invalid := range []CreateLinkInput{
		{Name: "<b>name</b>", URL: "https://example.com"},
		{Name: "name", Description: "<script>", URL: "https://example.com"},
		{Name: "name", URL: "javascript:alert(1)"},
		{Name: "name", URL: "https://user:pass@example.com"},
	} {
		if _, err := links.Create(context.Background(), admin, invalid); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Create(%+v) error = %v", invalid, err)
		}
	}
}
