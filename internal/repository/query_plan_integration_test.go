package repository_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepresentativePostgreSQLQueryPlans(t *testing.T) {
	databaseURL := repositoryTestDatabaseURL(t)
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set and SIMPLEBLOG_TEST_LOCAL_DB is not enabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer admin.Close(context.Background())
	schema := "simpleblog_plans_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatalf("create temporary schema: %v", err)
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
	seedRepresentativeData(t, ctx, pool)

	authorID := planUUID(1)
	queries := []struct {
		name      string
		query     string
		args      []any
		required  []string
		forbidden []string
	}{
		{
			name: "public posts",
			query: `SELECT p.id, p.author_id, p.category_id, p.slug, p.title, COALESCE(p.summary, ''),
			                p.tags, p.published_at, u.nickname, COALESCE(u.avatar_url, ''), c.name, c.slug
			         FROM posts p JOIN users u ON u.id = p.author_id JOIN categories c ON c.id = p.category_id
			         WHERE p.status = 'published' ORDER BY p.published_at DESC, p.id DESC LIMIT $1`,
			args: []any{21}, required: []string{"posts_public_idx"}, forbidden: []string{"Seq Scan on posts"},
		},
		{
			name: "public posts by category",
			query: `SELECT p.id FROM posts p JOIN categories c ON c.id = p.category_id
			         WHERE p.status = 'published' AND LOWER(c.slug) = LOWER($1)
			         ORDER BY p.published_at DESC, p.id DESC LIMIT $2`,
			args: []any{"category-777", 21}, required: []string{"categories_slug_uq", "posts_category_public_idx"}, forbidden: []string{"Seq Scan on posts"},
		},
		{
			name: "public posts by rare tag",
			query: `SELECT p.id FROM posts p WHERE p.status = 'published' AND p.tags @> ARRAY[$1]::TEXT[]
			         ORDER BY p.published_at DESC, p.id DESC LIMIT $2`,
			args: []any{"rare", 21}, required: []string{"posts_published_tags_gin"}, forbidden: []string{"Seq Scan on posts"},
		},
		{
			name: "managed posts by author",
			query: `SELECT id, title FROM posts WHERE author_id = $1
			         ORDER BY updated_at DESC, id DESC LIMIT $2`,
			args: []any{authorID, 51}, required: []string{"posts_author_updated_idx"}, forbidden: []string{"Seq Scan on posts"},
		},
		{
			name:      "managed posts all",
			query:     `SELECT id, title FROM posts ORDER BY updated_at DESC, id DESC LIMIT $1`,
			args:      []any{51},
			required:  []string{"posts_updated_idx"},
			forbidden: []string{"Seq Scan on posts"},
		},
		{
			name: "public comments by post",
			query: `SELECT c.id, c.email, c.content FROM comments c JOIN posts p ON p.id = c.post_id
			         WHERE LOWER(p.slug) = LOWER($1) AND p.status = 'published'
			         ORDER BY c.created_at, c.id`,
			args: []any{"post-1"}, required: []string{"posts_slug_uq", "comments_post_created_idx"}, forbidden: []string{"Seq Scan on comments"},
		},
		{
			name: "managed comments",
			query: `SELECT c.id, c.post_id, c.parent_id, c.nickname, c.email, c.content,
			                c.status, c.created_at, c.moderated_at, p.slug, p.title, p.author_id
			         FROM comments c JOIN posts p ON p.id = c.post_id
			         ORDER BY c.created_at DESC, c.id DESC LIMIT $1`,
			args: []any{51}, required: []string{"comments_created_idx"}, forbidden: []string{"Seq Scan on comments"},
		},
		{
			name: "managed comments by author",
			query: `WITH authored_posts AS MATERIALIZED (
			           SELECT id FROM posts WHERE author_id = $1
			         )
			         SELECT c.id, c.created_at
			         FROM authored_posts p
			         CROSS JOIN LATERAL (
			           SELECT id, created_at FROM comments
			           WHERE comments.post_id = p.id
			           ORDER BY comments.created_at DESC, comments.id DESC
			         ) c
			         ORDER BY c.created_at DESC, c.id DESC LIMIT $2`,
			args: []any{authorID, 51}, required: []string{"posts_author_updated_idx", "comments_post_created_idx"},
			forbidden: []string{"Seq Scan on comments", "Seq Scan on posts"},
		},
		{
			name: "login user",
			query: `SELECT id, password_hash FROM users
			         WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL`,
			args: []any{"USER-9999"}, required: []string{"users_username_active_uq"}, forbidden: []string{"Seq Scan on users"},
		},
	}

	for _, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			plan := explainAnalyze(t, ctx, pool, query.query, query.args...)
			t.Logf("%s\n%s", query.name, plan)
			for _, expected := range query.required {
				if !strings.Contains(plan, expected) {
					t.Errorf("plan does not use %s:\n%s", expected, plan)
				}
			}
			for _, unexpected := range query.forbidden {
				if strings.Contains(plan, unexpected) {
					t.Errorf("plan contains %s:\n%s", unexpected, plan)
				}
			}
		})
	}
}

