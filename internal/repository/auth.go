package repository

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	DB *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		DB: db,
	}
}

func (r *AuthRepository) GetByUsername(
	ctx context.Context,
	username string,
) (*model.Admin, error) {

	var admin model.Admin

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT
			id,
			username,
			password_hash,
			nickname,
			avatar_img,
			role
		FROM admins
		WHERE username = $1
		`,
		username,
	).Scan(
		&admin.ID,
		&admin.Username,
		&admin.PasswordHash,
		&admin.Nickname,
		&admin.AvatarImg,
		&admin.Role,
	)

	if err != nil {
		return nil, err
	}

	return &admin, nil
}
