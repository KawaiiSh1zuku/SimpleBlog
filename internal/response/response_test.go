package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccessUsesHTTP200AndBodyCode(t *testing.T) {
	recorder := httptest.NewRecorder()

	Success(recorder, map[string]string{"value": "ok"})

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var body Response[map[string]string]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() returned an error: %v", err)
	}
	if body.Code != http.StatusOK || body.Data["value"] != "ok" {
		t.Fatalf("unexpected response body: %+v", body)
	}
}

func TestErrorKeepsHTTP200AndUsesBusinessCode(t *testing.T) {
	recorder := httptest.NewRecorder()

	Error(recorder, http.StatusForbidden, "permission denied")

	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body Response[any]
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() returned an error: %v", err)
	}
	if body.Code != http.StatusForbidden || body.Msg != "permission denied" {
		t.Fatalf("unexpected response body: %+v", body)
	}
}
