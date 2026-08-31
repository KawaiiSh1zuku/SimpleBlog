package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccessKeepsEmptyArrayData(t *testing.T) {
	recorder := httptest.NewRecorder()
	Success(recorder, []string{})

	var body map[string]json.RawMessage
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := string(body["data"]); got != "[]" {
		t.Fatalf("data = %s, want []", got)
	}
}

func TestResponsesWithoutDataUseExplicitNull(t *testing.T) {
	for _, test := range []struct {
		name  string
		write func(http.ResponseWriter)
	}{
		{name: "success message", write: func(w http.ResponseWriter) { SuccessWithMsg[any](w, "deleted") }},
		{name: "error", write: func(w http.ResponseWriter) { Error(w, http.StatusNotFound, "not found") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.write(recorder)
			var body map[string]json.RawMessage
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got := string(body["data"]); got != "null" {
				t.Fatalf("data = %s, want null", got)
			}
		})
	}
}
