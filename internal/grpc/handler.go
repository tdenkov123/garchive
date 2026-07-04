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
	"github.com/tdenkov123/file-metadata-service/internal/audit"
	"github.com/tdenkov123/file-metadata-service/internal/auth"
	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

func Register(server *grpc.Server, svc *service.FileService, logger *slog.Logger, auditLog *audit.Logger) {
	filev1.RegisterFileServiceServer(server, &fileHandler{svc: svc, logger: logger, audit: auditLog})
}

type fileHandler struct {
	filev1.UnimplementedFileServiceServer
	svc    *service.FileService
	logger *slog.Logger
	audit  *audit.Logger
}

func (h *fileHandler) ownerID(ctx context.Context, requestOwnerID string) (string, error) {
	ownerID, err := auth.ResolveOwnerID(ctx, requestOwnerID)
	if err != nil {
		return "", status.Error(codes.PermissionDenied, "owner_id mismatch")
	}
	if ownerID == "" {
		return "", status.Error(codes.InvalidArgument, "owner_id is required")
	}
	return ownerID, nil
}

func (h *fileHandler) CreateUpload(ctx context.Context, req *filev1.CreateUploadRequest) (*filev1.CreateUploadResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.CreateUpload(ctx, ownerID, req.GetOriginalName(), req.GetContentType(), req.GetSizeBytes())
	if err != nil {
		return nil, mapError(err)
	}
	h.audit.Log(ctx, "file.created", "owner_id", ownerID, "file_id", result.Metadata.ID)
	return &filev1.CreateUploadResponse{
		Metadata:         toProtoFile(result.Metadata),
		UploadUrl:        result.UploadURL,
		ExpiresInSeconds: int64(result.ExpiresIn.Seconds()),
	}, nil
}

func (h *fileHandler) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*filev1.FileMetadata, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	file, err := h.svc.ConfirmUpload(ctx, req.GetId(), ownerID, req.GetChecksumSha256())
	if err != nil {
		return nil, mapError(err)
	}
	h.audit.Log(ctx, "upload.confirmed", "owner_id", ownerID, "file_id", file.ID)
	return toProtoFile(file), nil
}

func (h *fileHandler) GetFile(ctx context.Context, req *filev1.GetFileRequest) (*filev1.FileMetadata, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	file, err := h.svc.GetFile(ctx, req.GetId(), ownerID)
	if err != nil {
		return nil, mapError(err)
	}
	return toProtoFile(file), nil
}

func (h *fileHandler) ListFiles(ctx context.Context, req *filev1.ListFilesRequest) (*filev1.ListFilesResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.ListFiles(ctx, domain.ListFilter{
		OwnerID:   ownerID,
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
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.GetDownloadURL(ctx, req.GetId(), ownerID)
	if err != nil {
		return nil, mapError(err)
	}
	return &filev1.GetDownloadURLResponse{
		DownloadUrl:      result.URL,
		ExpiresInSeconds: int64(result.ExpiresIn.Seconds()),
	}, nil
}

func (h *fileHandler) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteFile(ctx, req.GetId(), ownerID); err != nil {
		return nil, mapError(err)
	}
	h.audit.Log(ctx, "file.deleted", "owner_id", ownerID, "file_id", req.GetId())
	return &filev1.DeleteFileResponse{Success: true}, nil
}

func (h *fileHandler) CreateMultipartUpload(ctx context.Context, req *filev1.CreateMultipartUploadRequest) (*filev1.CreateMultipartUploadResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.CreateMultipartUpload(ctx, ownerID, req.GetOriginalName(), req.GetContentType(), req.GetSizeBytes())
	if err != nil {
		return nil, mapError(err)
	}
	h.audit.Log(ctx, "file.created", "owner_id", ownerID, "file_id", result.Metadata.ID, "multipart", true)
	return &filev1.CreateMultipartUploadResponse{
		Metadata:      toProtoFile(result.Metadata),
		UploadId:      result.UploadID,
		PartSizeBytes: result.PartSize,
		TotalParts:    result.TotalParts,
	}, nil
}

func (h *fileHandler) GetPartUploadURL(ctx context.Context, req *filev1.GetPartUploadURLRequest) (*filev1.GetPartUploadURLResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.GetPartUploadURL(ctx, req.GetId(), ownerID, req.GetPartNumber())
	if err != nil {
		return nil, mapError(err)
	}
	return &filev1.GetPartUploadURLResponse{
		UploadUrl:        result.URL,
		ExpiresInSeconds: int64(result.ExpiresIn.Seconds()),
		PartNumber:       result.PartNumber,
		PartSizeBytes:    result.PartSize,
	}, nil
}

func (h *fileHandler) ReportPartUploaded(ctx context.Context, req *filev1.ReportPartUploadedRequest) (*filev1.ReportPartUploadedResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	part, err := h.svc.ReportPartUploaded(ctx, req.GetId(), ownerID, req.GetPartNumber(), req.GetEtag())
	if err != nil {
		return nil, mapError(err)
	}
	return &filev1.ReportPartUploadedResponse{
		PartNumber: part.PartNumber,
		Etag:       part.ETag,
	}, nil
}

func (h *fileHandler) ListUploadParts(ctx context.Context, req *filev1.ListUploadPartsRequest) (*filev1.ListUploadPartsResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.ListUploadParts(ctx, req.GetId(), ownerID)
	if err != nil {
		return nil, mapError(err)
	}
	parts := make([]*filev1.UploadPartInfo, 0, len(result.Parts))
	for _, p := range result.Parts {
		part := p
		parts = append(parts, &filev1.UploadPartInfo{
			PartNumber: part.PartNumber,
			Etag:       part.ETag,
			UploadedAt: timestamppb.New(part.UploadedAt),
		})
	}
	return &filev1.ListUploadPartsResponse{
		UploadId:      result.UploadID,
		PartSizeBytes: result.PartSize,
		TotalParts:    result.TotalParts,
		Parts:         parts,
	}, nil
}

func (h *fileHandler) CompleteMultipartUpload(ctx context.Context, req *filev1.CompleteMultipartUploadRequest) (*filev1.FileMetadata, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	file, err := h.svc.CompleteMultipartUpload(ctx, req.GetId(), ownerID, req.GetChecksumSha256())
	if err != nil {
		return nil, mapError(err)
	}
	h.audit.Log(ctx, "upload.confirmed", "owner_id", ownerID, "file_id", file.ID, "multipart", true)
	return toProtoFile(file), nil
}

func (h *fileHandler) AbortMultipartUpload(ctx context.Context, req *filev1.AbortMultipartUploadRequest) (*filev1.AbortMultipartUploadResponse, error) {
	ownerID, err := h.ownerID(ctx, req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	if err := h.svc.AbortMultipartUpload(ctx, req.GetId(), ownerID); err != nil {
		return nil, mapError(err)
	}
	return &filev1.AbortMultipartUploadResponse{Success: true}, nil
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
		UploadMode:     toProtoUploadMode(file.UploadMode),
		PartSizeBytes:  file.PartSize,
		TotalParts:     file.TotalParts(),
	}
}

func toProtoUploadMode(mode domain.UploadMode) filev1.UploadMode {
	switch mode {
	case domain.UploadModeSingle:
		return filev1.UploadMode_UPLOAD_MODE_SINGLE
	case domain.UploadModeMultipart:
		return filev1.UploadMode_UPLOAD_MODE_MULTIPART
	default:
		return filev1.UploadMode_UPLOAD_MODE_UNSPECIFIED
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
