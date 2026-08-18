package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/handler"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
)

type routeUploadSigner struct {
	called bool
}

func (signer *routeUploadSigner) PresignPost(
	context.Context,
	string,
	string,
	int64,
	time.Time,
) (string, map[string]string, error) {
	signer.called = true
	return "https://s3.example.com/simpleblog", map[string]string{"policy": "signed"}, nil
}

func TestUploadPresignRouteAllowsContributorAndRequiresAuthentication(t *testing.T) {
	contributor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	users := &routeUserStore{current: contributor}
	tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
	_, token, err := tokenAuth.Encode(map[string]any{"sub": contributor.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	signer := &routeUploadSigner{}
	uploadHandler := handler.NewUploadHandler(service.NewUploadService(signer, "https://cdn.example.com"))
	r := AdminRouter(nil, nil, nil, nil, uploadHandler, tokenAuth, users)

	body := []byte(`{"filename":"image.png","content_type":"image/png","size":128}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/uploads/presign", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(recorder, request)
	if code := decodeRouteBusinessCode(t, recorder); code != http.StatusOK || !signer.called {
		t.Fatalf("authenticated presign code = %d, signer called = %v", code, signer.called)
	}

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/uploads/presign", bytes.NewReader(body)))
	if code := decodeRouteBusinessCode(t, recorder); code != http.StatusUnauthorized {
		t.Fatalf("anonymous presign code = %d, want %d", code, http.StatusUnauthorized)
	}
}
