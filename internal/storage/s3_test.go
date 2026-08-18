package storage

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/config"
)

func TestS3SignerPresignPostIncludesStrictPolicy(t *testing.T) {
	signer, err := NewS3Signer(config.S3Config{
		Endpoint: "s3.example.com", Region: "us-east-1", Bucket: "simpleblog",
		AccessKey: "access-key", SecretKey: "secret-key", UseSSL: true,
	})
	if err != nil {
		t.Fatalf("NewS3Signer() error = %v", err)
	}
	uploadURL, fields, err := signer.PresignPost(
		context.Background(), "posts/user/object.webp", "image/webp", 1234,
		time.Now().UTC().Add(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("PresignPost() error = %v", err)
	}
	if uploadURL != "https://s3.example.com/simpleblog/" {
		t.Fatalf("upload URL = %q", uploadURL)
	}
	if fields["key"] != "posts/user/object.webp" || fields["Content-Type"] != "image/webp" {
		t.Fatalf("unexpected signed fields: %#v", fields)
	}
	policyJSON, err := base64.StdEncoding.DecodeString(fields["policy"])
	if err != nil {
		t.Fatalf("decode policy: %v", err)
	}
	policy := string(policyJSON)
	for _, expected := range []string{
		`["content-length-range", 1234, 1234]`,
		`["eq","$Content-Type","image/webp"]`,
		`["eq","$bucket","simpleblog"]`,
		`["eq","$key","posts/user/object.webp"]`,
	} {
		if !strings.Contains(policy, expected) {
			t.Errorf("policy %s does not contain %s", policy, expected)
		}
	}
}
