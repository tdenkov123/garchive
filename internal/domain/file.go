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
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
