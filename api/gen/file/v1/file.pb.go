package filev1

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type FileStatus int32

const (
	FileStatus_FILE_STATUS_UNSPECIFIED FileStatus = 0
	FileStatus_FILE_STATUS_PENDING     FileStatus = 1
	FileStatus_FILE_STATUS_READY       FileStatus = 2
	FileStatus_FILE_STATUS_DELETED     FileStatus = 3
)

type FileMetadata struct {
	Id             string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId        string                 `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	Bucket         string                 `protobuf:"bytes,3,opt,name=bucket,proto3" json:"bucket,omitempty"`
	ObjectKey      string                 `protobuf:"bytes,4,opt,name=object_key,json=objectKey,proto3" json:"object_key,omitempty"`
	OriginalName   string                 `protobuf:"bytes,5,opt,name=original_name,json=originalName,proto3" json:"original_name,omitempty"`
	ContentType    string                 `protobuf:"bytes,6,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	SizeBytes      int64                  `protobuf:"varint,7,opt,name=size_bytes,json=sizeBytes,proto3" json:"size_bytes,omitempty"`
	ChecksumSha256 string                 `protobuf:"bytes,8,opt,name=checksum_sha256,json=checksumSha256,proto3" json:"checksum_sha256,omitempty"`
	Status         FileStatus             `protobuf:"varint,9,opt,name=status,proto3,enum=file.v1.FileStatus" json:"status,omitempty"`
	CreatedAt      *timestamppb.Timestamp `protobuf:"bytes,10,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt      *timestamppb.Timestamp `protobuf:"bytes,11,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
}

func (x *FileMetadata) Reset()         { *x = FileMetadata{} }
func (x *FileMetadata) String() string { return "" }
func (*FileMetadata) ProtoMessage()    {}

func (x *FileMetadata) GetId() string             { if x != nil { return x.Id }; return "" }
func (x *FileMetadata) GetOwnerId() string        { if x != nil { return x.OwnerId }; return "" }
func (x *FileMetadata) GetBucket() string         { if x != nil { return x.Bucket }; return "" }
func (x *FileMetadata) GetObjectKey() string        { if x != nil { return x.ObjectKey }; return "" }
func (x *FileMetadata) GetOriginalName() string   { if x != nil { return x.OriginalName }; return "" }
func (x *FileMetadata) GetContentType() string    { if x != nil { return x.ContentType }; return "" }
func (x *FileMetadata) GetSizeBytes() int64       { if x != nil { return x.SizeBytes }; return 0 }
func (x *FileMetadata) GetChecksumSha256() string { if x != nil { return x.ChecksumSha256 }; return "" }
func (x *FileMetadata) GetStatus() FileStatus     { if x != nil { return x.Status }; return FileStatus_FILE_STATUS_UNSPECIFIED }
func (x *FileMetadata) GetCreatedAt() *timestamppb.Timestamp { if x != nil { return x.CreatedAt }; return nil }
func (x *FileMetadata) GetUpdatedAt() *timestamppb.Timestamp { if x != nil { return x.UpdatedAt }; return nil }

type CreateUploadRequest struct {
	OwnerId      string `protobuf:"bytes,1,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	OriginalName string `protobuf:"bytes,2,opt,name=original_name,json=originalName,proto3" json:"original_name,omitempty"`
	ContentType  string `protobuf:"bytes,3,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	SizeBytes    int64  `protobuf:"varint,4,opt,name=size_bytes,json=sizeBytes,proto3" json:"size_bytes,omitempty"`
}

func (x *CreateUploadRequest) Reset()         { *x = CreateUploadRequest{} }
func (x *CreateUploadRequest) String() string { return "" }
func (*CreateUploadRequest) ProtoMessage()    {}

func (x *CreateUploadRequest) GetOwnerId() string      { if x != nil { return x.OwnerId }; return "" }
func (x *CreateUploadRequest) GetOriginalName() string { if x != nil { return x.OriginalName }; return "" }
func (x *CreateUploadRequest) GetContentType() string  { if x != nil { return x.ContentType }; return "" }
func (x *CreateUploadRequest) GetSizeBytes() int64     { if x != nil { return x.SizeBytes }; return 0 }

type CreateUploadResponse struct {
	Metadata         *FileMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	UploadUrl        string        `protobuf:"bytes,2,opt,name=upload_url,json=uploadUrl,proto3" json:"upload_url,omitempty"`
	ExpiresInSeconds int64         `protobuf:"varint,3,opt,name=expires_in_seconds,json=expiresInSeconds,proto3" json:"expires_in_seconds,omitempty"`
}

func (x *CreateUploadResponse) Reset()         { *x = CreateUploadResponse{} }
func (x *CreateUploadResponse) String() string { return "" }
func (*CreateUploadResponse) ProtoMessage()    {}

func (x *CreateUploadResponse) GetMetadata() *FileMetadata { if x != nil { return x.Metadata }; return nil }
func (x *CreateUploadResponse) GetUploadUrl() string       { if x != nil { return x.UploadUrl }; return "" }
func (x *CreateUploadResponse) GetExpiresInSeconds() int64 { if x != nil { return x.ExpiresInSeconds }; return 0 }

