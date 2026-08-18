package dto

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

func TestCommentRSSIsValidXMLAndNeutralizesLegacyHTML(t *testing.T) {
	id := uuid.New()
	feed := NewCommentRSS("Blog", "Description", "https://blog.example.com/", []model.FeedComment{{
		ID: id, PostSlug: "hello-world", PostTitle: "Hello", Nickname: "<b>Visitor</b>", Content: "<script>alert(1)</script>", CreatedAt: time.Unix(1, 0),
	}})
	body, err := MarshalRSS(feed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "<script>") || !strings.Contains(string(body), "#comment-"+id.String()) {
		t.Fatalf("unsafe or incomplete RSS: %s", body)
	}
	var decoded RSS
	if err := xml.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	if decoded.Version != "2.0" || len(decoded.Channel.Items) != 1 || strings.ContainsAny(decoded.Channel.Items[0].Description, "<>") {
		t.Fatalf("decoded RSS = %+v", decoded)
	}
}
