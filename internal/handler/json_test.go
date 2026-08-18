package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
