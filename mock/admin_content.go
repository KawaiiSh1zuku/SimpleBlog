package main

import (
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/google/uuid"
)

var firstImagePattern = regexp.MustCompile(`(?i)(?:!\[[^\]]*\]\(|<img[^>]+src=["'])(https?://[^\s)"']+)`)

func (app *mockServer) listPosts(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parsePage(r, 50)
	if !ok {
		badRequest(w)
		return
	}
	actor := currentActor(r)
	app.store.mu.RLock()
	posts := make([]model.Post, 0, len(app.store.posts))
	for _, post := range app.store.posts {
		if actor.Role == model.RoleAdmin || post.AuthorID == actor.ID {
			posts = append(posts, post)
		}
	}
	app.store.mu.RUnlock()
	sortPosts(posts)
	page, cursor := paginate(posts, limit, offset)
	summaries := make([]model.PostSummary, 0, len(page))
	for _, post := range page {
		summaries = append(summaries, model.PostSummary{
			ID: post.ID, AuthorID: post.AuthorID, CategoryID: post.CategoryID, Slug: post.Slug,
			Title: post.Title, Summary: post.Summary, CoverImg: post.CoverImg, Tags: post.Tags,
			Status: post.Status, PublishedAt: post.PublishedAt, CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt,
		})
	}
	response.Success(w, dto.NewManagedPostPageResponse(summaries, cursor))
}

