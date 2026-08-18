package model

import "github.com/google/uuid"

type Admin struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Nickname     string
	AvatarImg    string
	Role         string
}
