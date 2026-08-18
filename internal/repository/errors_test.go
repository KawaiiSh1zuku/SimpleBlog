package repository

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgreSQLErrorClassification(t *testing.T) {
	tests := []struct {
		state      string
		unique     bool
		foreignKey bool
		input      bool
		retryable  bool
	}{
		{state: "23505", unique: true},
		{state: "23503", foreignKey: true},
		{state: "23001", foreignKey: true},
		{state: "23514", input: true},
		{state: "22001", input: true},
		{state: "40001", retryable: true},
		{state: "40P01", retryable: true},
		{state: "55P03", retryable: true},
	}
	for _, test := range tests {
		t.Run(test.state, func(t *testing.T) {
			err := fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: test.state})
			if IsUniqueViolation(err) != test.unique || IsForeignKeyViolation(err) != test.foreignKey ||
				IsInputViolation(err) != test.input || IsRetryable(err) != test.retryable || SQLState(err) != test.state {
				t.Fatalf("classification mismatch for %s", test.state)
			}
		})
	}
}
