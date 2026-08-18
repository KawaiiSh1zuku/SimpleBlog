package service

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapStoreAvailability(t *testing.T) {
	retryable := &pgconn.PgError{Code: "40P01"}
	if !errors.Is(mapStoreAvailability(retryable), ErrUnavailable) {
		t.Fatal("retryable PostgreSQL error was not mapped to ErrUnavailable")
	}
	original := errors.New("unexpected")
	if !errors.Is(mapStoreAvailability(original), original) {
		t.Fatal("non-retryable error was changed")
	}
}
