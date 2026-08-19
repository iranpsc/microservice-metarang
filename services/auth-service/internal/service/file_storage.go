package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	storagepb "metarang/shared/pb/storage"
)

var (
	ErrStorageUnavailable  = errors.New("storage service not available")
	ErrStorageUploadFailed = errors.New("storage service upload failed")
)

// FileStorage uploads and reads files via storage-service gRPC.
type FileStorage interface {
	UploadChunk(ctx context.Context, uploadID, uploadPath, filename, contentType string, data []byte) (relativePath string, err error)
	ReadFile(ctx context.Context, filePath string) (data []byte, contentType string, err error)
}

type grpcFileStorage struct {
	client storagepb.FileStorageServiceClient
}

func NewGRPCFileStorage(client storagepb.FileStorageServiceClient) FileStorage {
	if client == nil {
		return nil
	}
	return &grpcFileStorage{client: client}
}

func (s *grpcFileStorage) UploadChunk(ctx context.Context, uploadID, uploadPath, filename, contentType string, data []byte) (string, error) {
	if s == nil || s.client == nil {
		return "", ErrStorageUnavailable
	}

	chunkResp, err := s.client.ChunkUpload(ctx, &storagepb.ChunkUploadRequest{
		UploadId:    uploadID,
		ChunkData:   data,
		ChunkIndex:  0,
		TotalChunks: 1,
		Filename:    filename,
		ContentType: contentType,
		TotalSize:   int64(len(data)),
		UploadPath:  uploadPath,
	})
	if err != nil {
		return "", fmt.Errorf("storage upload: %w", err)
	}
	if !chunkResp.Success {
		return "", fmt.Errorf("%w: %s", ErrStorageUploadFailed, chunkResp.Message)
	}
	if !chunkResp.IsFinished {
		return "", fmt.Errorf("%w: upload did not complete", ErrStorageUploadFailed)
	}

	dirPath := chunkResp.FileUrl
	name := chunkResp.FilePath
	if name == "" {
		name = chunkResp.FinalFilename
	}
	if dirPath == "" || name == "" {
		return "", fmt.Errorf("%w: incomplete file path returned", ErrStorageUploadFailed)
	}

	return strings.TrimSuffix(dirPath, "/") + "/" + name, nil
}

func (s *grpcFileStorage) ReadFile(ctx context.Context, filePath string) ([]byte, string, error) {
	if s == nil || s.client == nil {
		return nil, "", ErrStorageUnavailable
	}

	stream, err := s.client.GetFile(ctx, &storagepb.GetFileRequest{FilePath: filePath})
	if err != nil {
		return nil, "", err
	}

	var data []byte
	contentType := ""
	for {
		resp, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			return nil, "", recvErr
		}
		if resp.ContentType != "" {
			contentType = resp.ContentType
		}
		data = append(data, resp.Data...)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("file not found at %q", filePath)
	}
	return data, contentType, nil
}

// PrependGatewayURL prepends the API gateway base URL when url is relative.
func PrependGatewayURL(apiGatewayURL, url string) string {
	if url == "" {
		return url
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	if apiGatewayURL == "" {
		return url
	}
	url = strings.TrimPrefix(url, "/")
	return strings.TrimSuffix(apiGatewayURL, "/") + "/" + url
}

// ResolvePublicURL returns a full URL, prepending the gateway when needed.
func ResolvePublicURL(apiGatewayURL, url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return PrependGatewayURL(apiGatewayURL, url)
}

// NewUploadID builds a unique upload identifier for storage-service chunk uploads.
func NewUploadID(prefix string, userID uint64) string {
	return fmt.Sprintf("%s_%d_%d", prefix, userID, time.Now().UnixNano())
}

func contentTypeFromFilename(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	default:
		return "video/mp4"
	}
}

// stagedVideoPaths returns candidate storage paths for a staged KYC video upload.
func stagedVideoPaths(videoPath, videoName string) []string {
	dir := strings.Trim(videoPath, "/")
	name := strings.TrimPrefix(videoName, "/")
	seen := make(map[string]struct{})
	var paths []string
	add := func(p string) {
		p = strings.ReplaceAll(p, "\\", "/")
		if _, ok := seen[p]; ok || p == "" {
			return
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	add(dir + "/" + name)
	if strings.HasPrefix(dir, "upload/") && !strings.HasPrefix(dir, "uploads/") {
		add("uploads/" + strings.TrimPrefix(dir, "upload/") + "/" + name)
	}
	if !strings.HasPrefix(dir, "uploads/") {
		add("uploads/" + dir + "/" + name)
	}
	return paths
}
