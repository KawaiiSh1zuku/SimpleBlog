package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
)

func (app *mockServer) login(w http.ResponseWriter, r *http.Request) {
	var request dto.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Username = strings.TrimSpace(request.Username)
	app.store.mu.RLock()
	user, ok := app.store.userByUsername(request.Username)
	if ok {
		ok = app.store.passwords[user.ID] == request.Password
	}
	var userID string
	if ok {
		userID = user.ID.String()
	}
	app.store.mu.RUnlock()
	if !ok {
		response.Error(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}
	token := "mock-user-" + userID
	if request.Username == "admin" {
		token = adminToken
	} else if request.Username == "writer" {
		token = contributorToken
	}
	response.Success(w, dto.LoginResponse{Token: token, ExpiresAt: nowUTC().Add(24 * time.Hour)})
}

func (app *mockServer) me(w http.ResponseWriter, r *http.Request) {
	response.Success(w, dto.NewUserResponse(currentActor(r)))
}