func seedRepresentativeData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	statements := []string{
		`INSERT INTO users (id, username, password_hash, nickname, role)
		 SELECT ('00000000-0000-0000-0000-' || LPAD(value::text, 12, '0'))::uuid,
		        'user-' || value, 'hash', 'User ' || value,
		        CASE WHEN value = 1 THEN 'admin' ELSE 'contributor' END
		 FROM GENERATE_SERIES(1, 10000) AS value`,
		`INSERT INTO categories (id, name, slug)
		 SELECT ('10000000-0000-0000-0000-' || LPAD(value::text, 12, '0'))::uuid,
		        'Category ' || value, 'category-' || value
		 FROM GENERATE_SERIES(1, 1000) AS value`,
		`INSERT INTO posts (id, author_id, category_id, slug, title, summary, content, tags, status, published_at, created_at, updated_at)
		 SELECT ('20000000-0000-0000-0000-' || LPAD(value::text, 12, '0'))::uuid,
		        ('00000000-0000-0000-0000-' || LPAD((((value - 1) % 100) + 1)::text, 12, '0'))::uuid,
		        ('10000000-0000-0000-0000-' || LPAD((((value - 1) % 1000) + 1)::text, 12, '0'))::uuid,
		        'post-' || value, 'Post ' || value, 'Summary', REPEAT('x', 1024),
		        CASE WHEN value % 997 = 0 THEN ARRAY['go', 'rare']::text[] ELSE ARRAY['go']::text[] END,
		        CASE WHEN value % 5 = 0 THEN 'draft' ELSE 'published' END,
		        CASE WHEN value % 5 = 0 THEN NULL ELSE NOW() - value * INTERVAL '1 second' END,
		        NOW() - value * INTERVAL '1 second', NOW() - value * INTERVAL '1 second'
		 FROM GENERATE_SERIES(1, 20000) AS value`,
		`INSERT INTO comments (id, post_id, nickname, email, content, status, created_at)
		 SELECT ('30000000-0000-0000-0000-' || LPAD(value::text, 12, '0'))::uuid,
		        ('20000000-0000-0000-0000-' || LPAD((((value - 1) % 20000) + 1)::text, 12, '0'))::uuid,
		        'Visitor ' || value, 'visitor' || value || '@example.com', 'Comment ' || value,
		        'visible', NOW() - value * INTERVAL '1 second'
		 FROM GENERATE_SERIES(1, 50000) AS value`,
		`ANALYZE users`, `ANALYZE categories`, `ANALYZE posts`, `ANALYZE comments`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("seed representative data: %v\n%s", err, statement)
		}
	}
}

func explainAnalyze(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	rows, err := pool.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN query: %v", err)
	}
	defer rows.Close()
	lines := make([]string, 0)
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(lines, "\n")
}

func planUUID(value int) uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-" + leftPad12(value))
}

func leftPad12(value int) string {
	return fmt.Sprintf("%012d", value)
}
