package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type PublicAuthorResponse struct {
	ID        uuid.UUID `json:"id"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
}

type PublicCategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
}

type PublicCategorySummaryResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type PublicPostListItem struct {
	ID          uuid.UUID                     `json:"id"`
	Slug        string                        `json:"slug"`
	Title       string                        `json:"title"`
	Summary     string                        `json:"summary"`
	CoverImg    *string                       `json:"cover_img"`
	Tags        []string                      `json:"tags"`
	PublishedAt *time.Time                    `json:"published_at"`
	Author      PublicAuthorResponse          `json:"author"`
	Category    PublicCategorySummaryResponse `json:"category"`
}

type PublicPostDetail struct {
	PublicPostListItem
	Content string `json:"content"`
}

type PublicPostPageResponse struct {
	Items      []PublicPostListItem `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

func NewPublicPostListItem(post *model.PublicPost) PublicPostListItem {
	tags := post.Tags
	if tags == nil {
		tags = []string{}
	}
	return PublicPostListItem{
		ID: post.ID, Slug: post.Slug, Title: post.Title, Summary: post.Summary,
		CoverImg: post.CoverImg, Tags: tags, PublishedAt: post.PublishedAt,
		Author: PublicAuthorResponse{
			ID: post.AuthorID, Nickname: post.AuthorNickname, AvatarURL: post.AuthorAvatarURL,
		},
		Category: PublicCategorySummaryResponse{
			ID: post.CategoryID, Name: post.CategoryName, Slug: post.CategorySlug,
		},
	}
}

func NewPublicPostPageResponse(posts []model.PublicPost, nextCursor string) PublicPostPageResponse {
	items := make([]PublicPostListItem, 0, len(posts))
	for index := range posts {
		items = append(items, NewPublicPostListItem(&posts[index]))
	}
	return PublicPostPageResponse{Items: items, NextCursor: nextCursor}
}

func NewPublicPostDetail(post *model.PublicPost) PublicPostDetail {
	return PublicPostDetail{PublicPostListItem: NewPublicPostListItem(post), Content: post.Content}
}

func NewPublicCategoryResponses(categories []model.Category) []PublicCategoryResponse {
	result := make([]PublicCategoryResponse, 0, len(categories))
	for index := range categories {
		category := &categories[index]
		result = append(result, PublicCategoryResponse{
			ID: category.ID, Name: category.Name, Slug: category.Slug, Description: category.Description,
		})
	}
	return result
}
