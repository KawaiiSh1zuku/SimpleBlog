package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

func NewCategoryResponse(category *model.Category) CategoryResponse {
	return CategoryResponse{
		ID: category.ID, Name: category.Name, Slug: category.Slug,
		Description: category.Description, CreatedAt: category.CreatedAt, UpdatedAt: category.UpdatedAt,
	}
}

func NewCategoryResponses(categories []model.Category) []CategoryResponse {
	result := make([]CategoryResponse, 0, len(categories))
	for index := range categories {
		result = append(result, NewCategoryResponse(&categories[index]))
	}
	return result
}
