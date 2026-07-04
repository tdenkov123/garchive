package validation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/validation"
)

func TestOwnerID(t *testing.T) {
	require.NoError(t, validation.OwnerID("user-1"))
	require.ErrorIs(t, validation.OwnerID(""), domain.ErrInvalidInput)
	require.ErrorIs(t, validation.OwnerID("../evil"), domain.ErrInvalidInput)
	require.ErrorIs(t, validation.OwnerID(strings.Repeat("a", 129)), domain.ErrInvalidInput)
}

func TestContentType(t *testing.T) {
	require.NoError(t, validation.ContentType("application/pdf"))
	require.ErrorIs(t, validation.ContentType("bad"), domain.ErrInvalidInput)
}

func TestChecksumSHA256(t *testing.T) {
	require.NoError(t, validation.ChecksumSHA256(""))
	require.NoError(t, validation.ChecksumSHA256(strings.Repeat("a", 64)))
	require.ErrorIs(t, validation.ChecksumSHA256("short"), domain.ErrInvalidInput)
}

func TestCreateUploadInput(t *testing.T) {
	err := validation.CreateUploadInput("user-1", "doc.pdf", "application/pdf", 1024, 2048)
	require.NoError(t, err)

	err = validation.CreateUploadInput("user-1", "doc.pdf", "application/pdf", 4096, 2048)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestFileSize(t *testing.T) {
	assert.NoError(t, validation.FileSize(100, 0))
	assert.ErrorIs(t, validation.FileSize(0, 100), domain.ErrInvalidInput)
}
