package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxLinkDescriptionRunes = 500
	maxLinkURLRunes         = 2048
)

type LinkStore interface {
	List(context.Context) ([]model.FriendLink, error)
	GetByID(context.Context, uuid.UUID) (*model.FriendLink, error)
	Create(context.Context, *model.FriendLink) (*model.FriendLink, error)
	Update(context.Context, uuid.UUID, repository.UpdateLinkParams) (*model.FriendLink, error)
	Delete(context.Context, uuid.UUID) error
}

type LinkService struct{ Links LinkStore }

type CreateLinkInput struct{ Name, Description, URL, FaviconURL string }
type UpdateLinkInput struct{ Name, Description, URL, FaviconURL *string }

func NewLinkService(links LinkStore) *LinkService { return &LinkService{Links: links} }

func (s *LinkService) ListPublic(ctx context.Context) ([]model.FriendLink, error) {
	links, err := s.Links.List(ctx)
	return links, mapStoreAvailability(err)
}

func (s *LinkService) ListManaged(ctx context.Context, actor *model.User) ([]model.FriendLink, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	return s.ListPublic(ctx)
}

func (s *LinkService) Get(ctx context.Context, actor *model.User, id uuid.UUID) (*model.FriendLink, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	link, err := s.Links.GetByID(ctx, id)
	return link, mapLinkStoreError(err)
}

func (s *LinkService) Create(ctx context.Context, actor *model.User, input CreateLinkInput) (*model.FriendLink, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	name, description, targetURL, faviconURL, ok := validateLink(input.Name, input.Description, input.URL, input.FaviconURL)
	if !ok {
		return nil, ErrInvalidInput
	}
	link, err := s.Links.Create(ctx, &model.FriendLink{ID: uuid.New(), Name: name, Description: description, URL: targetURL, FaviconURL: faviconURL})
	return link, mapLinkStoreError(err)
}

func (s *LinkService) Update(ctx context.Context, actor *model.User, id uuid.UUID, input UpdateLinkInput) (*model.FriendLink, error) {
	if !isAdmin(actor) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Description == nil && input.URL == nil && input.FaviconURL == nil {
		return nil, ErrInvalidInput
	}
	params := repository.UpdateLinkParams{}
	if input.Name != nil {
		value, ok := normalizePlainText(*input.Name, 1, 80, false)
		if !ok {
			return nil, ErrInvalidInput
		}
		params.Name = &value
	}
	if input.Description != nil {
		value, ok := normalizePlainText(*input.Description, 0, maxLinkDescriptionRunes, true)
		if !ok {
			return nil, ErrInvalidInput
		}
		params.Description = &value
	}
	if input.URL != nil {
		value, ok := validExternalURL(*input.URL, false)
		if !ok {
			return nil, ErrInvalidInput
		}
		params.URL = &value
	}
	if input.FaviconURL != nil {
		value, ok := validExternalURL(*input.FaviconURL, true)
		if !ok {
			return nil, ErrInvalidInput
		}
		params.FaviconURL = &value
	}
	link, err := s.Links.Update(ctx, id, params)
	return link, mapLinkStoreError(err)
}

func (s *LinkService) Delete(ctx context.Context, actor *model.User, id uuid.UUID) error {
	if !isAdmin(actor) {
		return ErrForbidden
	}
	return mapLinkStoreError(s.Links.Delete(ctx, id))
}

func validateLink(name, description, targetURL, faviconURL string) (string, string, string, string, bool) {
	name, nameOK := normalizePlainText(name, 1, 80, false)
	description, descriptionOK := normalizePlainText(description, 0, maxLinkDescriptionRunes, true)
	targetURL, targetOK := validExternalURL(targetURL, false)
	faviconURL, faviconOK := validExternalURL(faviconURL, true)
	return name, description, targetURL, faviconURL, nameOK && descriptionOK && targetOK && faviconOK
}

func validExternalURL(value string, allowEmpty bool) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return value, allowEmpty
	}
	if !validText(value, 1, maxLinkURLRunes) {
		return "", false
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return value, true
}

func mapLinkStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		return ErrNotFound
	case repository.IsInputViolation(err):
		return ErrInvalidInput
	case repository.IsRetryable(err):
		return ErrUnavailable
	default:
		return err
	}
}
