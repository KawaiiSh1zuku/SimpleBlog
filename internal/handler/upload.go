package handler

import (
	"net/http"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/dto"
	appmiddleware "github.com/KawaiiSh1zuku/SimpleBlog/internal/middleware"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/response"
	"github.com/KawaiiSh1zuku/SimpleBlog/internal/service"
)

type UploadHandler struct {
	Uploads *service.UploadService
}

func NewUploadHandler(uploads *service.UploadService) *UploadHandler {
	return &UploadHandler{Uploads: uploads}
}

func (h *UploadHandler) Presign(w http.ResponseWriter, r *http.Request) {
	var request dto.CreateUploadRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	actor, _ := appmiddleware.CurrentUser(r.Context())
	upload, err := h.Uploads.Create(r.Context(), actor, service.CreateUploadInput{
		Filename: request.Filename, ContentType: request.ContentType, Size: request.Size,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	response.Success(w, dto.NewUploadAuthorizationResponse(upload))
}
