package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type PostResponse struct {
	ID          uuid.UUID        `json:"id"`
	AuthorID    uuid.UUID        `json:"author_id"`
	CategoryID  uuid.UUID        `json:"category_id"`
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Summary     string           `json:"summary"`
	Content     string           `json:"content"`
	Tags        []string         `json:"tags"`
	Status      model.PostStatus `json:"status"`
	PublishedAt *time.Time       `json:"published_at"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type CreatePostRequest struct {
	CategoryID uuid.UUID `json:"category_id"`
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags"`
}

type UpdatePostRequest struct {
	CategoryID *uuid.UUID `json:"category_id"`
	Slug       *string    `json:"slug"`
	Title      *string    `json:"title"`
	Summary    *string    `json:"summary"`
	Content    *string    `json:"content"`
	Tags       *[]string  `json:"tags"`
}

type ManagedPostListItem struct {
	ID          uuid.UUID        `json:"id"`
	AuthorID    uuid.UUID        `json:"author_id"`
	CategoryID  uuid.UUID        `json:"category_id"`
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Summary     string           `json:"summary"`
	Tags        []string         `json:"tags"`
	Status      model.PostStatus `json:"status"`
	PublishedAt *time.Time       `json:"published_at"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type ManagedPostPageResponse struct {
	Items      []ManagedPostListItem `json:"items"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

func NewPostResponse(post *model.Post) PostResponse {
	tags := post.Tags
	if tags == nil {
		tags = []string{}
	}
	return PostResponse{
		ID: post.ID, AuthorID: post.AuthorID, CategoryID: post.CategoryID,
		Slug: post.Slug, Title: post.Title, Summary: post.Summary, Content: post.Content,
		Tags: tags, Status: post.Status, PublishedAt: post.PublishedAt,
		CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt,
	}
}

func NewPostResponses(posts []model.Post) []PostResponse {
	result := make([]PostResponse, 0, len(posts))
	for index := range posts {
		result = append(result, NewPostResponse(&posts[index]))
	}
	return result
}

func NewManagedPostPageResponse(posts []model.PostSummary, nextCursor string) ManagedPostPageResponse {
	items := make([]ManagedPostListItem, 0, len(posts))
	for index := range posts {
		post := &posts[index]
		tags := post.Tags
		if tags == nil {
			tags = []string{}
		}
		items = append(items, ManagedPostListItem{
			ID: post.ID, AuthorID: post.AuthorID, CategoryID: post.CategoryID,
			Slug: post.Slug, Title: post.Title, Summary: post.Summary, Tags: tags,
			Status: post.Status, PublishedAt: post.PublishedAt,
			CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt,
		})
	}
	return ManagedPostPageResponse{Items: items, NextCursor: nextCursor}
}
