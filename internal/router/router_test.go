package router

import (
	"encoding/json"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFallbackResponsesUseHTTP200AndBusinessCode(t *testing.T) {
	serviceRoutes := newRouter()
	serviceRoutes.Get("/only-get", func(w http.ResponseWriter, _ *http.Request) {
		response.Success(w, "ok")
	})
	adminRoutes := newRouter()
	r := New(serviceRoutes, newRouter(), adminRoutes, true)

	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
	}{
		{name: "not found", method: http.MethodGet, path: "/missing", wantCode: http.StatusNotFound},
		{name: "nested not found", method: http.MethodGet, path: "/service/missing", wantCode: http.StatusNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/service/only-get", wantCode: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))

			if recorder.Code != http.StatusOK {
				t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
			}
			var body response.Response[any]
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("Decode() returned an error: %v", err)
			}
			if body.Code != test.wantCode {
				t.Fatalf("body code = %d, want %d", body.Code, test.wantCode)
			}
		})
	}
}

func TestRecovererUsesUnifiedResponse(t *testing.T) {
	serviceRoutes := newRouter()
	serviceRoutes.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	r := New(serviceRoutes, chi.NewRouter(), chi.NewRouter(), true)
	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service/panic", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var body response.Response[any]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() returned an error: %v", err)
	}
	if body.Code != http.StatusInternalServerError {
		t.Fatalf("body code = %d, want %d", body.Code, http.StatusInternalServerError)
	}
}
