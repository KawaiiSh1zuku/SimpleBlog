package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func IsUniqueViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505"
}

func IsForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && (pgError.Code == "23503" || pgError.Code == "23001")
}

func IsInputViolation(err error) bool {
	return hasSQLState(err, "22001", "22003", "22P02", "23502", "23514")
}

func IsRetryable(err error) bool {
	return hasSQLState(err, "40001", "40P01", "55P03", "53300")
}

func SQLState(err error) string {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code
	}
	return ""
}

func hasSQLState(err error, states ...string) bool {
	state := SQLState(err)
	for _, candidate := range states {
		if state == candidate {
			return true
		}
	}
	return false
}
