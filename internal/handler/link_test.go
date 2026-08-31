package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/google/uuid"
)

type emptyLinkStore struct{}

func (emptyLinkStore) List(context.Context) ([]model.FriendLink, error) {
	return []model.FriendLink{}, nil
}

func (emptyLinkStore) GetByID(context.Context, uuid.UUID) (*model.FriendLink, error) {
	return nil, nil
}

func (emptyLinkStore) Create(_ context.Context, link *model.FriendLink) (*model.FriendLink, error) {
	return link, nil
}

func (emptyLinkStore) Update(context.Context, uuid.UUID, repository.UpdateLinkParams) (*model.FriendLink, error) {
	return nil, nil
}

func (emptyLinkStore) Delete(context.Context, uuid.UUID) error { return nil }

func TestLinkListKeepsEmptyDataArray(t *testing.T) {
	handler := NewLinkHandler(service.NewLinkService(emptyLinkStore{}))
	recorder := httptest.NewRecorder()
	handler.ListPublic(recorder, httptest.NewRequest(http.MethodGet, "/links", nil))

	var body response.Response[json.RawMessage]
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != http.StatusOK || string(body.Data) != "[]" {
		t.Fatalf("response = %s", recorder.Body.String())
	}
}
