package utils

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestLogErrorRedactsDatabaseDetail(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	LogError(context.Background(), "request failed", &pgconn.PgError{
		Code: "23505", ConstraintName: "users_username_active_uq",
		Message: "duplicate value", Detail: "visitor@example.com secret-token",
	})

	if !strings.Contains(logs.String(), `"sqlstate":"23505"`) ||
		!strings.Contains(logs.String(), `"constraint":"users_username_active_uq"`) {
		t.Fatalf("safe database fields missing: %s", logs.String())
	}
	for _, secret := range []string{"visitor@example.com", "secret-token", "duplicate value"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("error log leaked %q: %s", secret, logs.String())
		}
	}
}
