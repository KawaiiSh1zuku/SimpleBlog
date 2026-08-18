package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/utils"
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
		response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	response.Error(w, http.StatusBadRequest, "invalid request")
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.Error(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(w, http.StatusUnauthorized, "invalid username or password")
	case errors.Is(err, service.ErrNotFound):
		response.Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		response.Error(w, http.StatusConflict, "resource conflict")
	case errors.Is(err, service.ErrForbidden):
		response.Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrLastAdmin):
		response.Error(w, http.StatusConflict, "at least one admin must remain")
	case errors.Is(err, service.ErrUnavailable):
		response.Error(w, http.StatusServiceUnavailable, "temporarily unavailable")
	default:
		utils.LogError(r.Context(), "request failed", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
