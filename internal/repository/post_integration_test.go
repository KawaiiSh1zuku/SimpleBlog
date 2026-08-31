package repository

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestPostRepositoryCoverSearchAndTagFilter(t *testing.T) {
	databaseURL := repositoryTestDatabaseURL(t)
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set and SIMPLEBLOG_TEST_LOCAL_DB is not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	base, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer base.Close(context.Background())

	schemaName := "simpleblog_repository_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schemaName}.Sanitize()
	if _, err := base.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create temporary schema: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET search_path TO "+identifier+", public")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer func() {
		pool.Close()
		if _, err := base.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop temporary schema: %v", err)
		}
	}()
	applyRepositoryTestMigrations(t, ctx, pool)

	authorID := uuid.New()
	categoryID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, username, password_hash, nickname, role)
		VALUES ($1, 'author', 'hash', 'Author', 'admin')
	`, authorID); err != nil {
		t.Fatalf("insert author: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO categories (id, name, slug) VALUES ($1, 'Tech', 'tech')
	`, categoryID); err != nil {
		t.Fatalf("insert category: %v", err)
	}

	repository := NewPostRepository(pool)
	coverImg := "https://cdn.example.com/cover.webp"
	created, err := repository.Create(ctx, &model.Post{
		ID: uuid.New(), AuthorID: authorID, CategoryID: categoryID,
		Slug: "published-post", Title: "PostgreSQL Search", Summary: "indexed summary",
		CoverImg: &coverImg, Content: "Markdown body with unique-needle", Tags: []string{"go"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.CoverImg == nil || *created.CoverImg != coverImg {
		t.Fatalf("Create() cover_img = %v", created.CoverImg)
	}
	if _, err := repository.Publish(ctx, created.ID); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if _, err := repository.Create(ctx, &model.Post{
		ID: uuid.New(), AuthorID: authorID, CategoryID: categoryID,
		Slug: "draft-post", Title: "unique-needle draft", Content: "draft body", Tags: []string{"go"},
	}); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	results, err := repository.SearchPublished(ctx, SearchPublishedParams{Query: "UNIQUE-NEEDLE", Limit: 10})
	if err != nil {
		t.Fatalf("SearchPublished() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != created.ID || results[0].CoverImg == nil {
		t.Fatalf("SearchPublished() results = %+v", results)
	}
	for _, query := range []string{"%", "_' OR 1=1 --"} {
		results, err := repository.SearchPublished(ctx, SearchPublishedParams{Query: query, Limit: 10})
		if err != nil {
			t.Fatalf("SearchPublished(%q) error = %v", query, err)
		}
		if len(results) != 0 {
			t.Fatalf("SearchPublished(%q) returned %d posts", query, len(results))
		}
	}
	tagged, err := repository.ListPublished(ctx, ListPublishedParams{Tag: "go", Limit: 10})
	if err != nil || len(tagged) != 1 || tagged[0].ID != created.ID {
		t.Fatalf("ListPublished(tag) = %+v, %v", tagged, err)
	}
	empty := ""
	updated, err := repository.Update(ctx, created.ID, UpdatePostParams{CoverImg: &empty})
	if err != nil {
		t.Fatalf("Update(clear cover) error = %v", err)
	}
	if updated.CoverImg != nil {
		t.Fatalf("Update(clear cover) cover_img = %v", updated.CoverImg)
	}
}

func applyRepositoryTestMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	sort.Strings(files)
	for _, file := range files {
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin migration %s: %v", file, err)
		}
		for index, statement := range splitRepositoryTestMigration(string(contents)) {
			if _, err := tx.Exec(ctx, statement); err != nil {
				tx.Rollback(ctx)
				t.Fatalf("execute %s statement %d: %v", file, index+1, err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit migration %s: %v", file, err)
		}
	}
}

func splitRepositoryTestMigration(contents string) []string {
	contents = strings.TrimSpace(contents)
	contents = strings.TrimPrefix(contents, "BEGIN;")
	contents = strings.TrimSuffix(contents, "COMMIT;")
	var statements []string
	for _, statement := range strings.Split(contents, ";") {
		if statement = strings.TrimSpace(statement); statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}

func repositoryTestDatabaseURL(t *testing.T) string {
	t.Helper()
	if databaseURL := os.Getenv("TEST_DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}
	if os.Getenv("SIMPLEBLOG_TEST_LOCAL_DB") != "1" {
		return ""
	}
	if err := godotenv.Load(filepath.Join("..", "..", ".env")); err != nil {
		t.Fatalf("load local .env: %v", err)
	}
	database := config.DatabaseConfig{
		Host:     envOrRepositoryTest("DB_HOST", "localhost"),
		Port:     envOrRepositoryTest("DB_PORT", "5432"),
		Username: envOrRepositoryTest("DB_USER", "postgres"),
		Password: envOrRepositoryTest("DB_PASSWORD", "password"),
		DBName:   envOrRepositoryTest("DB_NAME", "mydb"),
		SSLMode:  envOrRepositoryTest("DB_SSLMODE", "disable"),
	}
	return database.ConnectionString()
}

func envOrRepositoryTest(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
