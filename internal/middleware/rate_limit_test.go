package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
)

func TestIPRateLimiterEnforcesAndResetsWindow(t *testing.T) {
	limiter := NewIPRateLimiter(2, time.Minute)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	limiter.lastCleanup = now
	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response.Success(w, "ok")
	}))

	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.RemoteAddr = "203.0.113.10:1234"
		handler.ServeHTTP(recorder, r)
		return recorder
	}
	assertBusinessCode(t, request(), http.StatusOK)
	assertBusinessCode(t, request(), http.StatusOK)
	limited := request()
	assertBusinessCode(t, limited, http.StatusTooManyRequests)
	if limited.Header().Get("Retry-After") == "" {
		t.Fatal("rate-limited response has no Retry-After")
	}

	now = now.Add(time.Minute)
	assertBusinessCode(t, request(), http.StatusOK)
}

func TestClientIPStripsPort(t *testing.T) {
	if got := clientIP("[2001:db8::1]:443"); got != "2001:db8::1" {
		t.Fatalf("clientIP() = %q", got)
	}
}
