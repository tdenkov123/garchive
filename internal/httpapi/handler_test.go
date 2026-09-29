package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type fakeAPI struct {
	files    []domain.FileMetadata
	body     []byte
	listErr  error
	openErr  error
	delErr   error
	uploaded domain.FileMetadata
}

func (f *fakeAPI) ListFiles(context.Context, domain.ListFilter) (domain.ListResult, error) {
	if f.listErr != nil {
		return domain.ListResult{}, f.listErr
	}
	return domain.ListResult{Files: f.files}, nil
}

func (f *fakeAPI) UploadContent(_ context.Context, ownerID, originalName, contentType string, r io.Reader, size int64) (domain.FileMetadata, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	f.body = b
	f.uploaded = domain.FileMetadata{
		ID:           "file-1",
		OwnerID:      ownerID,
		OriginalName: originalName,
		ContentType:  contentType,
		SizeBytes:    size,
		Status:       domain.FileStatusReady,
		CreatedAt:    time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
	}
	return f.uploaded, nil
}

func (f *fakeAPI) OpenContent(context.Context, string, string) (domain.FileMetadata, io.ReadCloser, error) {
	if f.openErr != nil {
		return domain.FileMetadata{}, nil, f.openErr
	}
	return domain.FileMetadata{
		ID:           "file-1",
		OriginalName: "note.txt",
		ContentType:  "text/plain",
		SizeBytes:    int64(len(f.body)),
		Status:       domain.FileStatusReady,
	}, io.NopCloser(bytes.NewReader(f.body)), nil
}

func (f *fakeAPI) DeleteFile(context.Context, string, string) error {
	return f.delErr
}

func TestListFiles(t *testing.T) {
	api := &fakeAPI{files: []domain.FileMetadata{{
		ID:           "file-1",
		OwnerID:      "user-1",
		OriginalName: "note.txt",
		ContentType:  "text/plain",
		SizeBytes:    4,
		Status:       domain.FileStatusReady,
		CreatedAt:    time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
	}}}
	srv := httptest.NewServer(newHandler(api, nil, 1024))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/files?owner_id=user-1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload struct {
		Files []fileJSON `json:"files"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Len(t, payload.Files, 1)
	assert.Equal(t, "note.txt", payload.Files[0].OriginalName)
	assert.Equal(t, "ready", payload.Files[0].Status)
}

func TestListFiles_InvalidOwner(t *testing.T) {
	api := &fakeAPI{listErr: domain.ErrInvalidInput}
	srv := httptest.NewServer(newHandler(api, nil, 1024))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/files")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUploadAndDownload(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(newHandler(api, nil, 1024))
	t.Cleanup(srv.Close)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("owner_id", "user-1"))
	part, err := writer.CreateFormFile("file", "note.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/files", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "user-1", api.uploaded.OwnerID)
	assert.Equal(t, []byte("hello"), api.body)

	resp2, err := http.Get(srv.URL + "/api/files/file-1/content?owner_id=user-1")
	require.NoError(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Equal(t, "text/plain", resp2.Header.Get("Content-Type"))
	downloaded, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), downloaded)
}

func TestDownload_NotFound(t *testing.T) {
	api := &fakeAPI{openErr: domain.ErrNotFound}
	srv := httptest.NewServer(newHandler(api, nil, 1024))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/files/missing/content?owner_id=user-1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDelete_Forbidden(t *testing.T) {
	api := &fakeAPI{delErr: domain.ErrAccessDenied}
	srv := httptest.NewServer(newHandler(api, nil, 1024))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/files/file-1?owner_id=user-2", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}
