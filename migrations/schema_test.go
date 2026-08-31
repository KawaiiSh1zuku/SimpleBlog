package migrations_test

import (
	"context"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestInitialSchema(t *testing.T) {
	databaseURL := testDatabaseURL(t)
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set and SIMPLEBLOG_TEST_LOCAL_DB is not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer conn.Close(context.Background())

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	schemaName := "simpleblog_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schemaName}.Sanitize()
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create temporary schema: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+identifier+", public"); err != nil {
		t.Fatalf("set search path: %v", err)
	}

	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(files)
	for _, file := range files {
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		for index, statement := range schemaStatements(string(contents)) {
			if _, err := tx.Exec(ctx, statement); err != nil {
				t.Fatalf("execute %s statement %d: %v\n%s", file, index+1, err, statement)
			}
		}
	}

	for _, table := range []string{"users", "categories", "posts", "comments", "friend_links"} {
		var relation *string
		if err := tx.QueryRow(ctx, "SELECT TO_REGCLASS($1)", table).Scan(&relation); err != nil {
			t.Fatalf("look up table %s: %v", table, err)
		}
		if relation == nil {
			t.Errorf("table %s was not created", table)
		}
	}

	assertPostDeleteCascadesCommentTree(t, ctx, tx)
	assertPostCoverImageColumn(t, ctx, tx)
	assertOptimizedIndexes(t, ctx, tx)
}

func assertOptimizedIndexes(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	for _, index := range []string{
		"posts_updated_idx", "posts_author_updated_idx", "posts_published_tags_gin",
		"posts_published_search_trgm_idx",
		"posts_category_public_idx", "comments_post_created_idx", "comments_created_idx",
		"friend_links_name_idx", "comments_visible_created_idx",
	} {
		var relation *string
		if err := tx.QueryRow(ctx, "SELECT TO_REGCLASS($1)", index).Scan(&relation); err != nil {
			t.Fatalf("look up index %s: %v", index, err)
		}
		if relation == nil {
			t.Errorf("index %s was not created", index)
		}
	}
}

func assertPostCoverImageColumn(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	var nullable string
	if err := tx.QueryRow(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'posts' AND column_name = 'cover_img'
	`).Scan(&nullable); err != nil {
		t.Fatalf("look up posts.cover_img: %v", err)
	}
	if nullable != "YES" {
		t.Fatalf("posts.cover_img is_nullable = %q, want YES", nullable)
	}
}

func assertPostDeleteCascadesCommentTree(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()
	userID := uuid.New()
	categoryID := uuid.New()
	postID := uuid.New()
	rootCommentID := uuid.New()
	replyID := uuid.New()

	statements := []struct {
		query string
		args  []any
	}{
		{
			query: "INSERT INTO users (id, username, password_hash, nickname, role) VALUES ($1, 'author', 'hash', 'Author', 'admin')",
			args:  []any{userID},
		},
		{
			query: "INSERT INTO categories (id, name, slug) VALUES ($1, 'General', 'general')",
			args:  []any{categoryID},
		},
		{
			query: "INSERT INTO posts (id, author_id, category_id, slug, title, content) VALUES ($1, $2, $3, 'post', 'Post', 'Content')",
			args:  []any{postID, userID, categoryID},
		},
		{
			query: "INSERT INTO comments (id, post_id, nickname, email, content) VALUES ($1, $2, 'Root', 'root@example.com', 'Root comment')",
			args:  []any{rootCommentID, postID},
		},
		{
			query: "INSERT INTO comments (id, post_id, parent_id, nickname, email, content) VALUES ($1, $2, $3, 'Reply', 'reply@example.com', 'Reply')",
			args:  []any{replyID, postID, rootCommentID},
		},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("prepare cascade test: %v", err)
		}
	}

	if _, err := tx.Exec(ctx, "DELETE FROM posts WHERE id = $1", postID); err != nil {
		t.Fatalf("delete post with comment tree: %v", err)
	}
	var count int
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM comments WHERE post_id = $1", postID).Scan(&count); err != nil {
		t.Fatalf("count comments after post deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("comment count after post deletion = %d, want 0", count)
	}
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	if databaseURL := os.Getenv("TEST_DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}
	if os.Getenv("SIMPLEBLOG_TEST_LOCAL_DB") != "1" {
		return ""
	}
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf("load local .env: %v", err)
	}

	database := config.DatabaseConfig{
		Host:     envOr("DB_HOST", "localhost"),
		Port:     envOr("DB_PORT", "5432"),
		Username: envOr("DB_USER", "postgres"),
		Password: envOr("DB_PASSWORD", "password"),
		DBName:   envOr("DB_NAME", "mydb"),
		SSLMode:  envOr("DB_SSLMODE", "disable"),
	}
	return database.ConnectionString()
}

func envOr(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func schemaStatements(contents string) []string {
	contents = strings.TrimSpace(contents)
	contents = strings.TrimPrefix(contents, "BEGIN;")
	contents = strings.TrimSuffix(contents, "COMMIT;")

	var statements []string
	for _, statement := range strings.Split(contents, ";") {
		statement = strings.TrimSpace(statement)
		if statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}
