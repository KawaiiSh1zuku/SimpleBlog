package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
)

func TestLimitRequestBodyReturnsUnified413(t *testing.T) {
	handler := LimitRequestBody(4)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("oversized request reached handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("12345"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assertBusinessCode(t, recorder, http.StatusRequestEntityTooLarge)
}

func TestAccessLogRecordsBusinessCodeWithoutSensitiveInputs(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	handler := AccessLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, http.StatusBadRequest, "invalid request")
	}))
	request := httptest.NewRequest(http.MethodPost, "/comments?email=visitor@example.com", strings.NewReader("secret-body"))
	request.Header.Set("Authorization", "Bearer secret-token")
	request.RemoteAddr = "203.0.113.9"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("decode access log: %v\n%s", err, logs.String())
	}
	if entry["business_code"] != float64(http.StatusBadRequest) || entry["path"] != "/comments" {
		t.Fatalf("unexpected access log: %#v", entry)
	}
	for _, secret := range []string{"visitor@example.com", "secret-token", "secret-body"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("access log leaked %q: %s", secret, logs.String())
		}
	}
}

func TestAccessResponseWriterUsesRealHTTPStatusWhenNoBusinessCodeExists(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &accessResponseWriter{ResponseWriter: recorder, businessCode: http.StatusOK}
	writer.WriteHeader(http.StatusInternalServerError)
	if writer.businessCode != http.StatusInternalServerError || recorder.Code != http.StatusInternalServerError {
		t.Fatalf("business/http status = %d/%d", writer.businessCode, recorder.Code)
	}
	writer.SetBusinessCode(http.StatusBadRequest)
	writer.WriteHeader(http.StatusOK)
	if writer.businessCode != http.StatusBadRequest {
		t.Fatalf("explicit business code was overwritten: %d", writer.businessCode)
	}
}
