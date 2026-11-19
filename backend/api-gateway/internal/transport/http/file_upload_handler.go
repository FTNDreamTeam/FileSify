package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/FTNDreamTeam/FileSify/backend/api-gateway/internal/service"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type FileUploadHandler struct {
	uploadService *service.FileUploadService
}

func NewFileUploadHandler(uploadService *service.FileUploadService) *FileUploadHandler {
	return &FileUploadHandler{
		uploadService: uploadService,
	}
}

func (h *FileUploadHandler) RegisterRoutes(mux *runtime.ServeMux) error {
	return mux.HandlePath("POST", "/api/storage/file", h.handleUpload)
}

func (h *FileUploadHandler) handleUpload(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		http.Error(w, "Missing authorization token", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Invalid multipart form data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing field 'file' in request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	err = h.uploadService.Upload(r.Context(), service.UploadInput{
		Token:       token,
		Filename:    header.Filename,
		MimeType:    mimeType,
		FolderPath:  r.FormValue("path"),
		FileSize:    header.Size,
		FileContent: file,
	})

	if err != nil {
		h.writeError(w, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"uploaded"}`))
}

func (h *FileUploadHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, service.ErrFileTooLarge), errors.Is(err, service.ErrQuotaExceeded):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrUpstreamFailed):
		http.Error(w, err.Error(), http.StatusBadGateway)
	default:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
