package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type testEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestSeedCoversAllStatesAndCommentCounts(t *testing.T) {
	store := newSeedStore()
	if len(store.posts) != 10 {
		t.Fatalf("posts = %d, want 10", len(store.posts))
	}
	if len(store.comments) != 45 {
		t.Fatalf("comments = %d, want 45", len(store.comments))
	}
	if len(store.links) < 4 {
		t.Fatalf("links = %d, want at least 4", len(store.links))
	}

	postStatuses := map[model.PostStatus]bool{}
	commentStatuses := map[model.CommentStatus]bool{}
	roles := map[model.Role]bool{}
	counts := make(map[uuid.UUID]int)
	hasCover, hasNoCover := false, false
	for _, user := range store.users {
		roles[user.Role] = true
	}
	for _, post := range store.posts {
		postStatuses[post.Status] = true
		hasCover = hasCover || post.CoverImg != nil
		hasNoCover = hasNoCover || post.CoverImg == nil
	}
	for _, comment := range store.comments {
		counts[comment.PostID]++
		commentStatuses[comment.Status] = true
	}
	actualCounts := make([]int, 0, len(store.posts))
	for _, post := range store.posts {
		actualCounts = append(actualCounts, counts[post.ID])
	}
	sort.Ints(actualCounts)
	for index, count := range actualCounts {
		if count != index {
			t.Fatalf("sorted comment counts = %v, want 0..9", actualCounts)
		}
	}
	if !postStatuses[model.PostStatusDraft] || !postStatuses[model.PostStatusPublished] {
		t.Fatalf("post statuses = %v", postStatuses)
	}
	for _, status := range []model.CommentStatus{model.CommentStatusVisible, model.CommentStatusHidden, model.CommentStatusDeleted} {
		if !commentStatuses[status] {
			t.Fatalf("missing comment status %q", status)
		}
	}
	if !roles[model.RoleAdmin] || !roles[model.RoleContributor] || !hasCover || !hasNoCover {
		t.Fatalf("roles=%v hasCover=%v hasNoCover=%v", roles, hasCover, hasNoCover)
	}
}

func TestPublicRoutesHideDraftsAndExposeCommentStates(t *testing.T) {
	app := newMockApp()
	envelope := performJSON(t, app, http.MethodGet, "/posts?limit=100", nil, nil)
	if envelope.Code != http.StatusOK {
		t.Fatalf("code = %d", envelope.Code)
	}
	var page dto.PublicPostPageResponse
	decodeData(t, envelope, &page)
	if len(page.Items) != 7 {
		t.Fatalf("published posts = %d, want 7", len(page.Items))
	}

	draft := performJSON(t, app, http.MethodGet, "/posts/draft-site-navigation", nil, nil)
	if draft.Code != http.StatusNotFound {
		t.Fatalf("draft detail code = %d, want 404", draft.Code)
	}

	comments := performJSON(t, app, http.MethodGet, "/posts/blog-permission-model/comments", nil, nil)
	var tree []*dto.PublicCommentResponse
	decodeData(t, comments, &tree)
	statuses := make(map[model.CommentStatus]bool)
	var visit func([]*dto.PublicCommentResponse)
	visit = func(nodes []*dto.PublicCommentResponse) {
		for _, node := range nodes {
			statuses[node.Status] = true
			if node.Status != model.CommentStatusVisible && (node.Content != "" || node.AvatarURL != "") {
				t.Fatalf("redacted comment contains public content: %+v", node)
			}
			visit(node.Replies)
		}
	}
	visit(tree)
	for _, status := range []model.CommentStatus{model.CommentStatusVisible, model.CommentStatusHidden, model.CommentStatusDeleted} {
		if !statuses[status] {
			t.Fatalf("public tree missing status %q: %v", status, statuses)
		}
	}
}

func TestAuthenticationAndContributorIsolation(t *testing.T) {
	app := newMockApp()
	missingTurnstile := performJSON(t, app, http.MethodPost, "/auth/login", map[string]any{"username": "admin", "password": "admin123456"}, nil)
	if missingTurnstile.Code != http.StatusBadRequest {
		t.Fatalf("login without turnstile code = %d", missingTurnstile.Code)
	}
	login := performJSON(t, app, http.MethodPost, "/auth/login", map[string]any{"username": "writer", "password": "writer123456"}, map[string]string{"X-Turnstile-Response": "mock"})
	var loginData dto.LoginResponse
	decodeData(t, login, &loginData)
	if loginData.Token != contributorToken {
		t.Fatalf("token = %q", loginData.Token)
	}

	headers := bearer(contributorToken)
	forbidden := performJSON(t, app, http.MethodGet, "/admin/users", nil, headers)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("contributor users code = %d", forbidden.Code)
	}
	posts := performJSON(t, app, http.MethodGet, "/admin/posts?limit=100", nil, headers)
	var page dto.ManagedPostPageResponse
	decodeData(t, posts, &page)
	writerID := seedID("user-writer")
	if len(page.Items) == 0 {
		t.Fatal("contributor post page is empty")
	}
	for _, post := range page.Items {
		if post.AuthorID != writerID {
			t.Fatalf("contributor saw post owned by %s", post.AuthorID)
		}
	}
}

