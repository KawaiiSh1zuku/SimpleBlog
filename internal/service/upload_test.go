package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

type uploadSignerStub struct {
	objectKey   string
	contentType string
	size        int64
	expiresAt   time.Time
	err         error
}

func (s *uploadSignerStub) PresignPost(
	_ context.Context,
	objectKey string,
	contentType string,
	size int64,
	expiresAt time.Time,
) (string, map[string]string, error) {
	s.objectKey, s.contentType, s.size, s.expiresAt = objectKey, contentType, size, expiresAt
	return "https://s3.example.com/simpleblog", map[string]string{"policy": "signed"}, s.err
}

func TestUploadServiceCreatesServerOwnedPostPolicy(t *testing.T) {
	actor := &model.User{ID: uuid.New(), Role: model.RoleContributor}
	signer := &uploadSignerStub{}
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	service := NewUploadService(signer, "https://cdn.example.com/assets/")
	service.Now = func() time.Time { return now }

	upload, err := service.Create(context.Background(), actor, CreateUploadInput{
		Filename: "../../not-used.webp", ContentType: "image/webp", Size: 1234,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	keyPattern := regexp.MustCompile(`^blog/uploads/20260818/[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\.webp$`)
	if !keyPattern.MatchString(upload.ObjectKey) {
		t.Fatalf("object key %q is not server-owned", upload.ObjectKey)
	}
	if strings.Contains(upload.ObjectKey, "not-used") || signer.objectKey != upload.ObjectKey {
		t.Fatalf("client filename leaked into object key %q", upload.ObjectKey)
	}
	if signer.contentType != "image/webp" || signer.size != 1234 {
		t.Fatalf("signer constraints = %q, %d", signer.contentType, signer.size)
	}
	if want := now.Add(UploadURLLifetime); !signer.expiresAt.Equal(want) || !upload.ExpiresAt.Equal(want) {
		t.Fatalf("expiration = %v / %v, want %v", signer.expiresAt, upload.ExpiresAt, want)
	}
	if upload.Method != "POST" || upload.PublicURL != "https://cdn.example.com/assets/"+upload.ObjectKey {
		t.Fatalf("unexpected upload response: %+v", upload)
	}
}

func TestUploadServiceUsesLocalDateForObjectKey(t *testing.T) {
	signer := &uploadSignerStub{}
	location := time.FixedZone("UTC+8", 8*60*60)
	service := NewUploadService(signer, "https://cdn.example.com")
	service.Now = func() time.Time { return time.Date(2026, 8, 19, 0, 30, 0, 0, location) }

	upload, err := service.Create(context.Background(), &model.User{ID: uuid.New(), Role: model.RoleAdmin}, CreateUploadInput{
		Filename: "image.png", ContentType: "image/png", Size: 1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !strings.HasPrefix(upload.ObjectKey, "blog/uploads/20260819/") {
		t.Fatalf("object key uses wrong calendar date: %q", upload.ObjectKey)
	}
}

func TestUploadServiceRejectsInvalidMetadataAndRole(t *testing.T) {
	actor := &model.User{ID: uuid.New(), Role: model.RoleAdmin}
	tests := []CreateUploadInput{
		{Filename: "", ContentType: "image/png", Size: 1},
		{Filename: "file.svg", ContentType: "image/svg+xml", Size: 1},
		{Filename: "file.png", ContentType: "image/png", Size: 0},
		{Filename: "file.png", ContentType: "image/png", Size: MaxUploadSize + 1},
	}
	for _, input := range tests {
		if _, err := NewUploadService(&uploadSignerStub{}, "https://cdn.example.com").Create(context.Background(), actor, input); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidInput", input, err)
		}
	}
	if _, err := NewUploadService(&uploadSignerStub{}, "https://cdn.example.com").Create(context.Background(), nil, CreateUploadInput{
		Filename: "file.png", ContentType: "image/png", Size: 1,
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Create(nil actor) error = %v, want ErrForbidden", err)
	}
}

func TestUploadServicePropagatesSignerError(t *testing.T) {
	want := errors.New("storage unavailable")
	signer := &uploadSignerStub{err: want}
	_, err := NewUploadService(signer, "https://cdn.example.com").Create(context.Background(), &model.User{
		ID: uuid.New(), Role: model.RoleAdmin,
	}, CreateUploadInput{Filename: "file.jpg", ContentType: "image/jpeg", Size: 1})
	if !errors.Is(err, want) {
		t.Fatalf("Create() error = %v, want %v", err, want)
	}
}
