package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type LinkResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	FaviconURL  string    `json:"favicon_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateLinkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	FaviconURL  string `json:"favicon_url"`
}

type UpdateLinkRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	URL         *string `json:"url"`
	FaviconURL  *string `json:"favicon_url"`
}

func NewLinkResponse(link *model.FriendLink) LinkResponse {
	return LinkResponse{ID: link.ID, Name: safePlainText(link.Name), Description: safePlainText(link.Description), URL: safeHTTPURL(link.URL), FaviconURL: safeHTTPURL(link.FaviconURL), CreatedAt: link.CreatedAt, UpdatedAt: link.UpdatedAt}
}

func NewLinkResponses(links []model.FriendLink) []LinkResponse {
	result := make([]LinkResponse, 0, len(links))
	for index := range links {
		result = append(result, NewLinkResponse(&links[index]))
	}
	return result
}
