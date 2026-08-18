package model

import (
	"github.com/google/uuid"
	"time"
)

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleContributor Role = "contributor"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleContributor
}

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Nickname     string
	AvatarURL    string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
