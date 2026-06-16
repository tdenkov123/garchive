package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	filev1 "github.com/tdenkov123/file-metadata-service/api/gen/file/v1"
	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

func Register(server *grpc.Server, svc *service.FileService, logger *slog.Logger) {
	filev1.RegisterFileServiceServer(server, &fileHandler{svc: svc, logger: logger})
}

type fileHandler struct {
	filev1.UnimplementedFileServiceServer
	svc    *service.FileService
	logger *slog.Logger
}

func (h *fileHandler) CreateUpload(ctx context.Context, req *filev1.CreateUploadRequest) (*filev1.CreateUploadResponse, error) {
	result, err := h.svc.CreateUpload(ctx, req.GetOwnerId(), req.GetOriginalName(), req.GetContentType(), req.GetSizeBytes())
	if err != nil {
		return nil, mapError(err)
	}
	return &filev1.CreateUploadResponse{
		Metadata:         toProtoFile(result.Metadata),
		UploadUrl:        result.UploadURL,
		ExpiresInSeconds: int64(result.ExpiresIn.Seconds()),
	}, nil
}

func (h *fileHandler) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*filev1.FileMetadata, error) {
	file, err := h.svc.ConfirmUpload(ctx, req.GetId(), req.GetOwnerId(), req.GetChecksumSha256())
	if err != nil {
		return nil, mapError(err)
	}
	return toProtoFile(file), nil
}

func (h *fileHandler) GetFile(ctx context.Context, req *filev1.GetFileRequest) (*filev1.FileMetadata, error) {
	file, err := h.svc.GetFile(ctx, req.GetId(), req.GetOwnerId())
	if err != nil {
		return nil, mapError(err)
	}
	return toProtoFile(file), nil
}

func (h *fileHandler) ListFiles(ctx context.Context, req *filev1.ListFilesRequest) (*filev1.ListFilesResponse, error) {
	result, err := h.svc.ListFiles(ctx, domain.ListFilter{
		OwnerID:   req.GetOwnerId(),
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	files := make([]*filev1.FileMetadata, 0, len(result.Files))
	for _, f := range result.Files {
		file := f
		files = append(files, toProtoFile(file))
	}

	return &filev1.ListFilesResponse{
		Files:         files,
		NextPageToken: result.NextPageToken,
	}, nil
}

func (h *fileHandler) GetDownloadURL(ctx context.Context, req *filev1.GetDownloadURLRequest) (*filev1.GetDownloadURLResponse, error) {
	result, err := h.svc.GetDownloadURL(ctx, req.GetId(), req.GetOwnerId())
	if err != nil {
		return nil, mapError(err)
	}
	return &filev1.GetDownloadURLResponse{
		DownloadUrl:      result.URL,
		ExpiresInSeconds: int64(result.ExpiresIn.Seconds()),
	}, nil
}

func (h *fileHandler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	if err := h.svc.DeleteFile(ctx, req.GetId(), req.GetOwnerId()); err != nil {
		return nil, mapError(err)
	}
	return &filev1.DeleteFileResponse{Success: true}, nil
}

func toProtoFile(file domain.FileMetadata) *filev1.FileMetadata {
	return &filev1.FileMetadata{
		Id:             file.ID,
		OwnerId:        file.OwnerID,
		Bucket:         file.Bucket,
		ObjectKey:      file.ObjectKey,
		OriginalName:   file.OriginalName,
		ContentType:    file.ContentType,
		SizeBytes:      file.SizeBytes,
		ChecksumSha256: file.ChecksumSHA256,
		Status:         toProtoStatus(file.Status),
		CreatedAt:      timestamppb.New(file.CreatedAt),
		UpdatedAt:      timestamppb.New(file.UpdatedAt),
	}
}

func toProtoStatus(status domain.FileStatus) filev1.FileStatus {
	switch status {
	case domain.FileStatusPending:
		return filev1.FileStatus_FILE_STATUS_PENDING
	case domain.FileStatusReady:
		return filev1.FileStatus_FILE_STATUS_READY
	case domain.FileStatusDeleted:
		return filev1.FileStatus_FILE_STATUS_DELETED
	default:
		return filev1.FileStatus_FILE_STATUS_UNSPECIFIED
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
