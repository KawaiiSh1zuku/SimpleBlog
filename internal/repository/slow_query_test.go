package repository

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSlowQueryTracerLogsFingerprintWithoutSQLArguments(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	tracer := NewSlowQueryTracer(250 * time.Millisecond)
	tracer.Now = func() time.Time { return now }
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL: "SELECT id FROM comments WHERE email = $1", Args: []any{"visitor@example.com"},
	})
	now = now.Add(300 * time.Millisecond)
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.NewCommandTag("SELECT 1"),
		Err:        &pgconn.PgError{Code: "57014", Detail: "visitor@example.com"},
	})

	if !strings.Contains(logs.String(), "query_fingerprint") || !strings.Contains(logs.String(), `"duration_ms":300`) {
		t.Fatalf("slow query fields missing: %s", logs.String())
	}
	for _, sensitive := range []string{"visitor@example.com", "SELECT id", "comments WHERE email"} {
		if strings.Contains(logs.String(), sensitive) {
			t.Fatalf("slow query log leaked %q: %s", sensitive, logs.String())
		}
	}
}
