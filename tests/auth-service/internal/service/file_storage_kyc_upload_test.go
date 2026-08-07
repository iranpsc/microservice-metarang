package service_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	storagepb "metarang/shared/pb/storage"
)

type memFileStorage struct {
	files map[string][]byte
	fail  bool
}

func (m *memFileStorage) UploadChunk(_ context.Context, _, _, filename, _ string, data []byte) (string, error) {
	if m.fail {
		return "", service.ErrStorageUploadFailed
	}
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	path := "/uploads/kyc/" + filename
	m.files[path] = append([]byte(nil), data...)
	return path, nil
}

func (m *memFileStorage) ReadFile(_ context.Context, filePath string) ([]byte, string, error) {
	if m.fail {
		return nil, "", service.ErrStorageUnavailable
	}
	data, ok := m.files[filePath]
	if !ok {
		return nil, "", io.EOF
	}
	return data, "video/mp4", nil
}

type stubStorageServer struct {
	storagepb.UnimplementedFileStorageServiceServer
}

func (stubStorageServer) ChunkUpload(context.Context, *storagepb.ChunkUploadRequest) (*storagepb.ChunkUploadResponse, error) {
	return &storagepb.ChunkUploadResponse{
		Success: true, IsFinished: true,
		FileUrl: "/uploads/kyc", FilePath: "card.png", FinalFilename: "card.png",
	}, nil
}

func (stubStorageServer) GetFile(req *storagepb.GetFileRequest, stream storagepb.FileStorageService_GetFileServer) error {
	if req.FilePath == "missing" {
		return nil
	}
	return stream.Send(&storagepb.GetFileResponse{Data: []byte("abc"), ContentType: "image/png"})
}

func TestGRPCFileStorage(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	storagepb.RegisterFileStorageServiceServer(srv, stubStorageServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	require.Nil(t, service.NewGRPCFileStorage(nil))
	fs := service.NewGRPCFileStorage(storagepb.NewFileStorageServiceClient(conn))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	path, err := fs.UploadChunk(ctx, "id", "/uploads/kyc", "card.png", "image/png", []byte("x"))
	require.NoError(t, err)
	require.Equal(t, "/uploads/kyc/card.png", path)

	data, ct, err := fs.ReadFile(ctx, "ok.png")
	require.NoError(t, err)
	require.Equal(t, []byte("abc"), data)
	require.Equal(t, "image/png", ct)

	_, _, err = fs.ReadFile(ctx, "missing")
	require.Error(t, err)

	require.Equal(t, "https://gw/a.png", service.PrependGatewayURL("https://gw", "/a.png"))
	require.Equal(t, "https://x.com/a", service.ResolvePublicURL("https://gw", "https://x.com/a"))
	require.Contains(t, service.NewUploadID("p", 1), "p_1_")
}

func TestKYCUpload_SubmitAndValidate(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	store := &memFileStorage{files: map[string][]byte{
		"uploads/tmp/v.mp4": []byte("video"),
	}}
	svc := service.NewKYCService(kycRepo, userRepo, store, "https://gw")

	_, err := svc.SubmitKYC(ctx, 1, service.KYCSubmission{})
	require.ErrorIs(t, err, service.ErrMelliCardRequired)

	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "a.png", MelliCardContentType: "image/png",
	})
	require.ErrorIs(t, err, service.ErrVideoRequired)

	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "", MelliCardContentType: "image/png",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrMelliCardFilenameRequired)

	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "a.png", MelliCardContentType: "",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrMelliCardContentTypeRequired)

	big := make([]byte, 5*1024*1024+1)
	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: big, MelliCardFilename: "a.png", MelliCardContentType: "image/png",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrMelliCardTooLarge)

	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "a.png", MelliCardContentType: "text/plain",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrInvalidMelliCardType)

	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "a.gif", MelliCardContentType: "image/png",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrInvalidMelliCardExtension)

	svcNoStore := service.NewKYCService(kycRepo, userRepo, nil, "")
	_, err = svcNoStore.SubmitKYC(ctx, 1, service.KYCSubmission{
		MelliCardData: []byte("x"), MelliCardFilename: "a.png", MelliCardContentType: "image/png",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.ErrorIs(t, err, service.ErrStorageUnavailable)

	kyc, err := svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		Fname: "Ali", Lname: "Karimi", MelliCode: "0123456789",
		Birthdate: "1403/01/15", Province: "Tehran", Gender: "male", VerifyTextID: 1,
		MelliCardData: []byte("img"), MelliCardFilename: "card.jpg", MelliCardContentType: "image/jpeg",
		VideoPath: "tmp", VideoName: "v.mp4",
	})
	require.NoError(t, err)
	require.NotNil(t, kyc)
	require.Contains(t, kyc.MelliCard, "https://gw/")

	store.files["uploads/tmp/v.webm"] = []byte("video2")
	store.files["uploads/tmp/v.mov"] = []byte("video3")
	kycRepo.kycs[1].Status = -1
	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		Fname: "Ali", Lname: "Karimi", MelliCode: "0123456789",
		Birthdate: "1403/01/15", Province: "Tehran", Gender: "female", VerifyTextID: 1,
		MelliCardData: []byte("img"), MelliCardFilename: "card.png", MelliCardContentType: "image/png",
		VideoPath: "tmp", VideoName: "v.webm",
	})
	require.NoError(t, err)
	kycRepo.kycs[1].Status = -1
	_, err = svc.SubmitKYC(ctx, 1, service.KYCSubmission{
		Fname: "Ali", Lname: "Karimi", MelliCode: "0123456789",
		Birthdate: "1403/01/15", Province: "Tehran", Gender: "male", VerifyTextID: 1,
		MelliCardData: []byte("img"), MelliCardFilename: "card.jpeg", MelliCardContentType: "image/jpg",
		VideoPath: "tmp", VideoName: "v.mov",
	})
	require.NoError(t, err)
}
