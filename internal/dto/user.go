package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type UserResponse struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Nickname  string     `json:"nickname"`
	AvatarURL string     `json:"avatar_url"`
	Role      model.Role `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreateUserRequest struct {
	Username  string     `json:"username"`
	Password  string     `json:"password"`
	Nickname  string     `json:"nickname"`
	AvatarURL string     `json:"avatar_url"`
	Role      model.Role `json:"role"`
}

type UpdateUserRequest struct {
	Username  *string     `json:"username"`
	Nickname  *string     `json:"nickname"`
	AvatarURL *string     `json:"avatar_url"`
	Role      *model.Role `json:"role"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

func NewUserResponse(user *model.User) UserResponse {
	return UserResponse{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname,
		AvatarURL: user.AvatarURL, Role: user.Role,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

func NewUserResponses(users []model.User) []UserResponse {
	result := make([]UserResponse, 0, len(users))
	for index := range users {
		result = append(result, NewUserResponse(&users[index]))
	}
	return result
}
