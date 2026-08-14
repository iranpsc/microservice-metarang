package service_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"metarang/features-service/internal/service"
	storagepb "metarang/shared/pb/storage"
)

type stubStorageServer struct {
	storagepb.UnimplementedFileStorageServiceServer
	resp *storagepb.ChunkUploadResponse
	err  error
}

func (s stubStorageServer) ChunkUpload(context.Context, *storagepb.ChunkUploadRequest) (*storagepb.ChunkUploadResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &storagepb.ChunkUploadResponse{
		Success:       true,
		IsFinished:    true,
		FileUrl:       "/uploads/features/9",
		FilePath:      "image_1.jpg",
		FinalFilename: "image_1.jpg",
	}, nil
}

func TestNewGRPCFileStorage_NilClient(t *testing.T) {
	require.Nil(t, service.NewGRPCFileStorage(nil))
}

func TestGRPCFileStorage_UploadChunk(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	storagepb.RegisterFileStorageServiceServer(srv, stubStorageServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	fs := service.NewGRPCFileStorage(storagepb.NewFileStorageServiceClient(conn))
	path, err := fs.UploadChunk(context.Background(), "id", "uploads/features/9", "a.jpg", "image/jpeg", []byte{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, "/uploads/features/9/image_1.jpg", path)
}

func TestGRPCFileStorage_UploadChunkFailed(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	storagepb.RegisterFileStorageServiceServer(srv, stubStorageServer{
		resp: &storagepb.ChunkUploadResponse{Success: false, Message: "disk full"},
	})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	fs := service.NewGRPCFileStorage(storagepb.NewFileStorageServiceClient(conn))
	_, err = fs.UploadChunk(context.Background(), "id", "uploads/features/9", "a.jpg", "image/jpeg", []byte{1})
	require.ErrorIs(t, err, service.ErrStorageUploadFailed)
}
