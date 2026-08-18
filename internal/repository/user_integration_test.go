package repository_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestUserRepositoryLifecycleAndAdminGuard(t *testing.T) {
	databaseURL := repositoryTestDatabaseURL(t)
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set and SIMPLEBLOG_TEST_LOCAL_DB is not enabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer admin.Close(context.Background())
	schema := "simpleblog_repository_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create temporary schema: %v", err)
	}
	defer admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE")

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	defer pool.Close()
	applyRepositorySchema(t, ctx, pool)

	repo := repository.NewUserRepository(pool)
	firstAdmin := repositoryUser("Admin", model.RoleAdmin)
	if _, err := repo.CreateBootstrapAdmin(ctx, firstAdmin); err != nil {
		t.Fatalf("CreateBootstrapAdmin() error = %v", err)
	}
	if _, err := repo.CreateBootstrapAdmin(ctx, repositoryUser("other", model.RoleAdmin)); !errors.Is(err, repository.ErrAdminExists) {
		t.Fatalf("second CreateBootstrapAdmin() error = %v", err)
	}
	loaded, err := repo.GetByUsername(ctx, "ADMIN")
	if err != nil || loaded.ID != firstAdmin.ID {
		t.Fatalf("case-insensitive GetByUsername() = %+v, %v", loaded, err)
	}
	contributor := model.RoleContributor
	if _, err := repo.Update(ctx, firstAdmin.ID, repository.UpdateUserParams{Role: &contributor}); !errors.Is(err, repository.ErrLastAdmin) {
		t.Fatalf("Update(last admin role) error = %v", err)
	}

	secondAdmin := repositoryUser("second", model.RoleAdmin)
	if _, err := repo.Create(ctx, secondAdmin); err != nil {
		t.Fatalf("Create(second admin) error = %v", err)
	}
	if _, err := repo.Update(ctx, firstAdmin.ID, repository.UpdateUserParams{Role: &contributor}); err != nil {
		t.Fatalf("Update(role) error = %v", err)
	}
	if err := repo.SoftDelete(ctx, secondAdmin.ID); !errors.Is(err, repository.ErrLastAdmin) {
		t.Fatalf("SoftDelete(last admin) error = %v", err)
	}

	thirdAdmin := repositoryUser("third", model.RoleAdmin)
	if _, err := repo.Create(ctx, thirdAdmin); err != nil {
		t.Fatalf("Create(third admin) error = %v", err)
	}
	if err := repo.SoftDelete(ctx, secondAdmin.ID); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	if _, err := repo.GetByID(ctx, secondAdmin.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetByID(soft deleted) error = %v", err)
	}
}

func repositoryUser(username string, role model.Role) *model.User {
	return &model.User{
		ID: uuid.New(), Username: username, PasswordHash: "hash",
		Nickname: username, Role: role,
	}
}

func applyRepositorySchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	contents, err := os.ReadFile("../../migrations/000001_init.sql")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	schema := strings.TrimSpace(string(contents))
	schema = strings.TrimPrefix(schema, "BEGIN;")
	schema = strings.TrimSuffix(schema, "COMMIT;")
	for _, statement := range strings.Split(schema, ";") {
		if statement = strings.TrimSpace(statement); statement != "" {
			if _, err := pool.Exec(ctx, statement); err != nil {
				t.Fatalf("apply schema: %v\n%s", err, statement)
			}
		}
	}
}

func repositoryTestDatabaseURL(t *testing.T) string {
	t.Helper()
	if databaseURL := os.Getenv("TEST_DATABASE_URL"); databaseURL != "" {
		return databaseURL
	}
	if os.Getenv("SIMPLEBLOG_TEST_LOCAL_DB") != "1" {
		return ""
	}
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load local .env: %v", err)
	}
	database := config.DatabaseConfig{
		Host: repositoryEnvOr("DB_HOST", "localhost"), Port: repositoryEnvOr("DB_PORT", "5432"),
		Username: repositoryEnvOr("DB_USER", "postgres"), Password: repositoryEnvOr("DB_PASSWORD", "password"),
		DBName: repositoryEnvOr("DB_NAME", "mydb"), SSLMode: repositoryEnvOr("DB_SSLMODE", "disable"),
	}
	return database.ConnectionString()
}

func repositoryEnvOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
