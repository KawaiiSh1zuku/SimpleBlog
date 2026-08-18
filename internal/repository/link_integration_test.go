package repository_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestM8RepositoriesParameterizeInputsAndFilterFeeds(t *testing.T) {
	databaseURL := repositoryTestDatabaseURL(t)
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set and SIMPLEBLOG_TEST_LOCAL_DB is not enabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(context.Background())
	schema := "simpleblog_links_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE")
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	applyRepositorySchema(t, ctx, pool)

	payload := "x'); DROP TABLE friend_links; --"
	links := repository.NewLinkRepository(pool)
	created, err := links.Create(ctx, &model.FriendLink{ID: uuid.New(), Name: payload, Description: payload, URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != payload || created.Description != payload {
		t.Fatalf("payload did not round-trip: %+v", created)
	}
	var relation *string
	if err := pool.QueryRow(ctx, "SELECT TO_REGCLASS('friend_links')").Scan(&relation); err != nil || relation == nil {
		t.Fatalf("friend_links was altered by payload: relation=%v err=%v", relation, err)
	}

	userID, categoryID, publishedID, draftID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{"INSERT INTO users (id, username, password_hash, nickname, role) VALUES ($1, 'feed-author', 'hash', 'Author', 'admin')", []any{userID}},
		{"INSERT INTO categories (id, name, slug) VALUES ($1, 'Feed', 'feed')", []any{categoryID}},
		{"INSERT INTO posts (id, author_id, category_id, slug, title, content, status, published_at) VALUES ($1, $2, $3, 'published', 'Published', 'body', 'published', NOW())", []any{publishedID, userID, categoryID}},
		{"INSERT INTO posts (id, author_id, category_id, slug, title, content) VALUES ($1, $2, $3, 'draft', 'Draft', 'body')", []any{draftID, userID, categoryID}},
		{"INSERT INTO comments (id, post_id, nickname, email, content) VALUES ($1, $2, 'Visible', 'visible@example.com', 'visible')", []any{uuid.New(), publishedID}},
		{"INSERT INTO comments (id, post_id, nickname, email, content, status) VALUES ($1, $2, 'Hidden', 'hidden@example.com', 'hidden', 'hidden')", []any{uuid.New(), publishedID}},
		{"INSERT INTO comments (id, post_id, nickname, email, content) VALUES ($1, $2, 'Draft', 'draft@example.com', 'draft')", []any{uuid.New(), draftID}},
	} {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed feed fixture: %v", err)
		}
	}
	feeds := repository.NewFeedRepository(pool)
	articles, err := feeds.ListArticles(ctx)
	if err != nil || len(articles) != 1 || articles[0].Slug != "published" {
		t.Fatalf("ListArticles() = %+v, %v", articles, err)
	}
	comments, err := feeds.ListComments(ctx)
	if err != nil || len(comments) != 1 || comments[0].Nickname != "Visible" {
		t.Fatalf("ListComments() = %+v, %v", comments, err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO comments (id, post_id, nickname, email, content) VALUES ($1, $2, '<script>', 'xss@example.com', 'body')", uuid.New(), publishedID)
	if !repository.IsInputViolation(err) {
		t.Fatalf("comment plain-text constraint error = %v", err)
	}
}
