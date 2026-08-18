package service

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/model"
	"github.com/google/uuid"
)

const (
	MaxUploadSize     int64 = 10 * 1024 * 1024
	UploadURLLifetime       = 10 * time.Minute
)

var uploadExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type UploadSigner interface {
	PresignPost(context.Context, string, string, int64, time.Time) (string, map[string]string, error)
}

type UploadService struct {
	Signer        UploadSigner
	PublicBaseURL string
	Now           func() time.Time
}

type CreateUploadInput struct {
	Filename    string
	ContentType string
	Size        int64
}

type UploadAuthorization struct {
	Method    string
	ObjectKey string
	UploadURL string
	Fields    map[string]string
	PublicURL string
	ExpiresAt time.Time
}

func NewUploadService(signer UploadSigner, publicBaseURL string) *UploadService {
	return &UploadService{Signer: signer, PublicBaseURL: strings.TrimRight(publicBaseURL, "/"), Now: time.Now}
}

func (s *UploadService) Create(
	ctx context.Context,
	actor *model.User,
	input CreateUploadInput,
) (*UploadAuthorization, error) {
	if !canManagePosts(actor) || actor.ID == uuid.Nil {
		return nil, ErrForbidden
	}
	filename := strings.TrimSpace(input.Filename)
	extension, allowed := uploadExtensions[input.ContentType]
	if filename == "" || utf8.RuneCountInString(filename) > 255 || !allowed || input.Size < 1 || input.Size > MaxUploadSize {
		return nil, ErrInvalidInput
	}

	objectKey := "posts/" + actor.ID.String() + "/" + uuid.NewString() + extension
	expiresAt := s.Now().UTC().Add(UploadURLLifetime)
	uploadURL, fields, err := s.Signer.PresignPost(ctx, objectKey, input.ContentType, input.Size, expiresAt)
	if err != nil {
		return nil, err
	}
	publicURL, err := url.JoinPath(s.PublicBaseURL, objectKey)
	if err != nil {
		return nil, err
	}
	return &UploadAuthorization{
		Method: "POST", ObjectKey: objectKey, UploadURL: uploadURL,
		Fields: fields, PublicURL: publicURL, ExpiresAt: expiresAt,
	}, nil
}
