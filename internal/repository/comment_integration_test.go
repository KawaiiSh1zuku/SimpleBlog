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

func TestCommentRepositoryNestedRepliesAndTombstones(t *testing.T) {
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
	schema := "simpleblog_comments_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	comments := repository.NewCommentRepository(pool)
	author := repositoryUser("comment-author", model.RoleContributor)
	if _, err := users.Create(ctx, author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	category, err := categories.Create(ctx, &model.Category{ID: uuid.New(), Name: "General", Slug: "general"})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	createPost := func(slug string, publish bool) *model.Post {
		t.Helper()
		post, err := posts.Create(ctx, &model.Post{
			ID: uuid.New(), AuthorID: author.ID, CategoryID: category.ID,
			Slug: slug, Title: slug, Content: "body", Tags: []string{},
		})
		if err != nil {
			t.Fatalf("create post %s: %v", slug, err)
		}
		if publish {
			post, err = posts.Publish(ctx, post.ID)
			if err != nil {
				t.Fatalf("publish post %s: %v", slug, err)
			}
		}
		return post
	}
	createPost("draft", false)
	createPost("published", true)
	createPost("other", true)

	newComment := func(parentID *uuid.UUID, nickname string) *model.Comment {
		return &model.Comment{
			ID: uuid.New(), ParentID: parentID, Nickname: nickname,
			Email: strings.ToLower(nickname) + "@example.com", Content: nickname,
		}
	}
	if _, err := comments.CreateForPublishedPost(ctx, "draft", newComment(nil, "Draft")); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("CreateForPublishedPost(draft) error = %v", err)
	}
	root, err := comments.CreateForPublishedPost(ctx, "published", newComment(nil, "Root"))
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	reply, err := comments.CreateForPublishedPost(ctx, "published", newComment(&root.ID, "Reply"))
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if _, err := comments.CreateForPublishedPost(ctx, "published", newComment(&reply.ID, "Leaf")); err != nil {
		t.Fatalf("create nested reply: %v", err)
	}
	if _, err := comments.CreateForPublishedPost(ctx, "other", newComment(&root.ID, "Cross")); !repository.IsForeignKeyViolation(err) {
		t.Fatalf("create cross-post reply error = %v", err)
	}

	publicComments, exists, err := comments.ListPublicByPostSlug(ctx, "published")
	if err != nil || !exists || len(publicComments) != 3 {
		t.Fatalf("ListPublicByPostSlug() = %d, %v, %v", len(publicComments), exists, err)
	}
	for _, comment := range publicComments {
		if comment.Email == "" {
			t.Fatalf("public query did not load email for avatar generation: %+v", comment)
		}
	}
	if _, err := comments.Hide(ctx, root.ID); err != nil {
		t.Fatalf("Hide() error = %v", err)
	}
	if _, err := comments.Show(ctx, root.ID); err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	deleted, err := comments.Delete(ctx, root.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.Status != model.CommentStatusDeleted || deleted.Nickname != "该评论已被删除" || deleted.Email != "" || deleted.Content != "" {
		t.Fatalf("deleted tombstone = %+v", deleted)
	}
	if _, err := comments.CreateForPublishedPost(ctx, "published", newComment(&root.ID, "AfterDelete")); err != nil {
		t.Fatalf("reply to tombstone: %v", err)
	}
	publicComments, exists, err = comments.ListPublicByPostSlug(ctx, "published")
	if err != nil || !exists || len(publicComments) != 4 || publicComments[0].Status != model.CommentStatusDeleted {
		t.Fatalf("public comments after tombstone = %+v, %v, %v", publicComments, exists, err)
	}
	managed, err := comments.ListManaged(ctx, repository.ListCommentsParams{AuthorID: &author.ID, Limit: 100})
	if err != nil || len(managed) != 4 {
		t.Fatalf("ListManaged() = %d, %v", len(managed), err)
	}
	if _, err := comments.Delete(ctx, root.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("second Delete() error = %v", err)
	}
	missing, exists, err := comments.ListPublicByPostSlug(ctx, "missing")
	if err != nil || exists || len(missing) != 0 {
		t.Fatalf("missing public post = %v, %v, %v", missing, exists, err)
	}
}
