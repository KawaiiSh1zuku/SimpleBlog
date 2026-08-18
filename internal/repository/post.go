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

type TimeCursor struct {
	At time.Time
	ID uuid.UUID
}

type ListPostsParams struct {
	AuthorID *uuid.UUID
	Limit    int
	Cursor   *TimeCursor
}

type ListPublishedParams struct {
	Limit        int
	CategorySlug string
	Tag          string
	Cursor       *PublishedCursor
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository { return &PostRepository{DB: db} }

func (r *PostRepository) List(ctx context.Context, params ListPostsParams) ([]model.PostSummary, error) {
	query := postSummarySelect + " WHERE TRUE"
	args := make([]any, 0, 4)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if params.AuthorID != nil {
		query += " AND author_id = " + addArgument(*params.AuthorID)
	}
	if params.Cursor != nil {
		at := addArgument(params.Cursor.At)
		id := addArgument(params.Cursor.ID)
		query += " AND (updated_at, id) < (" + at + ", " + id + ")"
	}
	query += " ORDER BY updated_at DESC, id DESC LIMIT " + addArgument(params.Limit)
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts := make([]model.PostSummary, 0)
	for rows.Next() {
		post, err := scanPostSummary(rows)
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
	query := publicPostListSelect + " WHERE p.status = 'published'"
	args := make([]any, 0, 5)
	addArgument := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if params.CategorySlug != "" {
		query += " AND LOWER(c.slug) = LOWER(" + addArgument(params.CategorySlug) + ")"
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
		post, err := scanPublicPostList(rows)
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
	return scanPublicPostDetail(r.DB.QueryRow(ctx, publicPostDetailSelect+`
		WHERE p.status = 'published' AND LOWER(p.slug) = LOWER($1)
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

const postSummarySelect = `
	SELECT id, author_id, category_id, slug, title, COALESCE(summary, ''),
	       tags, status, published_at, created_at, updated_at
	FROM posts`

const publicPostListSelect = `
	SELECT p.id, p.author_id, p.category_id, p.slug, p.title, COALESCE(p.summary, ''),
	       p.tags, p.published_at,
	       u.nickname, COALESCE(u.avatar_url, ''), c.name, c.slug
	FROM posts p
	JOIN users u ON u.id = p.author_id
	JOIN categories c ON c.id = p.category_id`

const publicPostDetailSelect = `
	SELECT p.id, p.author_id, p.category_id, p.slug, p.title, COALESCE(p.summary, ''),
	       p.content, p.tags, p.published_at,
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

func scanPostSummary(row rowScanner) (*model.PostSummary, error) {
	var post model.PostSummary
	if err := row.Scan(
		&post.ID, &post.AuthorID, &post.CategoryID, &post.Slug, &post.Title,
		&post.Summary, &post.Tags, &post.Status, &post.PublishedAt,
		&post.CreatedAt, &post.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &post, nil
}

func scanPublicPostList(row rowScanner) (*model.PublicPost, error) {
	var post model.PublicPost
	if err := row.Scan(
		&post.ID, &post.AuthorID, &post.CategoryID, &post.Slug, &post.Title,
		&post.Summary, &post.Tags, &post.PublishedAt, &post.AuthorNickname, &post.AuthorAvatarURL,
		&post.CategoryName, &post.CategorySlug,
	); err != nil {
		return nil, err
	}
	return &post, nil
}

func scanPublicPostDetail(row rowScanner) (*model.PublicPost, error) {
	var post model.PublicPost
	if err := row.Scan(
		&post.ID, &post.AuthorID, &post.CategoryID, &post.Slug, &post.Title,
		&post.Summary, &post.Content, &post.Tags, &post.PublishedAt,
		&post.AuthorNickname, &post.AuthorAvatarURL, &post.CategoryName, &post.CategorySlug,
	); err != nil {
		return nil, err
	}
	return &post, nil
}
