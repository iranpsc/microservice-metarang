package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "metargb/shared/pb/common"
	storagepb "metargb/shared/pb/storage"
	"metargb/storage-service/internal/service"
)

type ImageHandler struct {
	storagepb.UnimplementedImageServiceServer
	service *service.ImageService
}

func RegisterImageHandler(grpcServer *grpc.Server, svc *service.ImageService) {
	handler := &ImageHandler{service: svc}
	storagepb.RegisterImageServiceServer(grpcServer, handler)
}

// CreateImage creates a new image record
func (h *ImageHandler) CreateImage(ctx context.Context, req *storagepb.CreateImageRequest) (*storagepb.ImageResponse, error) {
	var imageType *string
	if req.Type != "" {
		imageType = &req.Type
	}

	image, err := h.service.CreateImage(ctx, req.ImageableType, req.ImageableId, req.Url, imageType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create image: %v", err)
	}

	return &storagepb.ImageResponse{
		Id:            image.ID,
		ImageableType: image.ImageableType,
		ImageableId:   image.ImageableID,
		Url:           image.URL,
		Type:          stringOrEmpty(image.Type),
		CreatedAt:     image.CreatedAt.Format("2006/01/02"),
	}, nil
}

// GetImages retrieves images for an entity
func (h *ImageHandler) GetImages(ctx context.Context, req *storagepb.GetImagesRequest) (*storagepb.ImagesResponse, error) {
	images, err := h.service.GetImages(ctx, req.ImageableType, req.ImageableId, req.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get images: %v", err)
	}

	response := &storagepb.ImagesResponse{
		Images: make([]*storagepb.ImageResponse, 0, len(images)),
	}

	for _, img := range images {
		response.Images = append(response.Images, &storagepb.ImageResponse{
			Id:            img.ID,
			ImageableType: img.ImageableType,
			ImageableId:   img.ImageableID,
			Url:           img.URL,
			Type:          stringOrEmpty(img.Type),
			CreatedAt:     img.CreatedAt.Format("2006/01/02"),
		})
	}

	return response, nil
}

// DeleteImage deletes an image
func (h *ImageHandler) DeleteImage(ctx context.Context, req *storagepb.DeleteImageRequest) (*commonpb.Empty, error) {
	if err := h.service.DeleteImage(ctx, req.ImageId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete image: %v", err)
	}

	return &commonpb.Empty{}, nil
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