func TestPublishMutationMakesDraftPublic(t *testing.T) {
	app := newMockApp()
	id := seedID("post-08")
	headers := bearer(contributorToken)
	published := performJSON(t, app, http.MethodPost, "/admin/posts/"+id.String()+"/publish", nil, headers)
	var post dto.PostResponse
	decodeData(t, published, &post)
	if post.Status != model.PostStatusPublished || post.PublishedAt == nil {
		t.Fatalf("published post = %+v", post)
	}
	detail := performJSON(t, app, http.MethodGet, "/posts/draft-site-navigation", nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("published draft detail code = %d", detail.Code)
	}
	conflict := performJSON(t, app, http.MethodPost, "/admin/posts/"+id.String()+"/publish", nil, headers)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("duplicate publish code = %d", conflict.Code)
	}
}

func TestPaginationSearchRSSAndUpload(t *testing.T) {
	app := newMockApp()
	first := performJSON(t, app, http.MethodGet, "/posts?limit=2", nil, nil)
	var firstPage dto.PublicPostPageResponse
	decodeData(t, first, &firstPage)
	if len(firstPage.Items) != 2 || firstPage.NextCursor == "" {
		t.Fatalf("first page = %+v", firstPage)
	}
	second := performJSON(t, app, http.MethodGet, "/posts?limit=2&cursor="+firstPage.NextCursor, nil, nil)
	var secondPage dto.PublicPostPageResponse
	decodeData(t, second, &secondPage)
	if len(secondPage.Items) != 2 || secondPage.Items[0].ID == firstPage.Items[0].ID {
		t.Fatalf("second page = %+v", secondPage)
	}

	search := performJSON(t, app, http.MethodGet, "/search?q=LaTeX", nil, nil)
	var searchPage dto.PublicPostPageResponse
	decodeData(t, search, &searchPage)
	if len(searchPage.Items) != 1 || searchPage.Items[0].Slug != "markdown-latex-html" {
		t.Fatalf("search results = %+v", searchPage.Items)
	}

	rssRequest := httptest.NewRequest(http.MethodGet, "/rss/article", nil)
	rssResponse := httptest.NewRecorder()
	app.ServeHTTP(rssResponse, rssRequest)
	if rssResponse.Code != http.StatusOK || !strings.HasPrefix(rssResponse.Header().Get("Content-Type"), "application/rss+xml") || strings.Contains(rssResponse.Body.String(), "草稿") {
		t.Fatalf("rss status=%d content-type=%q body=%s", rssResponse.Code, rssResponse.Header().Get("Content-Type"), rssResponse.Body.String())
	}

	upload := performJSON(t, app, http.MethodPost, "/admin/uploads/presign", map[string]any{"filename": "cover.webp", "content_type": "image/webp", "size": 1024}, bearer(adminToken))
	var authorization dto.UploadAuthorizationResponse
	decodeData(t, upload, &authorization)
	if authorization.Method != http.MethodPost || !strings.HasSuffix(authorization.UploadURL, "/mock-upload") || authorization.PublicURL == "" {
		t.Fatalf("upload authorization = %+v", authorization)
	}

	assetRequest := httptest.NewRequest(http.MethodGet, "/mock-assets/cover-01.png", nil)
	assetResponse := httptest.NewRecorder()
	app.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != http.StatusOK || assetResponse.Header().Get("Content-Type") != "image/png" || assetResponse.Body.Len() < 100 {
		t.Fatalf("asset status=%d content-type=%q bytes=%d", assetResponse.Code, assetResponse.Header().Get("Content-Type"), assetResponse.Body.Len())
	}
}

func performJSON(t *testing.T, app http.Handler, method, target string, body any, headers map[string]string) testEnvelope {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, target, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	app.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var envelope testEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v; body=%s", err, recorder.Body.String())
	}
	return envelope
}

func decodeData(t *testing.T, envelope testEnvelope, target any) {
	t.Helper()
	if envelope.Code != http.StatusOK {
		t.Fatalf("business code = %d, msg = %q", envelope.Code, envelope.Msg)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("decode data: %v; data=%s", err, envelope.Data)
	}
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}
