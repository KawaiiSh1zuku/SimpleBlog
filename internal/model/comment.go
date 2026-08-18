package model

import (
	"time"

	"github.com/google/uuid"
)

type CommentStatus string

const (
	CommentStatusVisible CommentStatus = "visible"
	CommentStatusHidden  CommentStatus = "hidden"
	CommentStatusDeleted CommentStatus = "deleted"
)

type Comment struct {
	ID          uuid.UUID
	PostID      uuid.UUID
	ParentID    *uuid.UUID
	Nickname    string
	Email       string
	Content     string
	Status      CommentStatus
	CreatedAt   time.Time
	ModeratedAt *time.Time
}

type ManagedComment struct {
	Comment
	PostSlug     string
	PostTitle    string
	PostAuthorID uuid.UUID
}
