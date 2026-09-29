package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

type fileAPI interface {
	ListFiles(ctx context.Context, filter domain.ListFilter) (domain.ListResult, error)
	UploadContent(ctx context.Context, ownerID, originalName, contentType string, r io.Reader, size int64) (domain.FileMetadata, error)
	OpenContent(ctx context.Context, id, ownerID string) (domain.FileMetadata, io.ReadCloser, error)
	DeleteFile(ctx context.Context, id, ownerID string) error
}

type Handler struct {
	api      fileAPI
	logger   *slog.Logger
	maxBytes int64
}

func NewHandler(svc *service.FileService, logger *slog.Logger, maxBytes int64) http.Handler {
	return newHandler(svc, logger, maxBytes)
}

func newHandler(api fileAPI, logger *slog.Logger, maxBytes int64) http.Handler {
	h := &Handler{api: api, logger: logger, maxBytes: maxBytes}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/files", h.list)
	mux.HandleFunc("POST /api/files", h.upload)
	mux.HandleFunc("GET /api/files/{id}/content", h.download)
	mux.HandleFunc("DELETE /api/files/{id}", h.delete)
	return mux
}

type fileJSON struct {
	ID           string `json:"id"`
	OwnerID      string `json:"owner_id"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

func toJSON(file domain.FileMetadata) fileJSON {
	return fileJSON{
		ID:           file.ID,
		OwnerID:      file.OwnerID,
		OriginalName: file.OriginalName,
		ContentType:  file.ContentType,
		SizeBytes:    file.SizeBytes,
		Status:       string(file.Status),
		CreatedAt:    file.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	result, err := h.api.ListFiles(r.Context(), domain.ListFilter{
		OwnerID:  r.URL.Query().Get("owner_id"),
		PageSize: 100,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	files := make([]fileJSON, 0, len(result.Files))
	for _, file := range result.Files {
		files = append(files, toJSON(file))
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	limit := h.maxBytes
	if limit <= 0 {
		limit = 512 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file exceeds max size"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart body"})
		return
	}

	ownerID := r.FormValue("owner_id")
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer func() { _ = file.Close() }()

	contentType := normalizeContentType(header.Header.Get("Content-Type"))
	meta, err := h.api.UploadContent(r.Context(), ownerID, header.Filename, contentType, file, header.Size)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toJSON(meta))
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	file, body, err := h.api.OpenContent(r.Context(), r.PathValue("id"), r.URL.Query().Get("owner_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	defer func() { _ = body.Close() }()

	contentType := file.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": file.OriginalName})
	if disposition == "" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition)
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil && h.logger != nil {
		h.logger.Error("write download", "error", err, "file_id", file.ID)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.api.DeleteFile(r.Context(), r.PathValue("id"), r.URL.Query().Get("owner_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrAccessDenied):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrAlreadyExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	default:
		if h.logger != nil {
			h.logger.Error("http api", "error", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func normalizeContentType(value string) string {
	value = strings.TrimSpace(value)
	if i := strings.Index(value, ";"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	if value == "" {
		return "application/octet-stream"
	}
	return value
}
