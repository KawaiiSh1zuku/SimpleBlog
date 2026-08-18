package model

import (
	"time"

	"github.com/google/uuid"
)

type FeedArticle struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Summary     string
	PublishedAt time.Time
}

type FeedComment struct {
	ID        uuid.UUID
	PostSlug  string
	PostTitle string
	Nickname  string
	Content   string
	CreatedAt time.Time
}
