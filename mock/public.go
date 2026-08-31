package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (app *mockServer) listPublicPosts(w http.ResponseWriter, r *http.Request) {
	app.publicPostPage(w, r, "")
}

func (app *mockServer) searchPosts(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" || utf8.RuneCountInString(query) > 100 || strings.ContainsAny(query, "\r\n") {
		badRequest(w)
		return
	}
	app.publicPostPage(w, r, strings.ToLower(query))
}

func (app *mockServer) publicPostPage(w http.ResponseWriter, r *http.Request, search string) {
	limit, offset, ok := parsePage(r, 20)
	if !ok {
		badRequest(w)
		return
	}
	categorySlug := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")

	app.store.mu.RLock()
	posts := make([]model.PublicPost, 0, len(app.store.posts))
	for index := range app.store.posts {
		post := &app.store.posts[index]
		if post.Status != model.PostStatusPublished {
			continue
		}
		category, categoryOK := app.store.categoryByID(post.CategoryID)
		author, authorOK := app.store.userByID(post.AuthorID)
		if !categoryOK || !authorOK || (categorySlug != "" && category.Slug != categorySlug) || (tag != "" && !contains(post.Tags, tag)) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(post.Title+"\n"+post.Summary+"\n"+post.Content), search) {
			continue
		}
		posts = append(posts, model.PublicPost{Post: *post, AuthorNickname: author.Nickname, AuthorAvatarURL: author.AvatarURL, CategoryName: category.Name, CategorySlug: category.Slug})
	}
	app.store.mu.RUnlock()
	sort.SliceStable(posts, func(i, j int) bool {
		return posts[i].PublishedAt.After(*posts[j].PublishedAt)
	})
	page, cursor := paginate(posts, limit, offset)
	response.Success(w, dto.NewPublicPostPageResponse(page, cursor))
}

func (app *mockServer) getPublicPost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	app.store.mu.RLock()
	post, ok := app.store.postBySlug(slug)
	if !ok || post.Status != model.PostStatusPublished {
		app.store.mu.RUnlock()
		notFound(w)
		return
	}
	category, categoryOK := app.store.categoryByID(post.CategoryID)
	author, authorOK := app.store.userByID(post.AuthorID)
	if !categoryOK || !authorOK {
		app.store.mu.RUnlock()
		notFound(w)
		return
	}
	publicPost := model.PublicPost{Post: *post, AuthorNickname: author.Nickname, AuthorAvatarURL: author.AvatarURL, CategoryName: category.Name, CategorySlug: category.Slug}
	app.store.mu.RUnlock()
	response.Success(w, dto.NewPublicPostDetail(&publicPost))
}

func (app *mockServer) listPublicComments(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	app.store.mu.RLock()
	post, ok := app.store.postBySlug(slug)
	if !ok || post.Status != model.PostStatusPublished {
		app.store.mu.RUnlock()
		notFound(w)
		return
	}
	comments := make([]model.Comment, 0)
	for _, comment := range app.store.comments {
		if comment.PostID == post.ID {
			comments = append(comments, comment)
		}
	}
	app.store.mu.RUnlock()
	sort.SliceStable(comments, func(i, j int) bool { return comments[i].CreatedAt.Before(comments[j].CreatedAt) })
	response.Success(w, dto.NewPublicCommentTree(comments, "https://www.gravatar.com/avatar"))
}

func (app *mockServer) createPublicComment(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateCommentRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Nickname = strings.TrimSpace(request.Nickname)
	request.Email = strings.TrimSpace(request.Email)
	request.Content = strings.TrimSpace(strings.ReplaceAll(request.Content, "\r\n", "\n"))
	if !validNonEmpty(request.Nickname, 80) || !validNonEmpty(request.Email, 254) || !strings.Contains(request.Email, "@") || !validNonEmpty(request.Content, 10000) || strings.ContainsAny(request.Nickname+request.Content, "<>") {
		badRequest(w)
		return
	}

	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	post, ok := app.store.postBySlug(chi.URLParam(r, "slug"))
	if !ok || post.Status != model.PostStatusPublished {
		notFound(w)
		return
	}
	if request.ParentID != nil {
		parent, exists := app.store.commentByID(*request.ParentID)
		if !exists || parent.PostID != post.ID {
			badRequest(w)
			return
		}
	}
	comment := model.Comment{ID: uuid.New(), PostID: post.ID, ParentID: request.ParentID, Nickname: request.Nickname, Email: request.Email, Content: request.Content, Status: model.CommentStatusVisible, CreatedAt: nowUTC()}
	app.store.comments = append(app.store.comments, comment)
	response.Success(w, dto.NewPublicCommentResponse(&comment, "https://www.gravatar.com/avatar"))
}

func (app *mockServer) listPublicCategories(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	categories := append([]model.Category(nil), app.store.categories...)
	app.store.mu.RUnlock()
	response.Success(w, dto.NewPublicCategoryResponses(categories))
}

func (app *mockServer) listPublicTags(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	seen := make(map[string]struct{})
	for _, post := range app.store.posts {
		if post.Status == model.PostStatusPublished {
			for _, tag := range post.Tags {
				seen[tag] = struct{}{}
			}
		}
	}
	app.store.mu.RUnlock()
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	response.Success(w, tags)
}

func (app *mockServer) listPublicLinks(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	links := append([]model.FriendLink(nil), app.store.links...)
	app.store.mu.RUnlock()
	response.Success(w, dto.NewLinkResponses(links))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mockAvatar(email string) string {
	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?d=identicon", hash)
}
