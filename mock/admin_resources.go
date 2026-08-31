package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/google/uuid"
)

func (app *mockServer) listUsers(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	users := append([]model.User(nil), app.store.users...)
	app.store.mu.RUnlock()
	response.Success(w, dto.NewUserResponses(users))
}

func (app *mockServer) getUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "用户 ID 错误")
	if !ok {
		return
	}
	app.store.mu.RLock()
	user, exists := app.store.userByID(id)
	if exists {
		copy := *user
		user = &copy
	}
	app.store.mu.RUnlock()
	if !exists {
		notFound(w)
		return
	}
	response.Success(w, dto.NewUserResponse(user))
}

func (app *mockServer) createUser(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateUserRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	request.Nickname = strings.TrimSpace(request.Nickname)
	request.AvatarURL = strings.TrimSpace(request.AvatarURL)
	if !validNonEmpty(request.Username, 64) || len(request.Username) < 3 || !validNonEmpty(request.Nickname, 80) || len(request.Password) < 10 || len(request.Password) > 72 || !request.Role.Valid() {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	if _, exists := app.store.userByUsername(request.Username); exists {
		conflict(w)
		return
	}
	now := nowUTC()
	user := model.User{ID: uuid.New(), Username: request.Username, Nickname: request.Nickname, AvatarURL: request.AvatarURL, Role: request.Role, CreatedAt: now, UpdatedAt: now}
	app.store.users = append(app.store.users, user)
	app.store.passwords[user.ID] = request.Password
	response.Success(w, dto.NewUserResponse(&user))
}

func (app *mockServer) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "用户 ID 错误")
	if !ok {
		return
	}
	var request dto.UpdateUserRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Username == nil && request.Nickname == nil && request.AvatarURL == nil && request.Role == nil {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	user, exists := app.store.userByID(id)
	if !exists {
		notFound(w)
		return
	}
	if request.Username != nil {
		value := strings.TrimSpace(*request.Username)
		if len(value) < 3 || !validNonEmpty(value, 64) {
			badRequest(w)
			return
		}
		if other, duplicate := app.store.userByUsername(value); duplicate && other.ID != id {
			conflict(w)
			return
		}
		user.Username = value
	}
	if request.Nickname != nil {
		value := strings.TrimSpace(*request.Nickname)
		if !validNonEmpty(value, 80) {
			badRequest(w)
			return
		}
		user.Nickname = value
	}
	if request.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	if request.Role != nil {
		if !request.Role.Valid() {
			badRequest(w)
			return
		}
		if user.Role == model.RoleAdmin && *request.Role != model.RoleAdmin && app.adminCount() == 1 {
			response.Error(w, http.StatusConflict, "需要至少存在一名管理员")
			return
		}
		user.Role = *request.Role
	}
	user.UpdatedAt = nowUTC()
	response.Success(w, dto.NewUserResponse(user))
}

func (app *mockServer) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "用户 ID 错误")
	if !ok {
		return
	}
	if currentActor(r).ID == id {
		response.Error(w, http.StatusForbidden, "无权访问")
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	for index := range app.store.users {
		if app.store.users[index].ID != id {
			continue
		}
		if app.store.users[index].Role == model.RoleAdmin && app.adminCount() == 1 {
			response.Error(w, http.StatusConflict, "需要至少存在一名管理员")
			return
		}
		app.store.users = removeAt(app.store.users, index)
		delete(app.store.passwords, id)
		response.SuccessWithMsg[any](w, "删除成功")
		return
	}
	notFound(w)
}

func (app *mockServer) updateUserPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "userID", "用户 ID 错误")
	if !ok {
		return
	}
	var request dto.UpdatePasswordRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.Password) < 10 || len(request.Password) > 72 {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	if _, exists := app.store.userByID(id); !exists {
		notFound(w)
		return
	}
	app.store.passwords[id] = request.Password
	response.SuccessWithMsg[any](w, "密码更改成功")
}

func (app *mockServer) adminCount() int {
	count := 0
	for _, user := range app.store.users {
		if user.Role == model.RoleAdmin {
			count++
		}
	}
	return count
}

func (app *mockServer) listCategories(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	categories := append([]model.Category(nil), app.store.categories...)
	app.store.mu.RUnlock()
	response.Success(w, dto.NewCategoryResponses(categories))
}

func (app *mockServer) getCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "categoryID", "分类 ID 错误")
	if !ok {
		return
	}
	app.store.mu.RLock()
	category, exists := app.store.categoryByID(id)
	if exists {
		copy := *category
		category = &copy
	}
	app.store.mu.RUnlock()
	if !exists {
		notFound(w)
		return
	}
	response.Success(w, dto.NewCategoryResponse(category))
}

