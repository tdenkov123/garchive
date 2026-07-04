package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

func TestFileMetadata_IsOwnedBy(t *testing.T) {
	file := domain.FileMetadata{OwnerID: "user-1"}
	assert.True(t, file.IsOwnedBy("user-1"))
	assert.False(t, file.IsOwnedBy("user-2"))
}

func TestFileMetadata_CanDownload(t *testing.T) {
	pending := domain.FileMetadata{Status: domain.FileStatusPending}
	ready := domain.FileMetadata{Status: domain.FileStatusReady}
	deleted := domain.FileMetadata{Status: domain.FileStatusDeleted}
	assert.False(t, pending.CanDownload())
	assert.True(t, ready.CanDownload())
	assert.False(t, deleted.CanDownload())
}

func TestFileMetadata_TotalParts(t *testing.T) {
	file := domain.FileMetadata{SizeBytes: testPartSize + 1024, PartSize: testPartSize}
	assert.Equal(t, int32(2), file.TotalParts())
}

func TestFileMetadata_IsMultipart(t *testing.T) {
	multipart := domain.FileMetadata{UploadMode: domain.UploadModeMultipart}
	single := domain.FileMetadata{UploadMode: domain.UploadModeSingle}
	assert.True(t, multipart.IsMultipart())
	assert.False(t, single.IsMultipart())
}

const testPartSize = 5 * 1024 * 1024
