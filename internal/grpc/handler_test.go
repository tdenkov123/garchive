package grpc_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	filev1 "github.com/tdenkov123/file-metadata-service/api/gen/file/v1"
	"github.com/tdenkov123/file-metadata-service/internal/audit"
	grpchandler "github.com/tdenkov123/file-metadata-service/internal/grpc"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

const bufSize = 1024 * 1024

func setupHandlerServer(t *testing.T, svc *service.FileService) (*grpc.ClientConn, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	auditLog := audit.New(nil)
	grpchandler.Register(s, svc, nil, auditLog)

	go func() { _ = s.Serve(lis) }()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	cleanup := func() {
		conn.Close()
		s.Stop()
	}
	return conn, cleanup
}

func TestMapError_NotFound(t *testing.T) {
	repo := newHandlerMockRepo()
	svc := service.NewFileService(repo, &handlerMockStorage{bucket: "files"}, &handlerMockCache{}, &handlerMockEvents{}, 5*1024*1024, 5*1024*1024*1024)
	conn, cleanup := setupHandlerServer(t, svc)
	defer cleanup()

	client := filev1.NewFileServiceClient(conn)
	_, err := client.GetFile(context.Background(), &filev1.GetFileRequest{Id: "missing", OwnerId: "user-1"})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestHandler_CreateUpload(t *testing.T) {
	repo := newHandlerMockRepo()
	svc := service.NewFileService(repo, &handlerMockStorage{bucket: "files"}, &handlerMockCache{}, &handlerMockEvents{}, 5*1024*1024, 5*1024*1024*1024)
	conn, cleanup := setupHandlerServer(t, svc)
	defer cleanup()

	client := filev1.NewFileServiceClient(conn)
	resp, err := client.CreateUpload(context.Background(), &filev1.CreateUploadRequest{
		OwnerId:      "user-1",
		OriginalName: "doc.pdf",
		ContentType:  "application/pdf",
		SizeBytes:    1024,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetMetadata().GetId())
	require.NotEmpty(t, resp.GetUploadUrl())
}
