package repository

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepository struct{ DB *pgxpool.Pool }

type UpdateLinkParams struct {
	Name        *string
	Description *string
	URL         *string
	FaviconURL  *string
}

func NewLinkRepository(db *pgxpool.Pool) *LinkRepository { return &LinkRepository{DB: db} }

func (r *LinkRepository) List(ctx context.Context) ([]model.FriendLink, error) {
	rows, err := r.DB.Query(ctx, linkSelect+" ORDER BY LOWER(name), id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	links := make([]model.FriendLink, 0)
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, *link)
	}
	return links, rows.Err()
}

func (r *LinkRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FriendLink, error) {
	return scanLink(r.DB.QueryRow(ctx, linkSelect+" WHERE id = $1", id))
}

func (r *LinkRepository) Create(ctx context.Context, link *model.FriendLink) (*model.FriendLink, error) {
	return scanLink(r.DB.QueryRow(ctx, `
		INSERT INTO friend_links (id, name, description, url, favicon_url)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''))
		RETURNING id, name, COALESCE(description, ''), url, COALESCE(favicon_url, ''), created_at, updated_at
	`, link.ID, link.Name, link.Description, link.URL, link.FaviconURL))
}

func (r *LinkRepository) Update(ctx context.Context, id uuid.UUID, params UpdateLinkParams) (*model.FriendLink, error) {
	var description, faviconURL string
	if params.Description != nil {
		description = *params.Description
	}
	if params.FaviconURL != nil {
		faviconURL = *params.FaviconURL
	}
	return scanLink(r.DB.QueryRow(ctx, `
		UPDATE friend_links
		SET name = COALESCE($2, name),
		    description = CASE WHEN $3 THEN NULLIF($4, '') ELSE description END,
		    url = COALESCE($5, url),
		    favicon_url = CASE WHEN $6 THEN NULLIF($7, '') ELSE favicon_url END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, COALESCE(description, ''), url, COALESCE(favicon_url, ''), created_at, updated_at
	`, id, params.Name, params.Description != nil, description, params.URL, params.FaviconURL != nil, faviconURL))
}

func (r *LinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	command, err := r.DB.Exec(ctx, "DELETE FROM friend_links WHERE id = $1", id)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

const linkSelect = `
	SELECT id, name, COALESCE(description, ''), url, COALESCE(favicon_url, ''), created_at, updated_at
	FROM friend_links`

func scanLink(row rowScanner) (*model.FriendLink, error) {
	var link model.FriendLink
	if err := row.Scan(&link.ID, &link.Name, &link.Description, &link.URL, &link.FaviconURL, &link.CreatedAt, &link.UpdatedAt); err != nil {
		return nil, err
	}
	return &link, nil
}