func (app *mockServer) getPost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "postID", "invalid post id")
	if !ok {
		return
	}
	actor := currentActor(r)
	app.store.mu.RLock()
	post, exists := app.store.postByID(id)
	if exists {
		copy := *post
		copy.Tags = append([]string(nil), post.Tags...)
		post = &copy
	}
	app.store.mu.RUnlock()
	if !exists {
		notFound(w)
		return
	}
	if !canManagePost(actor, post) {
		response.Error(w, http.StatusForbidden, "无权访问")
		return
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (app *mockServer) createPost(w http.ResponseWriter, r *http.Request) {
	var request dto.CreatePostRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Slug = strings.ToLower(strings.TrimSpace(request.Slug))
	request.Title = strings.TrimSpace(request.Title)
	request.Summary = strings.TrimSpace(request.Summary)
	request.CoverImg = strings.TrimSpace(request.CoverImg)
	if !validSlug(request.Slug) || !validNonEmpty(request.Title, 200) || strings.TrimSpace(request.Content) == "" || len(request.Content) > 1024*1024 || !validHTTPURL(request.CoverImg, true) {
		badRequest(w)
		return
	}
	tags, ok := normalizeTags(request.Tags)
	if !ok {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	if _, exists := app.store.categoryByID(request.CategoryID); !exists {
		notFound(w)
		return
	}
	if _, exists := app.store.postBySlug(request.Slug); exists {
		conflict(w)
		return
	}
	var cover *string
	if request.CoverImg != "" {
		cover = &request.CoverImg
	} else {
		cover = firstMockImageURL(request.Content)
	}
	now := nowUTC()
	post := model.Post{ID: uuid.New(), AuthorID: currentActor(r).ID, CategoryID: request.CategoryID, Slug: request.Slug, Title: request.Title, Summary: request.Summary, CoverImg: cover, Content: request.Content, Tags: tags, Status: model.PostStatusDraft, CreatedAt: now, UpdatedAt: now}
	app.store.posts = append(app.store.posts, post)
	response.Success(w, dto.NewPostResponse(&post))
}

func (app *mockServer) updatePost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "postID", "invalid post id")
	if !ok {
		return
	}
	var request dto.UpdatePostRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.CategoryID == nil && request.Slug == nil && request.Title == nil && request.Summary == nil && request.CoverImg == nil && request.Content == nil && request.Tags == nil {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	post, exists := app.store.postByID(id)
	if !exists {
		notFound(w)
		return
	}
	if !canManagePost(currentActor(r), post) {
		response.Error(w, http.StatusForbidden, "无权访问")
		return
	}
	if request.CategoryID != nil {
		if _, categoryExists := app.store.categoryByID(*request.CategoryID); !categoryExists {
			notFound(w)
			return
		}
		post.CategoryID = *request.CategoryID
	}
	if request.Slug != nil {
		value := strings.ToLower(strings.TrimSpace(*request.Slug))
		if !validSlug(value) {
			badRequest(w)
			return
		}
		if other, duplicate := app.store.postBySlug(value); duplicate && other.ID != id {
			conflict(w)
			return
		}
		post.Slug = value
	}
	if request.Title != nil {
		value := strings.TrimSpace(*request.Title)
		if !validNonEmpty(value, 200) {
			badRequest(w)
			return
		}
		post.Title = value
	}
	if request.Summary != nil {
		post.Summary = strings.TrimSpace(*request.Summary)
	}
	if request.Content != nil {
		if strings.TrimSpace(*request.Content) == "" || len(*request.Content) > 1024*1024 {
			badRequest(w)
			return
		}
		post.Content = *request.Content
	}
	if request.Tags != nil {
		values, tagsOK := normalizeTags(*request.Tags)
		if !tagsOK {
			badRequest(w)
			return
		}
		post.Tags = values
	}
	if request.CoverImg != nil {
		value := strings.TrimSpace(*request.CoverImg)
		if !validHTTPURL(value, true) {
			badRequest(w)
			return
		}
		if value == "" {
			post.CoverImg = firstMockImageURL(post.Content)
		} else {
			post.CoverImg = &value
		}
	} else if post.CoverImg == nil {
		post.CoverImg = firstMockImageURL(post.Content)
	}
	post.UpdatedAt = nowUTC()
	response.Success(w, dto.NewPostResponse(post))
}

func (app *mockServer) deletePost(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "postID", "invalid post id")
	if !ok {
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	for index := range app.store.posts {
		if app.store.posts[index].ID != id {
			continue
		}
		if !canManagePost(currentActor(r), &app.store.posts[index]) {
			response.Error(w, http.StatusForbidden, "无权访问")
			return
		}
		app.store.posts = removeAt(app.store.posts, index)
		comments := app.store.comments[:0]
		for _, comment := range app.store.comments {
			if comment.PostID != id {
				comments = append(comments, comment)
			}
		}
		app.store.comments = comments
		response.SuccessWithMsg[any](w, "删除成功")
		return
	}
	notFound(w)
}

func (app *mockServer) publishPost(w http.ResponseWriter, r *http.Request) {
	app.changePostStatus(w, r, model.PostStatusDraft, model.PostStatusPublished)
}

func (app *mockServer) withdrawPost(w http.ResponseWriter, r *http.Request) {
	app.changePostStatus(w, r, model.PostStatusPublished, model.PostStatusDraft)
}

func (app *mockServer) changePostStatus(w http.ResponseWriter, r *http.Request, from, to model.PostStatus) {
	id, ok := pathID(w, r, "postID", "invalid post id")
	if !ok {
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	post, exists := app.store.postByID(id)
	if !exists {
		notFound(w)
		return
	}
	if !canManagePost(currentActor(r), post) {
		response.Error(w, http.StatusForbidden, "无权访问")
		return
	}
	if post.Status != from {
		conflict(w)
		return
	}
	post.Status = to
	now := nowUTC()
	post.UpdatedAt = now
	if to == model.PostStatusPublished {
		post.PublishedAt = &now
	} else {
		post.PublishedAt = nil
	}
	response.Success(w, dto.NewPostResponse(post))
}

func (app *mockServer) listComments(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := parsePage(r, 50)
	if !ok {
		badRequest(w)
		return
	}
	actor := currentActor(r)
	app.store.mu.RLock()
	items := make([]model.ManagedComment, 0, len(app.store.comments))
	for _, comment := range app.store.comments {
		post, postExists := app.store.postByID(comment.PostID)
		if !postExists || (actor.Role == model.RoleContributor && post.AuthorID != actor.ID) {
			continue
		}
		items = append(items, model.ManagedComment{Comment: comment, PostSlug: post.Slug, PostTitle: post.Title, PostAuthorID: post.AuthorID})
	}
	app.store.mu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	page, cursor := paginate(items, limit, offset)
	response.Success(w, dto.NewManagedCommentPageResponse(page, cursor))
}

func (app *mockServer) hideComment(w http.ResponseWriter, r *http.Request) {
	app.changeCommentStatus(w, r, model.CommentStatusVisible, model.CommentStatusHidden)
}

func (app *mockServer) showComment(w http.ResponseWriter, r *http.Request) {
	app.changeCommentStatus(w, r, model.CommentStatusHidden, model.CommentStatusVisible)
}

func (app *mockServer) deleteComment(w http.ResponseWriter, r *http.Request) {
	app.changeCommentStatus(w, r, "", model.CommentStatusDeleted)
}

func (app *mockServer) changeCommentStatus(w http.ResponseWriter, r *http.Request, from, to model.CommentStatus) {
	id, ok := pathID(w, r, "commentID", "评论 ID 错误")
	if !ok {
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	comment, exists := app.store.commentByID(id)
	if !exists {
		notFound(w)
		return
	}
	post, postExists := app.store.postByID(comment.PostID)
	if !postExists || !canManagePost(currentActor(r), post) {
		response.Error(w, http.StatusForbidden, "无权访问")
		return
	}
	if (from != "" && comment.Status != from) || comment.Status == model.CommentStatusDeleted {
		conflict(w)
		return
	}
	now := nowUTC()
	comment.Status = to
	comment.ModeratedAt = &now
	response.Success(w, dto.NewCommentModerationResponse(comment))
}

func canManagePost(actor *model.User, post *model.Post) bool {
	return actor != nil && (actor.Role == model.RoleAdmin || (actor.Role == model.RoleContributor && post.AuthorID == actor.ID))
}

func normalizeTags(tags []string) ([]string, bool) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if len([]rune(tag)) > 30 {
			return nil, false
		}
		if _, duplicate := seen[tag]; duplicate {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result, len(result) <= 10
}

func firstMockImageURL(content string) *string {
	match := firstImagePattern.FindStringSubmatch(content)
	if len(match) < 2 || !validHTTPURL(match[1], false) {
		return nil
	}
	value := match[1]
	return &value
}
