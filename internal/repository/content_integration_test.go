package repository_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCategoryAndPostRepositoryLifecycle(t *testing.T) {
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
	schema := "simpleblog_content_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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

	users := repository.NewUserRepository(pool)
	categories := repository.NewCategoryRepository(pool)
	posts := repository.NewPostRepository(pool)
	author := repositoryUser("writer", model.RoleContributor)
	if _, err := users.Create(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	category := &model.Category{ID: uuid.New(), Name: "General", Slug: "general"}
	category, err = categories.Create(ctx, category)
	if err != nil {
		t.Fatalf("Create(category) error = %v", err)
	}
	if _, err := categories.Create(ctx, &model.Category{
		ID: uuid.New(), Name: "GENERAL", Slug: "other",
	}); !repository.IsUniqueViolation(err) {
		t.Fatalf("duplicate Create(category) error = %v", err)
	}
	description := "Updated description"
	updatedCategory, err := categories.Update(ctx, category.ID, repository.UpdateCategoryParams{Description: &description})
	if err != nil || updatedCategory.Description != description {
		t.Fatalf("Update(category) = %+v, %v", updatedCategory, err)
	}

	content := "# Formula\n\n$$E = mc^2$$"
	post := &model.Post{
		ID: uuid.New(), AuthorID: author.ID, CategoryID: category.ID,
		Slug: "formula", Title: "Formula", Content: content,
		Tags: []string{"go", "latex"}, Status: model.PostStatusDraft,
	}
	created, err := posts.Create(ctx, post)
	if err != nil {
		t.Fatalf("Create(post) error = %v", err)
	}
	if created.Content != content || created.Status != model.PostStatusDraft {
		t.Fatalf("created post = %+v", created)
	}
	emptySummary := ""
	updatedTags := []string{"updated"}
	updatedPost, err := posts.Update(ctx, created.ID, repository.UpdatePostParams{
		Summary: &emptySummary, Tags: &updatedTags,
	})
	if err != nil || updatedPost.Summary != "" || len(updatedPost.Tags) != 1 || updatedPost.Tags[0] != "updated" {
		t.Fatalf("Update(post) = %+v, %v", updatedPost, err)
	}
	if _, err := posts.Create(ctx, &model.Post{
		ID: uuid.New(), AuthorID: author.ID, CategoryID: category.ID,
		Slug: "FORMULA", Title: "Duplicate", Content: "body", Tags: []string{},
	}); !repository.IsUniqueViolation(err) {
		t.Fatalf("duplicate Create(post) error = %v", err)
	}

	authorPosts, err := posts.List(ctx, &author.ID)
	if err != nil || len(authorPosts) != 1 {
		t.Fatalf("List(author) = %d, %v", len(authorPosts), err)
	}
	allPosts, err := posts.List(ctx, nil)
	if err != nil || len(allPosts) != 1 {
		t.Fatalf("List(all) = %d, %v", len(allPosts), err)
	}

	published, err := posts.Publish(ctx, created.ID)
	if err != nil || published.PublishedAt == nil || published.Status != model.PostStatusPublished {
		t.Fatalf("Publish() = %+v, %v", published, err)
	}
	firstPublishedAt := *published.PublishedAt
	withdrawn, err := posts.Withdraw(ctx, created.ID)
	if err != nil || withdrawn.PublishedAt == nil || !withdrawn.PublishedAt.Equal(firstPublishedAt) {
		t.Fatalf("Withdraw() = %+v, %v", withdrawn, err)
	}
	republished, err := posts.Publish(ctx, created.ID)
	if err != nil || republished.PublishedAt == nil || !republished.PublishedAt.Equal(firstPublishedAt) {
		t.Fatalf("second Publish() = %+v, %v", republished, err)
	}

	if err := categories.Delete(ctx, category.ID); !repository.IsForeignKeyViolation(err) {
		t.Fatalf("Delete(referenced category) error = %v", err)
	}
	if err := posts.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete(post) error = %v", err)
	}
	if err := categories.Delete(ctx, category.ID); err != nil {
		t.Fatalf("Delete(category) error = %v", err)
	}
	if _, err := posts.GetByID(ctx, created.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetByID(deleted post) error = %v", err)
	}
}
