package router

import (
	"encoding/json"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyHeadersAreOnlyTrustedWhenConfigured(t *testing.T) {
	publicRoutes := newRouter()
	publicRoutes.Get("/ip", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, chimiddleware.GetClientIP(r.Context()))
	})
	for _, test := range []struct {
		name       string
		trustProxy bool
		want       string
	}{
		{name: "direct", trustProxy: false, want: "198.51.100.8"},
		{name: "trusted proxy", trustProxy: true, want: "203.0.113.7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := New(newRouter(), newRouter(), newRouter(), newRouter(), publicRoutes, test.trustProxy)
			request := httptest.NewRequest(http.MethodGet, "/ip", nil)
			request.RemoteAddr = "198.51.100.8:1234"
			request.Header.Set("X-Real-IP", "203.0.113.7")
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, request)
			if !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("response = %s, want IP %s", recorder.Body.String(), test.want)
			}
		})
	}
}

func TestFallbackResponsesUseHTTP200AndBusinessCode(t *testing.T) {
	serviceRoutes := newRouter()
	serviceRoutes.Get("/only-get", func(w http.ResponseWriter, _ *http.Request) {
		response.Success(w, "ok")
	})
	adminRoutes := newRouter()
	r := New(serviceRoutes, newRouter(), adminRoutes, newRouter(), newRouter(), true)

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
	r := New(serviceRoutes, chi.NewRouter(), chi.NewRouter(), chi.NewRouter(), chi.NewRouter(), true)
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
