package service

import (
	"errors"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrForbidden          = errors.New("forbidden")
	ErrLastAdmin          = errors.New("cannot remove the last admin")
	ErrUnavailable        = errors.New("temporarily unavailable")
)

func mapStoreAvailability(err error) error {
	if repository.IsRetryable(err) {
		return ErrUnavailable
	}
	return err
}
