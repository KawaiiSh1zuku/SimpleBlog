package repository

import (
	"context"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	DB *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{DB: db}
}

func (r *CommentRepository) CreateForPublishedPost(ctx context.Context, postSlug string, comment *model.Comment) (*model.Comment, error) {
	return scanComment(r.DB.QueryRow(ctx, `
		INSERT INTO comments (id, post_id, parent_id, nickname, email, content)
		SELECT $1, p.id, $2, $3, $4, $5
		FROM posts p
		WHERE p.slug = $6 AND p.status = 'published'
		RETURNING id, post_id, parent_id, nickname, email, content, status, created_at, moderated_at
	`, comment.ID, comment.ParentID, comment.Nickname, comment.Email, comment.Content, postSlug))
}

func (r *CommentRepository) ListPublicByPostSlug(ctx context.Context, postSlug string) ([]model.Comment, bool, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT c.id, c.post_id, c.parent_id, c.nickname, c.content,
		       c.status, c.created_at, c.moderated_at
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE p.slug = $1 AND p.status = 'published'
		ORDER BY c.created_at, c.id
	`, postSlug)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	comments := make([]model.Comment, 0)
	for rows.Next() {
		comment, err := scanPublicComment(rows)
		if err != nil {
			return nil, false, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(comments) > 0 {
		return comments, true, nil
	}
	var exists bool
	if err := r.DB.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM posts WHERE slug = $1 AND status = 'published')
	`, postSlug).Scan(&exists); err != nil {
		return nil, false, err
	}
	return comments, exists, nil
}

func (r *CommentRepository) ListManaged(ctx context.Context, authorID *uuid.UUID) ([]model.ManagedComment, error) {
	query := managedCommentSelect
	args := make([]any, 0, 1)
	if authorID != nil {
		query += " WHERE p.author_id = $1"
		args = append(args, *authorID)
	}
	query += " ORDER BY c.created_at DESC, c.id DESC"
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := make([]model.ManagedComment, 0)
	for rows.Next() {
		comment, err := scanManagedComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *CommentRepository) GetManagedByID(ctx context.Context, id uuid.UUID) (*model.ManagedComment, error) {
	return scanManagedComment(r.DB.QueryRow(ctx, managedCommentSelect+" WHERE c.id = $1", id))
}

func (r *CommentRepository) Hide(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	return scanComment(r.DB.QueryRow(ctx, `
		UPDATE comments
		SET status = 'hidden', moderated_at = NOW()
		WHERE id = $1 AND status = 'visible'
		RETURNING id, post_id, parent_id, nickname, email, content, status, created_at, moderated_at
	`, id))
}

func (r *CommentRepository) Show(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	return scanComment(r.DB.QueryRow(ctx, `
		UPDATE comments
		SET status = 'visible', moderated_at = NOW()
		WHERE id = $1 AND status = 'hidden'
		RETURNING id, post_id, parent_id, nickname, email, content, status, created_at, moderated_at
	`, id))
}

func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	return scanComment(r.DB.QueryRow(ctx, `
		UPDATE comments
		SET status = 'deleted', nickname = '该评论已被删除', email = '', content = '', moderated_at = NOW()
		WHERE id = $1 AND status IN ('visible', 'hidden')
		RETURNING id, post_id, parent_id, nickname, email, content, status, created_at, moderated_at
	`, id))
}

const managedCommentSelect = `
	SELECT c.id, c.post_id, c.parent_id, c.nickname, c.email, c.content,
	       c.status, c.created_at, c.moderated_at,
	       p.slug, p.title, p.author_id
	FROM comments c
	JOIN posts p ON p.id = c.post_id`

func scanComment(row rowScanner) (*model.Comment, error) {
	var comment model.Comment
	if err := row.Scan(
		&comment.ID, &comment.PostID, &comment.ParentID, &comment.Nickname,
		&comment.Email, &comment.Content, &comment.Status, &comment.CreatedAt,
		&comment.ModeratedAt,
	); err != nil {
		return nil, err
	}
	return &comment, nil
}

func scanPublicComment(row rowScanner) (*model.Comment, error) {
	var comment model.Comment
	if err := row.Scan(
		&comment.ID, &comment.PostID, &comment.ParentID, &comment.Nickname,
		&comment.Content, &comment.Status, &comment.CreatedAt, &comment.ModeratedAt,
	); err != nil {
		return nil, err
	}
	return &comment, nil
}

func scanManagedComment(row rowScanner) (*model.ManagedComment, error) {
	var comment model.ManagedComment
	if err := row.Scan(
		&comment.ID, &comment.PostID, &comment.ParentID, &comment.Nickname,
		&comment.Email, &comment.Content, &comment.Status, &comment.CreatedAt,
		&comment.ModeratedAt, &comment.PostSlug, &comment.PostTitle, &comment.PostAuthorID,
	); err != nil {
		return nil, err
	}
	return &comment, nil
}
