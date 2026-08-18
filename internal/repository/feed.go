package repository

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

const FeedLimit = 50

type FeedRepository struct{ DB *pgxpool.Pool }

func NewFeedRepository(db *pgxpool.Pool) *FeedRepository { return &FeedRepository{DB: db} }

func (r *FeedRepository) ListArticles(ctx context.Context) ([]model.FeedArticle, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, slug, title, COALESCE(summary, ''), published_at
		FROM posts
		WHERE status = 'published'
		ORDER BY published_at DESC, id DESC
		LIMIT $1
	`, FeedLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FeedArticle, 0, FeedLimit)
	for rows.Next() {
		var item model.FeedArticle
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &item.Summary, &item.PublishedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *FeedRepository) ListComments(ctx context.Context) ([]model.FeedComment, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT c.id, p.slug, p.title, c.nickname, c.content, c.created_at
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.status = 'visible' AND p.status = 'published'
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $1
	`, FeedLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.FeedComment, 0, FeedLimit)
	for rows.Next() {
		var item model.FeedComment
		if err := rows.Scan(&item.ID, &item.PostSlug, &item.PostTitle, &item.Nickname, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
