package dto

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

const (
	hiddenCommentNickname  = "评论已隐藏"
	deletedCommentNickname = "该评论已被删除"
)

type CreateCommentRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
	Nickname string     `json:"nickname"`
	Email    string     `json:"email"`
	Content  string     `json:"content"`
}

type PublicCommentResponse struct {
	ID        uuid.UUID                `json:"id"`
	ParentID  *uuid.UUID               `json:"parent_id"`
	Nickname  string                   `json:"nickname"`
	AvatarURL string                   `json:"avatar_url"`
	Content   string                   `json:"content"`
	Status    model.CommentStatus      `json:"status"`
	CreatedAt time.Time                `json:"created_at"`
	Replies   []*PublicCommentResponse `json:"replies"`
}

type ManagedCommentResponse struct {
	ID          uuid.UUID           `json:"id"`
	PostID      uuid.UUID           `json:"post_id"`
	PostSlug    string              `json:"post_slug"`
	PostTitle   string              `json:"post_title"`
	ParentID    *uuid.UUID          `json:"parent_id"`
	Nickname    string              `json:"nickname"`
	Email       string              `json:"email"`
	Content     string              `json:"content"`
	Status      model.CommentStatus `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	ModeratedAt *time.Time          `json:"moderated_at"`
}

type ManagedCommentPageResponse struct {
	Items      []ManagedCommentResponse `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
}

type CommentModerationResponse struct {
	ID          uuid.UUID           `json:"id"`
	Status      model.CommentStatus `json:"status"`
	ModeratedAt *time.Time          `json:"moderated_at"`
}

func NewPublicCommentResponse(comment *model.Comment, gravatarBaseURL string) *PublicCommentResponse {
	nickname := safePlainText(comment.Nickname)
	content := safePlainText(comment.Content)
	avatarURL := gravatarURL(gravatarBaseURL, comment.Email)
	switch comment.Status {
	case model.CommentStatusHidden:
		nickname = hiddenCommentNickname
		content = ""
		avatarURL = ""
	case model.CommentStatusDeleted:
		nickname = deletedCommentNickname
		content = ""
		avatarURL = ""
	}
	return &PublicCommentResponse{
		ID: comment.ID, ParentID: comment.ParentID, Nickname: nickname,
		AvatarURL: avatarURL, Content: content, Status: comment.Status, CreatedAt: comment.CreatedAt,
		Replies: make([]*PublicCommentResponse, 0),
	}
}

func NewPublicCommentTree(comments []model.Comment, gravatarBaseURL string) []*PublicCommentResponse {
	nodes := make(map[uuid.UUID]*PublicCommentResponse, len(comments))
	for index := range comments {
		nodes[comments[index].ID] = NewPublicCommentResponse(&comments[index], gravatarBaseURL)
	}
	roots := make([]*PublicCommentResponse, 0)
	for index := range comments {
		comment := &comments[index]
		node := nodes[comment.ID]
		if comment.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[*comment.ParentID]
		if !ok {
			// The database foreign key prevents this; treating a corrupt orphan as a root keeps the response usable.
			roots = append(roots, node)
			continue
		}
		parent.Replies = append(parent.Replies, node)
	}
	return roots
}

func gravatarURL(baseURL, email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return ""
	}
	// Gravatar specifies MD5 as a public avatar identifier, not as a security primitive.
	hash := md5.Sum([]byte(normalized))
	return strings.TrimRight(baseURL, "/") + "/" + fmt.Sprintf("%x", hash) + "?d=identicon"
}

func NewManagedCommentResponse(comment *model.ManagedComment) ManagedCommentResponse {
	return ManagedCommentResponse{
		ID: comment.ID, PostID: comment.PostID, PostSlug: comment.PostSlug,
		PostTitle: comment.PostTitle, ParentID: comment.ParentID,
		Nickname: safePlainText(comment.Nickname), Email: comment.Email, Content: safePlainText(comment.Content),
		Status: comment.Status, CreatedAt: comment.CreatedAt, ModeratedAt: comment.ModeratedAt,
	}
}

func NewManagedCommentResponses(comments []model.ManagedComment) []ManagedCommentResponse {
	result := make([]ManagedCommentResponse, 0, len(comments))
	for index := range comments {
		result = append(result, NewManagedCommentResponse(&comments[index]))
	}
	return result
}

func NewManagedCommentPageResponse(comments []model.ManagedComment, nextCursor string) ManagedCommentPageResponse {
	return ManagedCommentPageResponse{Items: NewManagedCommentResponses(comments), NextCursor: nextCursor}
}

func NewCommentModerationResponse(comment *model.Comment) CommentModerationResponse {
	return CommentModerationResponse{ID: comment.ID, Status: comment.Status, ModeratedAt: comment.ModeratedAt}
}
