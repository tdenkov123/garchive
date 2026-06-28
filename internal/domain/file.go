package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("file not found")
	ErrAccessDenied  = errors.New("access denied")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyExists = errors.New("file already confirmed")
)

type FileStatus string

const (
	FileStatusPending FileStatus = "pending"
	FileStatusReady   FileStatus = "ready"
	FileStatusDeleted FileStatus = "deleted"
)

type UploadMode string

const (
	UploadModeSingle    UploadMode = "single"
	UploadModeMultipart UploadMode = "multipart"
)

type FileMetadata struct {
	ID             string
	OwnerID        string
	Bucket         string
	ObjectKey      string
	OriginalName   string
	ContentType    string
	SizeBytes      int64
	ChecksumSHA256 string
	Status         FileStatus
	UploadMode     UploadMode
	UploadID       string
	PartSize       int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UploadPart struct {
	PartNumber int32
	ETag       string
	UploadedAt time.Time
}

type CompletedPart struct {
	PartNumber int32
	ETag       string
}

func (f *FileMetadata) TotalParts() int32 {
	if f.PartSize <= 0 || f.SizeBytes <= 0 {
		return 0
	}
	return int32((f.SizeBytes + f.PartSize - 1) / f.PartSize)
}

func (f *FileMetadata) IsMultipart() bool {
	return f.UploadMode == UploadModeMultipart
}

type ListFilter struct {
	OwnerID   string
	PageSize  int32
	PageToken string
}

type ListResult struct {
	Files         []FileMetadata
	NextPageToken string
}

func (f *FileMetadata) IsOwnedBy(ownerID string) bool {
	return f.OwnerID == ownerID
}

func (f *FileMetadata) CanDownload() bool {
	return f.Status == FileStatusReady
}
