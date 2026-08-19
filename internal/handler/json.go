package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeDecodeError(w, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
		return false
	}
	return true
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		response.Error(w, http.StatusRequestEntityTooLarge, "请求体大小异常")
		return
	}
	response.Error(w, http.StatusBadRequest, "错误请求")
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.Error(w, http.StatusBadRequest, "错误请求")
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "用户名或密码错误")
	case errors.Is(err, service.ErrNotFound):
		response.Error(w, http.StatusNotFound, "找不到资源")
	case errors.Is(err, service.ErrConflict):
		response.Error(w, http.StatusConflict, "资源冲突")
	case errors.Is(err, service.ErrForbidden):
		response.Error(w, http.StatusForbidden, "无权访问")
	case errors.Is(err, service.ErrLastAdmin):
		response.Error(w, http.StatusConflict, "需要至少存在一名管理员")
	case errors.Is(err, service.ErrUnavailable):
		response.Error(w, http.StatusServiceUnavailable, "服务暂时不可用")
	default:
		slog.ErrorContext(r.Context(), "request failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "内部服务错误")
	}
}
