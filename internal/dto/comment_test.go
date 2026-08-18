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
	tree := NewPublicCommentTree(comments)
	if len(tree) != 1 || len(tree[0].Replies) != 1 || len(tree[0].Replies[0].Replies) != 1 {
		t.Fatalf("tree shape = %+v", tree)
	}
	if tree[0].Nickname != deletedCommentNickname || tree[0].Content != "" {
		t.Fatalf("deleted node = %+v", tree[0])
	}
	if tree[0].Replies[0].Nickname != hiddenCommentNickname || tree[0].Replies[0].Content != "" {
		t.Fatalf("hidden node = %+v", tree[0].Replies[0])
	}
	encoded, err := json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "@example.com") || strings.Contains(string(encoded), "email") {
		t.Fatalf("public tree leaked email: %s", encoded)
	}
}
