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

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
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
		response.Error(w, http.StatusConflict, "resource already exists")
	case errors.Is(err, service.ErrForbidden):
		response.Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrLastAdmin):
		response.Error(w, http.StatusConflict, "at least one admin must remain")
	default:
		slog.ErrorContext(r.Context(), "request failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
