package dto

import (
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

type CreateUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type UploadAuthorizationResponse struct {
	Method    string            `json:"method"`
	ObjectKey string            `json:"object_key"`
	UploadURL string            `json:"upload_url"`
	Fields    map[string]string `json:"fields"`
	PublicURL string            `json:"public_url"`
	ExpiresAt time.Time         `json:"expires_at"`
}

func NewUploadAuthorizationResponse(upload *service.UploadAuthorization) UploadAuthorizationResponse {
	return UploadAuthorizationResponse{
		Method: upload.Method, ObjectKey: upload.ObjectKey, UploadURL: upload.UploadURL,
		Fields: upload.Fields, PublicURL: upload.PublicURL, ExpiresAt: upload.ExpiresAt,
	}
}
