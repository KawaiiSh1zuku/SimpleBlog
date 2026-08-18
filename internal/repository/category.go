package repository

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	DB *pgxpool.Pool
}

type UpdateCategoryParams struct {
	Name        *string
	Slug        *string
	Description *string
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{DB: db}
}

func (r *CategoryRepository) List(ctx context.Context) ([]model.Category, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), created_at, updated_at
		FROM categories
		ORDER BY LOWER(name), id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := make([]model.Category, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Category, error) {
	return scanCategory(r.DB.QueryRow(ctx, `
		SELECT id, name, slug, COALESCE(description, ''), created_at, updated_at
		FROM categories
		WHERE id = $1
	`, id))
}

func (r *CategoryRepository) Create(ctx context.Context, category *model.Category) (*model.Category, error) {
	return scanCategory(r.DB.QueryRow(ctx, `
		INSERT INTO categories (id, name, slug, description)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id, name, slug, COALESCE(description, ''), created_at, updated_at
	`, category.ID, category.Name, category.Slug, category.Description))
}

func (r *CategoryRepository) Update(ctx context.Context, id uuid.UUID, params UpdateCategoryParams) (*model.Category, error) {
	var description string
	hasDescription := params.Description != nil
	if hasDescription {
		description = *params.Description
	}
	return scanCategory(r.DB.QueryRow(ctx, `
		UPDATE categories
		SET name = COALESCE($2, name),
		    slug = COALESCE($3, slug),
		    description = CASE WHEN $4 THEN NULLIF($5, '') ELSE description END,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, slug, COALESCE(description, ''), created_at, updated_at
	`, id, params.Name, params.Slug, hasDescription, description))
}

func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	command, err := r.DB.Exec(ctx, "DELETE FROM categories WHERE id = $1", id)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanCategory(row rowScanner) (*model.Category, error) {
	var category model.Category
	if err := row.Scan(
		&category.ID, &category.Name, &category.Slug, &category.Description,
		&category.CreatedAt, &category.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &category, nil
}
