package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

func TestManagedPostListNeverContainsContent(t *testing.T) {
	response := NewManagedPostPageResponse([]model.PostSummary{{
		ID: uuid.New(), Slug: "post", Title: "Post", Tags: []string{"go"},
	}}, "next")
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "content") {
		t.Fatalf("managed post list leaked content field: %s", encoded)
	}
}
