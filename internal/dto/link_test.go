package dto

import (
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
)

func TestLinkResponseDropsUnsafeLegacyURLs(t *testing.T) {
	response := NewLinkResponse(&model.FriendLink{Name: "Example", URL: "javascript:alert(1)", FaviconURL: "https://user:pass@example.com/favicon.ico"})
	if response.URL != "" || response.FaviconURL != "" {
		t.Fatalf("unsafe URLs reached DTO: %+v", response)
	}
}