func (app *mockServer) createCategory(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateCategoryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Name, request.Slug, request.Description = strings.TrimSpace(request.Name), strings.TrimSpace(request.Slug), strings.TrimSpace(request.Description)
	if !validNonEmpty(request.Name, 80) || !validSlug(request.Slug) {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	for _, category := range app.store.categories {
		if category.Slug == request.Slug || category.Name == request.Name {
			conflict(w)
			return
		}
	}
	now := nowUTC()
	category := model.Category{ID: uuid.New(), Name: request.Name, Slug: request.Slug, Description: request.Description, CreatedAt: now, UpdatedAt: now}
	app.store.categories = append(app.store.categories, category)
	response.Success(w, dto.NewCategoryResponse(&category))
}

func (app *mockServer) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "categoryID", "分类 ID 错误")
	if !ok {
		return
	}
	var request dto.UpdateCategoryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Name == nil && request.Slug == nil && request.Description == nil {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	category, exists := app.store.categoryByID(id)
	if !exists {
		notFound(w)
		return
	}
	if request.Name != nil {
		value := strings.TrimSpace(*request.Name)
		if !validNonEmpty(value, 80) {
			badRequest(w)
			return
		}
		category.Name = value
	}
	if request.Slug != nil {
		value := strings.TrimSpace(*request.Slug)
		if !validSlug(value) {
			badRequest(w)
			return
		}
		category.Slug = value
	}
	if request.Description != nil {
		category.Description = strings.TrimSpace(*request.Description)
	}
	category.UpdatedAt = nowUTC()
	response.Success(w, dto.NewCategoryResponse(category))
}

func (app *mockServer) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "categoryID", "分类 ID 错误")
	if !ok {
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	for _, post := range app.store.posts {
		if post.CategoryID == id {
			conflict(w)
			return
		}
	}
	for index := range app.store.categories {
		if app.store.categories[index].ID == id {
			app.store.categories = removeAt(app.store.categories, index)
			response.SuccessWithMsg[any](w, "删除成功")
			return
		}
	}
	notFound(w)
}

func (app *mockServer) listLinks(w http.ResponseWriter, _ *http.Request) {
	app.store.mu.RLock()
	links := append([]model.FriendLink(nil), app.store.links...)
	app.store.mu.RUnlock()
	response.Success(w, dto.NewLinkResponses(links))
}

func (app *mockServer) getLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "linkID", "友链 ID 错误")
	if !ok {
		return
	}
	app.store.mu.RLock()
	link, exists := app.store.linkByID(id)
	if exists {
		copy := *link
		link = &copy
	}
	app.store.mu.RUnlock()
	if !exists {
		notFound(w)
		return
	}
	response.Success(w, dto.NewLinkResponse(link))
}

func (app *mockServer) createLink(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateLinkRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validLinkInput(request.Name, request.URL, request.FaviconURL) {
		badRequest(w)
		return
	}
	now := nowUTC()
	link := model.FriendLink{ID: uuid.New(), Name: strings.TrimSpace(request.Name), Description: strings.TrimSpace(request.Description), URL: strings.TrimSpace(request.URL), FaviconURL: strings.TrimSpace(request.FaviconURL), CreatedAt: now, UpdatedAt: now}
	app.store.mu.Lock()
	app.store.links = append(app.store.links, link)
	app.store.mu.Unlock()
	response.Success(w, dto.NewLinkResponse(&link))
}

func (app *mockServer) updateLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "linkID", "友链 ID 错误")
	if !ok {
		return
	}
	var request dto.UpdateLinkRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.Name == nil && request.Description == nil && request.URL == nil && request.FaviconURL == nil {
		badRequest(w)
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	link, exists := app.store.linkByID(id)
	if !exists {
		notFound(w)
		return
	}
	name, targetURL, faviconURL := link.Name, link.URL, link.FaviconURL
	if request.Name != nil {
		name = *request.Name
	}
	if request.URL != nil {
		targetURL = *request.URL
	}
	if request.FaviconURL != nil {
		faviconURL = *request.FaviconURL
	}
	if !validLinkInput(name, targetURL, faviconURL) {
		badRequest(w)
		return
	}
	link.Name, link.URL, link.FaviconURL = strings.TrimSpace(name), strings.TrimSpace(targetURL), strings.TrimSpace(faviconURL)
	if request.Description != nil {
		link.Description = strings.TrimSpace(*request.Description)
	}
	link.UpdatedAt = nowUTC()
	response.Success(w, dto.NewLinkResponse(link))
}

func (app *mockServer) deleteLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "linkID", "友链 ID 错误")
	if !ok {
		return
	}
	app.store.mu.Lock()
	defer app.store.mu.Unlock()
	for index := range app.store.links {
		if app.store.links[index].ID == id {
			app.store.links = removeAt(app.store.links, index)
			response.SuccessWithMsg[any](w, "删除成功")
			return
		}
	}
	notFound(w)
}

func validLinkInput(name, targetURL, faviconURL string) bool {
	if !validNonEmpty(strings.TrimSpace(name), 80) || !validHTTPURL(targetURL, false) || !validHTTPURL(faviconURL, true) {
		return false
	}
	return true
}

func validHTTPURL(value string, allowEmpty bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return allowEmpty
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Host != "" && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}
