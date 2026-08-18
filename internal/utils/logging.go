package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgconn"
)

// LogError records diagnostic categories without serializing error messages,
// which can contain SQL values, credentials, email addresses, or remote URLs.
func LogError(ctx context.Context, message string, err error) {
	attributes := []any{
		"request_id", chimiddleware.GetReqID(ctx),
		"error_type", fmt.Sprintf("%T", err),
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		attributes = append(attributes, "sqlstate", pgError.Code, "constraint", pgError.ConstraintName)
	}
	if errors.Is(err, context.Canceled) {
		attributes = append(attributes, "cause", "context_canceled")
	} else if errors.Is(err, context.DeadlineExceeded) {
		attributes = append(attributes, "cause", "deadline_exceeded")
	}
	slog.ErrorContext(ctx, message, attributes...)
}
