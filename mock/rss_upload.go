package main

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/google/uuid"
)

func (app *mockServer) articleRSS(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	items := make([]model.FeedArticle, 0, len(app.store.posts))
	for _, post := range app.store.posts {
		if post.Status == model.PostStatusPublished && post.PublishedAt != nil {
			items = append(items, model.FeedArticle{ID: post.ID, Slug: post.Slug, Title: post.Title, Summary: post.Summary, PublishedAt: *post.PublishedAt})
		}
	}
	app.store.mu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool { return items[i].PublishedAt.After(items[j].PublishedAt) })
	if len(items) > 50 {
		items = items[:50]
	}
	app.writeRSS(w, dto.NewArticleRSS("SimpleBlog Mock", "SimpleBlog 固定测试数据", "http://localhost:8090", items))
}

func (app *mockServer) commentRSS(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	items := make([]model.FeedComment, 0, len(app.store.comments))
	for _, comment := range app.store.comments {
		if comment.Status != model.CommentStatusVisible {
			continue
		}
		post, exists := app.store.postByID(comment.PostID)
		if !exists || post.Status != model.PostStatusPublished {
			continue
		}
		items = append(items, model.FeedComment{ID: comment.ID, PostSlug: post.Slug, PostTitle: post.Title, Nickname: comment.Nickname, Content: comment.Content, CreatedAt: comment.CreatedAt})
	}
	app.store.mu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > 50 {
		items = items[:50]
	}
	app.writeRSS(w, dto.NewCommentRSS("SimpleBlog Mock", "SimpleBlog 固定测试数据", "http://localhost:8090", items))
}

func (app *mockServer) writeRSS(w http.ResponseWriter, feed dto.RSS) {
	body, err := dto.MarshalRSS(feed)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (app *mockServer) presignUpload(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateUploadRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Filename = strings.TrimSpace(filepath.Base(request.Filename))
	allowedType := request.ContentType == "image/jpeg" || request.ContentType == "image/png" || request.ContentType == "image/webp" || request.ContentType == "image/gif"
	if request.Filename == "" || request.Filename == "." || !allowedType || request.Size < 1 || request.Size > 10*1024*1024 {
		badRequest(w)
		return
	}
	objectKey := "blog/uploads/" + nowUTC().Format("20060102") + "/" + uuid.NewString() + "-" + url.PathEscape(request.Filename)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	uploadURL := scheme + "://" + r.Host + "/mock-upload"
	publicURL := scheme + "://" + r.Host + "/mock-assets/upload-" + url.PathEscape(uuid.NewString()) + ".png"
	response.Success(w, dto.UploadAuthorizationResponse{
		Method: "POST", ObjectKey: objectKey, UploadURL: uploadURL,
		Fields:    map[string]string{"key": objectKey, "policy": "mock-policy", "x-amz-algorithm": "MOCK"},
		PublicURL: publicURL, ExpiresAt: nowUTC().Add(10 * time.Minute),
	})
}

func (app *mockServer) mockUpload(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		badRequest(w)
		return
	}
	_, _ = io.Copy(io.Discard, r.Body)
	w.WriteHeader(http.StatusNoContent)
}
