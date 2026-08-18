package model

import (
	"time"

	"github.com/google/uuid"
)

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
)

func (status PostStatus) Valid() bool {
	return status == PostStatusDraft || status == PostStatusPublished
}

type Post struct {
	ID          uuid.UUID
	AuthorID    uuid.UUID
	CategoryID  uuid.UUID
	Slug        string
	Title       string
	Summary     string
	Content     string
	Tags        []string
	Status      PostStatus
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
