package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLastAdmin   = errors.New("cannot remove the last admin")
	ErrAdminExists = errors.New("an active admin already exists")
)

const adminGuardLockID int64 = 0x53424C4F4741444D

type UserRepository struct {
	DB *pgxpool.Pool
}

type UpdateUserParams struct {
	Username  *string
	Nickname  *string
	AvatarURL *string
	Role      *model.Role
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return scanUser(r.DB.QueryRow(ctx, `
		SELECT id, username, password_hash, nickname,
		       COALESCE(avatar_url, ''), role, created_at, updated_at, deleted_at
		FROM users
		WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL
	`, username))
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return scanUser(r.DB.QueryRow(ctx, `
		SELECT id, username, password_hash, nickname,
		       COALESCE(avatar_url, ''), role, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id))
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, username, password_hash, nickname,
		       COALESCE(avatar_url, ''), role, created_at, updated_at, deleted_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY LOWER(username), id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	return createUser(ctx, r.DB, user)
}

func (r *UserRepository) CreateBootstrapAdmin(ctx context.Context, user *model.User) (*model.User, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())

	if err := lockAdminGuard(ctx, tx); err != nil {
		return nil, err
	}
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM users
		WHERE role = 'admin' AND deleted_at IS NULL
	`).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAdminExists
	}

	created, err := createUser(ctx, tx, user)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, params UpdateUserParams) (*model.User, error) {
	if params.Role == nil {
		return updateUser(ctx, r.DB, id, params)
	}

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())

	if err := lockAdminGuard(ctx, tx); err != nil {
		return nil, err
	}
	var currentRole model.Role
	if err := tx.QueryRow(ctx, `
		SELECT role FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&currentRole); err != nil {
		return nil, err
	}
	if currentRole == model.RoleAdmin && *params.Role != model.RoleAdmin {
		if err := ensureAnotherAdmin(ctx, tx, id); err != nil {
			return nil, err
		}
	}

	updated, err := updateUser(ctx, tx, id, params)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	if err := lockAdminGuard(ctx, tx); err != nil {
		return err
	}
	var role model.Role
	if err := tx.QueryRow(ctx, `
		SELECT role FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, id).Scan(&role); err != nil {
		return err
	}
	if role == model.RoleAdmin {
		if err := ensureAnotherAdmin(ctx, tx, id); err != nil {
			return err
		}
	}

	command, err := tx.Exec(ctx, `
		UPDATE users
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	command, err := r.DB.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, passwordHash)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

type database interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type rowScanner interface {
	Scan(...any) error
}

func createUser(ctx context.Context, db database, user *model.User) (*model.User, error) {
	return scanUser(db.QueryRow(ctx, `
		INSERT INTO users (id, username, password_hash, nickname, avatar_url, role)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6)
		RETURNING id, username, password_hash, nickname,
		          COALESCE(avatar_url, ''), role, created_at, updated_at, deleted_at
	`, user.ID, user.Username, user.PasswordHash, user.Nickname, user.AvatarURL, user.Role))
}

func updateUser(ctx context.Context, db database, id uuid.UUID, params UpdateUserParams) (*model.User, error) {
	var avatarURL string
	hasAvatarURL := params.AvatarURL != nil
	if hasAvatarURL {
		avatarURL = *params.AvatarURL
	}

	return scanUser(db.QueryRow(ctx, `
		UPDATE users
		SET username = COALESCE($2, username),
		    nickname = COALESCE($3, nickname),
		    avatar_url = CASE WHEN $4 THEN NULLIF($5, '') ELSE avatar_url END,
		    role = COALESCE($6, role),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, username, password_hash, nickname,
		          COALESCE(avatar_url, ''), role, created_at, updated_at, deleted_at
	`, id, params.Username, params.Nickname, hasAvatarURL, avatarURL, params.Role))
}

func scanUser(row rowScanner) (*model.User, error) {
	var user model.User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.AvatarURL,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}

func lockAdminGuard(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", adminGuardLockID)
	return err
}

func ensureAnotherAdmin(ctx context.Context, tx pgx.Tx, excludedID uuid.UUID) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users
			WHERE role = 'admin' AND deleted_at IS NULL AND id <> $1
		)
	`, excludedID).Scan(&exists); err != nil {
		return fmt.Errorf("check remaining admins: %w", err)
	}
	if !exists {
		return ErrLastAdmin
	}
	return nil
}
