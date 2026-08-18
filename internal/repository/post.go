package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepository struct {
	DB *pgxpool.Pool
}

type UpdatePostParams struct {
	CategoryID *uuid.UUID
	Slug       *string
	Title      *string
	Summary    *string
	Content    *string
	Tags       *[]string
}

type PublishedCursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type ListPublishedParams struct {
	Limit        int
	CategorySlug string
	Tag          string
	Cursor       *PublishedCursor
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository { return &PostRepository{DB: db} }

func (r *PostRepository) List(ctx context.Context, authorID *uuid.UUID) ([]model.Post, error) {
	query := postSelect
	args := make([]any, 0, 1)
	if authorID != nil {
		query += " WHERE author_id = $1"
		args = append(args, *authorID)
	}
	query += " ORDER BY updated_at DESC, id DESC"
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]model.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, *post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	return scanPost(r.DB.QueryRow(ctx, postSelect+" WHERE id = $1", id))
}

func (r *PostRepository) ListPublished(ctx context.Context, params ListPublishedParams) ([]model.PublicPost, error) {
	query := publicPostSelect + " WHERE p.status = 'published'"
	args := make([]any, 0, 5)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if params.CategorySlug != "" {
		query += " AND c.slug = " + addArgument(params.CategorySlug)
	}
	if params.Tag != "" {
		query += " AND p.tags @> ARRAY[" + addArgument(params.Tag) + "]::TEXT[]"
	}
	if params.Cursor != nil {
		publishedAt := addArgument(params.Cursor.PublishedAt)
		id := addArgument(params.Cursor.ID)
		query += " AND (p.published_at, p.id) < (" + publishedAt + ", " + id + ")"
	}
	query += " ORDER BY p.published_at DESC, p.id DESC LIMIT " + addArgument(params.Limit)

	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]model.PublicPost, 0)
	for rows.Next() {
		post, err := scanPublicPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, *post)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepository) GetPublishedBySlug(ctx context.Context, slug string) (*model.PublicPost, error) {
	return scanPublicPost(r.DB.QueryRow(ctx, publicPostSelect+`
		WHERE p.status = 'published' AND p.slug = $1
	`, slug))
}

func (r *PostRepository) ListPublishedTags(ctx context.Context) ([]string, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT DISTINCT tag
		FROM posts
		CROSS JOIN LATERAL UNNEST(tags) AS tag
		WHERE status = 'published'
		ORDER BY tag
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *PostRepository) Create(ctx context.Context, post *model.Post) (*model.Post, error) {
	return scanPost(r.DB.QueryRow(ctx, `
		INSERT INTO posts (id, author_id, category_id, slug, title, summary, content, tags, status)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, 'draft')
		RETURNING id, author_id, category_id, slug, title, COALESCE(summary, ''),
		          content, tags, status, published_at, created_at, updated_at
	`, post.ID, post.AuthorID, post.CategoryID, post.Slug, post.Title, post.Summary, post.Content, post.Tags))
}

func (r *PostRepository) Update(ctx context.Context, id uuid.UUID, params UpdatePostParams) (*model.Post, error) {
	var summary string
	hasSummary := params.Summary != nil
	if hasSummary {
		summary = *params.Summary
	}
	return scanPost(r.DB.QueryRow(ctx, `
		UPDATE posts
		SET category_id = COALESCE($2, category_id),
		    slug = COALESCE($3, slug),
		    title = COALESCE($4, title),
		    summary = CASE WHEN $5 THEN NULLIF($6, '') ELSE summary END,
		    content = COALESCE($7, content),
		    tags = COALESCE($8, tags),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, author_id, category_id, slug, title, COALESCE(summary, ''),
		          content, tags, status, published_at, created_at, updated_at
	`, id, params.CategoryID, params.Slug, params.Title, hasSummary, summary, params.Content, params.Tags))
}

func (r *PostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	command, err := r.DB.Exec(ctx, "DELETE FROM posts WHERE id = $1", id)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PostRepository) Publish(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	return scanPost(r.DB.QueryRow(ctx, `
		UPDATE posts
		SET status = 'published', published_at = COALESCE(published_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND status = 'draft'
		RETURNING id, author_id, category_id, slug, title, COALESCE(summary, ''),
		          content, tags, status, published_at, created_at, updated_at
	`, id))
}

func (r *PostRepository) Withdraw(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	return scanPost(r.DB.QueryRow(ctx, `
		UPDATE posts
		SET status = 'draft', updated_at = NOW()
		WHERE id = $1 AND status = 'published'
		RETURNING id, author_id, category_id, slug, title, COALESCE(summary, ''),
		          content, tags, status, published_at, created_at, updated_at
	`, id))
}

const postSelect = `
	SELECT id, author_id, category_id, slug, title, COALESCE(summary, ''),
	       content, tags, status, published_at, created_at, updated_at
	FROM posts`

const publicPostSelect = `
	SELECT p.id, p.author_id, p.category_id, p.slug, p.title, COALESCE(p.summary, ''),
	       p.content, p.tags, p.status, p.published_at, p.created_at, p.updated_at,
	       u.nickname, COALESCE(u.avatar_url, ''), c.name, c.slug
	FROM posts p
	JOIN users u ON u.id = p.author_id
	JOIN categories c ON c.id = p.category_id`

func scanPost(row rowScanner) (*model.Post, error) {
	var post model.Post
	if err := row.Scan(
		&post.ID, &post.AuthorID, &post.CategoryID, &post.Slug, &post.Title,
		&post.Summary, &post.Content, &post.Tags, &post.Status, &post.PublishedAt,
		&post.CreatedAt, &post.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &post, nil
}

func scanPublicPost(row rowScanner) (*model.PublicPost, error) {
	var post model.PublicPost
	if err := row.Scan(
		&post.ID, &post.AuthorID, &post.CategoryID, &post.Slug, &post.Title,
		&post.Summary, &post.Content, &post.Tags, &post.Status, &post.PublishedAt,
		&post.CreatedAt, &post.UpdatedAt, &post.AuthorNickname, &post.AuthorAvatarURL,
		&post.CategoryName, &post.CategorySlug,
	); err != nil {
		return nil, err
	}
	return &post, nil
}
