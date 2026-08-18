package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Signer struct {
	client *minio.Client
	bucket string
}

func NewS3Signer(cfg config.S3Config) (*S3Signer, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	return &S3Signer{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3Signer) PresignPost(
	ctx context.Context,
	objectKey string,
	contentType string,
	size int64,
	expiresAt time.Time,
) (string, map[string]string, error) {
	policy := minio.NewPostPolicy()
	setters := []func() error{
		func() error { return policy.SetBucket(s.bucket) },
		func() error { return policy.SetKey(objectKey) },
		func() error { return policy.SetContentType(contentType) },
		func() error { return policy.SetContentLengthRange(size, size) },
		func() error { return policy.SetExpires(expiresAt) },
	}
	for _, set := range setters {
		if err := set(); err != nil {
			return "", nil, fmt.Errorf("build S3 upload policy: %w", err)
		}
	}

	uploadURL, fields, err := s.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return "", nil, fmt.Errorf("presign S3 upload: %w", err)
	}
	return uploadURL.String(), fields, nil
}
