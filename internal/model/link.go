package model

import (
	"time"

	"github.com/google/uuid"
)

type FriendLink struct {
	ID          uuid.UUID
	Name        string
	Description string
	URL         string
	FaviconURL  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
