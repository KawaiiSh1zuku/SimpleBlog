package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func TestS3SignerRealUploadAndPublicRead(t *testing.T) {
	if os.Getenv("SIMPLEBLOG_TEST_S3_LOCAL") != "1" {
		t.Skip("set SIMPLEBLOG_TEST_S3_LOCAL=1 to run against a disposable local S3-compatible server")
	}

	endpoint := environmentOr("SIMPLEBLOG_TEST_S3_ENDPOINT", "127.0.0.1:9000")
	region := environmentOr("SIMPLEBLOG_TEST_S3_REGION", "us-east-1")
	bucket := "simpleblog-m5-" + uuid.NewString()[:8]
	cfg := config.S3Config{
		Endpoint: endpoint, Region: region, Bucket: bucket,
		AccessKey: environmentOr("SIMPLEBLOG_TEST_S3_ACCESS_KEY", "minioadmin"),
		SecretKey: environmentOr("SIMPLEBLOG_TEST_S3_SECRET_KEY", "minioadmin"),
	}
	signer, err := NewS3Signer(cfg)
	if err != nil {
		t.Fatalf("NewS3Signer() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := signer.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
		t.Fatalf("MakeBucket() error = %v", err)
	}
	objectKey := "blog/uploads/20260822/pixel.gif"
	defer func() {
		_ = signer.client.RemoveObject(context.Background(), bucket, objectKey, minio.RemoveObjectOptions{})
		_ = signer.client.RemoveBucket(context.Background(), bucket)
	}()
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, bucket)
	if err := signer.client.SetBucketPolicy(ctx, bucket, policy); err != nil {
		t.Fatalf("SetBucketPolicy() error = %v", err)
	}

	content := []byte("GIF89a")
	uploadURL, fields, err := signer.PresignPost(ctx, objectKey, "image/gif", int64(len(content)), time.Now().Add(10*time.Minute))
	if err != nil {
		t.Fatalf("PresignPost() error = %v", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", "pixel.gif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: 10 * time.Second}
	result, err := client.Do(request)
	if err != nil {
		t.Fatalf("upload request error = %v", err)
	}
	_, _ = io.Copy(io.Discard, result.Body)
	_ = result.Body.Close()
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		t.Fatalf("upload status = %s", result.Status)
	}

	publicURL, err := url.JoinPath("http://"+endpoint, bucket, objectKey)
	if err != nil {
		t.Fatal(err)
	}
	publicResult, err := client.Get(publicURL)
	if err != nil {
		t.Fatalf("public GET error = %v", err)
	}
	got, readErr := io.ReadAll(publicResult.Body)
	_ = publicResult.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if publicResult.StatusCode != http.StatusOK || !bytes.Equal(got, content) {
		t.Fatalf("public GET status/content = %s / %q", publicResult.Status, got)
	}
}

func environmentOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
