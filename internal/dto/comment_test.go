package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

func TestNewPublicCommentTreePreservesRepliesAndRedactsSensitiveData(t *testing.T) {
	rootID := uuid.New()
	replyID := uuid.New()
	comments := []model.Comment{
		{ID: rootID, Nickname: "Root", Email: "root@example.com", Content: "root", Status: model.CommentStatusDeleted},
		{ID: replyID, ParentID: &rootID, Nickname: "Reply", Email: "reply@example.com", Content: "reply", Status: model.CommentStatusHidden},
		{ID: uuid.New(), ParentID: &replyID, Nickname: "Leaf", Email: "leaf@example.com", Content: "leaf", Status: model.CommentStatusVisible},
	}
	tree := NewPublicCommentTree(comments, "https://avatar.example/avatar")
	if len(tree) != 1 || len(tree[0].Replies) != 1 || len(tree[0].Replies[0].Replies) != 1 {
		t.Fatalf("tree shape = %+v", tree)
	}
	if tree[0].Nickname != deletedCommentNickname || tree[0].Content != "" {
		t.Fatalf("deleted node = %+v", tree[0])
	}
	if tree[0].Replies[0].Nickname != hiddenCommentNickname || tree[0].Replies[0].Content != "" {
		t.Fatalf("hidden node = %+v", tree[0].Replies[0])
	}
	leaf := tree[0].Replies[0].Replies[0]
	if leaf.AvatarURL != "https://avatar.example/avatar/b351c25b141dbc022b504ebf451b2b46?d=identicon" {
		t.Fatalf("visible avatar URL = %q", leaf.AvatarURL)
	}
	if tree[0].AvatarURL != "" || tree[0].Replies[0].AvatarURL != "" {
		t.Fatalf("placeholder nodes retained avatars: %+v", tree)
	}
	encoded, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "@example.com") || strings.Contains(string(encoded), "email") {
		t.Fatalf("public tree leaked email: %s", encoded)
	}
}

func TestGravatarURLNormalizesEmailAndMirror(t *testing.T) {
	got := gravatarURL("https://cravatar.example/avatar/", "  MyEmailAddress@example.com ")
	want := "https://cravatar.example/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346?d=identicon"
	if got != want {
		t.Fatalf("gravatarURL() = %q, want %q", got, want)
	}
}

func TestCommentResponsesNeutralizeLegacyHTMLDelimiters(t *testing.T) {
	comment := &model.Comment{ID: uuid.New(), Nickname: "<b>name</b>", Content: "<script>alert(1)</script>\x00", Status: model.CommentStatusVisible}
	response := NewPublicCommentResponse(comment, "https://avatar.example/avatar")
	if strings.ContainsAny(response.Nickname+response.Content, "<>") || strings.ContainsRune(response.Content, '\x00') {
		t.Fatalf("legacy comment was not neutralized: %+v", response)
	}
}