type ConfirmUploadRequest struct {
	Id             string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId        string `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	ChecksumSha256 string `protobuf:"bytes,3,opt,name=checksum_sha256,json=checksumSha256,proto3" json:"checksum_sha256,omitempty"`
}

func (x *ConfirmUploadRequest) Reset()         { *x = ConfirmUploadRequest{} }
func (x *ConfirmUploadRequest) String() string { return "" }
func (*ConfirmUploadRequest) ProtoMessage()    {}

func (x *ConfirmUploadRequest) GetId() string             { if x != nil { return x.Id }; return "" }
func (x *ConfirmUploadRequest) GetOwnerId() string        { if x != nil { return x.OwnerId }; return "" }
func (x *ConfirmUploadRequest) GetChecksumSha256() string { if x != nil { return x.ChecksumSha256 }; return "" }

type GetFileRequest struct {
	Id      string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId string `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
}

func (x *GetFileRequest) Reset()         { *x = GetFileRequest{} }
func (x *GetFileRequest) String() string { return "" }
func (*GetFileRequest) ProtoMessage()    {}

func (x *GetFileRequest) GetId() string      { if x != nil { return x.Id }; return "" }
func (x *GetFileRequest) GetOwnerId() string { if x != nil { return x.OwnerId }; return "" }

type ListFilesRequest struct {
	OwnerId   string `protobuf:"bytes,1,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	PageSize  int32  `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	PageToken string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
}

func (x *ListFilesRequest) Reset()         { *x = ListFilesRequest{} }
func (x *ListFilesRequest) String() string { return "" }
func (*ListFilesRequest) ProtoMessage()    {}

func (x *ListFilesRequest) GetOwnerId() string   { if x != nil { return x.OwnerId }; return "" }
func (x *ListFilesRequest) GetPageSize() int32   { if x != nil { return x.PageSize }; return 0 }
func (x *ListFilesRequest) GetPageToken() string { if x != nil { return x.PageToken }; return "" }

type ListFilesResponse struct {
	Files         []*FileMetadata `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
	NextPageToken string          `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

func (x *ListFilesResponse) Reset()         { *x = ListFilesResponse{} }
func (x *ListFilesResponse) String() string { return "" }
func (*ListFilesResponse) ProtoMessage()    {}

func (x *ListFilesResponse) GetFiles() []*FileMetadata { if x != nil { return x.Files }; return nil }
func (x *ListFilesResponse) GetNextPageToken() string  { if x != nil { return x.NextPageToken }; return "" }

type GetDownloadURLRequest struct {
	Id      string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId string `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
}

func (x *GetDownloadURLRequest) Reset()         { *x = GetDownloadURLRequest{} }
func (x *GetDownloadURLRequest) String() string { return "" }
func (*GetDownloadURLRequest) ProtoMessage()    {}

func (x *GetDownloadURLRequest) GetId() string      { if x != nil { return x.Id }; return "" }
func (x *GetDownloadURLRequest) GetOwnerId() string { if x != nil { return x.OwnerId }; return "" }

type GetDownloadURLResponse struct {
	DownloadUrl      string `protobuf:"bytes,1,opt,name=download_url,json=downloadUrl,proto3" json:"download_url,omitempty"`
	ExpiresInSeconds int64  `protobuf:"varint,2,opt,name=expires_in_seconds,json=expiresInSeconds,proto3" json:"expires_in_seconds,omitempty"`
}

func (x *GetDownloadURLResponse) Reset()         { *x = GetDownloadURLResponse{} }
func (x *GetDownloadURLResponse) String() string { return "" }
func (*GetDownloadURLResponse) ProtoMessage()    {}

func (x *GetDownloadURLResponse) GetDownloadUrl() string      { if x != nil { return x.DownloadUrl }; return "" }
func (x *GetDownloadURLResponse) GetExpiresInSeconds() int64  { if x != nil { return x.ExpiresInSeconds }; return 0 }

type DeleteFileRequest struct {
	Id      string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	OwnerId string `protobuf:"bytes,2,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
}

func (x *DeleteFileRequest) Reset()         { *x = DeleteFileRequest{} }
func (x *DeleteFileRequest) String() string { return "" }
func (*DeleteFileRequest) ProtoMessage()    {}

func (x *DeleteFileRequest) GetId() string      { if x != nil { return x.Id }; return "" }
func (x *DeleteFileRequest) GetOwnerId() string { if x != nil { return x.OwnerId }; return "" }

type DeleteFileResponse struct {
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

func (x *DeleteFileResponse) Reset()         { *x = DeleteFileResponse{} }
func (x *DeleteFileResponse) String() string { return "" }
func (*DeleteFileResponse) ProtoMessage()    {}

func (x *DeleteFileResponse) GetSuccess() bool { if x != nil { return x.Success }; return false }
