package repository_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicContentQueriesNeverExposeDraftsAndPageStably(t *testing.T) {
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
	schema := "simpleblog_public_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	author := repositoryUser("public-writer", model.RoleContributor)
	author.Nickname = "Public Writer"
	if _, err := users.Create(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	general, err := categories.Create(ctx, &model.Category{ID: uuid.New(), Name: "General", Slug: "general"})
	if err != nil {
		t.Fatalf("create general category: %v", err)
	}
	tech, err := categories.Create(ctx, &model.Category{ID: uuid.New(), Name: "Tech", Slug: "tech"})
	if err != nil {
		t.Fatalf("create tech category: %v", err)
	}

	base := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	specs := []struct {
		slug       string
		categoryID uuid.UUID
		tags       []string
		published  bool
		at         time.Time
	}{
		{slug: "latest", categoryID: tech.ID, tags: []string{"go"}, published: true, at: base.Add(2 * time.Hour)},
		{slug: "middle", categoryID: general.ID, tags: []string{"go", "latex"}, published: true, at: base.Add(time.Hour)},
		{slug: "oldest", categoryID: general.ID, tags: []string{"latex"}, published: true, at: base},
		{slug: "draft-secret", categoryID: tech.ID, tags: []string{"secret"}, published: false},
	}
	for _, spec := range specs {
		created, err := posts.Create(ctx, &model.Post{
			ID: uuid.New(), AuthorID: author.ID, CategoryID: spec.categoryID,
			Slug: spec.slug, Title: spec.slug, Summary: "summary", Content: "# " + spec.slug,
			Tags: spec.tags, Status: model.PostStatusDraft,
		})
		if err != nil {
			t.Fatalf("create post %s: %v", spec.slug, err)
		}
		if spec.published {
			if _, err := posts.Publish(ctx, created.ID); err != nil {
				t.Fatalf("publish post %s: %v", spec.slug, err)
			}
			if _, err := pool.Exec(ctx, "UPDATE posts SET published_at = $2 WHERE id = $1", created.ID, spec.at); err != nil {
				t.Fatalf("set published_at for %s: %v", spec.slug, err)
			}
		}
	}

	publicService := service.NewPublicService(posts, categories)
	first, err := publicService.ListPosts(ctx, service.ListPublicPostsInput{Limit: "2"})
	if err != nil {
		t.Fatalf("ListPosts(first) error = %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].Slug != "latest" || first.Items[1].Slug != "middle" || first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}
	second, err := publicService.ListPosts(ctx, service.ListPublicPostsInput{Limit: "2", Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("ListPosts(second) error = %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].Slug != "oldest" || second.NextCursor != "" {
		t.Fatalf("second page = %+v", second)
	}

	filtered, err := publicService.ListPosts(ctx, service.ListPublicPostsInput{CategorySlug: "GENERAL", Tag: "GO"})
	if err != nil || len(filtered.Items) != 1 || filtered.Items[0].Slug != "middle" {
		t.Fatalf("filtered page = %+v, %v", filtered, err)
	}
	tags, err := publicService.ListTags(ctx)
	if err != nil || len(tags) != 2 || tags[0] != "go" || tags[1] != "latex" {
		t.Fatalf("ListTags() = %v, %v", tags, err)
	}
	detail, err := publicService.GetPost(ctx, "middle")
	if err != nil || detail.Content != "# middle" || detail.AuthorNickname != "Public Writer" || detail.CategorySlug != "general" {
		t.Fatalf("GetPost(middle) = %+v, %v", detail, err)
	}
	if _, err := publicService.GetPost(ctx, "draft-secret"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetPost(draft) error = %v", err)
	}
}
