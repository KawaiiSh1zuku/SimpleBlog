package service

import (
	"errors"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
)

func TestManagedPaginationValidatesLimitAndCursorKind(t *testing.T) {
	if _, _, err := parseManagedPage("101", "", "posts"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("oversized limit error = %v", err)
	}
	cursor := repository.TimeCursor{At: time.Now().UTC(), ID: uuid.New()}
	encoded, err := encodeManagedCursor("posts", cursor)
	if err != nil {
		t.Fatal(err)
	}
	limit, decoded, err := parseManagedPage("25", encoded, "posts")
	if err != nil || limit != 25 || decoded.ID != cursor.ID || !decoded.At.Equal(cursor.At) {
		t.Fatalf("decoded page = %d, %+v, %v", limit, decoded, err)
	}
	if _, _, err := parseManagedPage("", encoded, "comments"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("cross-resource cursor error = %v", err)
	}
}
