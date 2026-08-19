package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

func TestWriteServiceErrorUsesHTTP200AndBusinessCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{name: "invalid", err: service.ErrInvalidInput, code: http.StatusBadRequest},
		{name: "forbidden", err: service.ErrForbidden, code: http.StatusForbidden},
		{name: "not found", err: service.ErrNotFound, code: http.StatusNotFound},
		{name: "conflict", err: service.ErrConflict, code: http.StatusConflict},
		{name: "unavailable", err: service.ErrUnavailable, code: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			writeServiceError(recorder, request, test.err)
			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d", recorder.Code)
			}
			var body response.Response[any]
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Code != test.code {
				t.Fatalf("business code = %d, want %d", body.Code, test.code)
			}
		})
	}
}

func TestWriteServiceErrorLogsFullDiagnostic(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	diagnostic := "database SimpleBlog does not exist"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	writeServiceError(recorder, request, errors.New(diagnostic))

	if !strings.Contains(logs.String(), diagnostic) || !strings.Contains(logs.String(), `"error"`) {
		t.Fatalf("full error diagnostic missing: %s", logs.String())
	}
}

func TestDecodeJSONMapsMaxBytesErrorTo413(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"value":"too large"}`)))
	recorder := httptest.NewRecorder()
	request.Body = http.MaxBytesReader(recorder, request.Body, 4)
	var target map[string]any
	if decodeJSON(recorder, request, &target) {
		t.Fatal("decodeJSON() accepted an oversized body")
	}
	var body response.Response[any]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("business code = %d, want %d", body.Code, http.StatusRequestEntityTooLarge)
	}
}
