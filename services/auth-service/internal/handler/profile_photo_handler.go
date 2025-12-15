package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

type profilePhotoHandler struct {
	pb.UnimplementedProfilePhotoServiceServer
	profilePhotoService service.ProfilePhotoService
}

func RegisterProfilePhotoHandler(grpcServer *grpc.Server, profilePhotoService service.ProfilePhotoService) {
	pb.RegisterProfilePhotoServiceServer(grpcServer, &profilePhotoHandler{
		profilePhotoService: profilePhotoService,
	})
}

// ListProfilePhotos returns all profile photos for the authenticated user
func (h *profilePhotoHandler) ListProfilePhotos(ctx context.Context, req *pb.ListProfilePhotosRequest) (*pb.ListProfilePhotosResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	photos, err := h.profilePhotoService.ListProfilePhotos(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list profile photos: %v", err)
	}

	response := &pb.ListProfilePhotosResponse{
		Data: make([]*pb.ProfilePhoto, 0, len(photos)),
	}

	for _, photo := range photos {
		response.Data = append(response.Data, &pb.ProfilePhoto{
			Id:  photo.ID,
			Url: photo.URL,
		})
	}

	return response, nil
}

// UploadProfilePhoto uploads a new profile photo for the authenticated user
func (h *profilePhotoHandler) UploadProfilePhoto(ctx context.Context, req *pb.UploadProfilePhotoRequest) (*pb.ProfilePhotoResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	if len(req.ImageData) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "image_data is required")
	}

	if req.Filename == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filename is required")
	}

	if req.ContentType == "" {
		return nil, status.Errorf(codes.InvalidArgument, "content_type is required")
	}

	photo, err := h.profilePhotoService.UploadProfilePhoto(ctx, req.UserId, req.ImageData, req.Filename, req.ContentType)
	if err != nil {
		// Map service errors to gRPC status codes
		switch err {
		case service.ErrImageRequired:
			return nil, status.Errorf(codes.InvalidArgument, "image is required")
		case service.ErrInvalidImage:
			return nil, status.Errorf(codes.InvalidArgument, "invalid image: must be PNG or JPEG, ≤1 MB")
		default:
			return nil, status.Errorf(codes.Internal, "failed to upload profile photo: %v", err)
		}
	}

	return &pb.ProfilePhotoResponse{
		Id:  photo.ID,
		Url: photo.URL,
	}, nil
}

// GetProfilePhoto retrieves a profile photo by ID
func (h *profilePhotoHandler) GetProfilePhoto(ctx context.Context, req *pb.GetProfilePhotoRequest) (*pb.ProfilePhotoResponse, error) {
	if req.ProfilePhotoId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "profile_photo_id is required")
	}

	photo, err := h.profilePhotoService.GetProfilePhoto(ctx, req.ProfilePhotoId)
	if err != nil {
		switch err {
		case service.ErrProfilePhotoNotFound:
			return nil, status.Errorf(codes.NotFound, "profile photo not found")
		default:
			return nil, status.Errorf(codes.Internal, "failed to get profile photo: %v", err)
		}
	}

	return &pb.ProfilePhotoResponse{
		Id:  photo.ID,
		Url: photo.URL,
	}, nil
}

// DeleteProfilePhoto deletes a profile photo (with ownership check)
func (h *profilePhotoHandler) DeleteProfilePhoto(ctx context.Context, req *pb.DeleteProfilePhotoRequest) (*emptypb.Empty, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	if req.ProfilePhotoId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "profile_photo_id is required")
	}

	err := h.profilePhotoService.DeleteProfilePhoto(ctx, req.UserId, req.ProfilePhotoId)
	if err != nil {
		switch err {
		case service.ErrProfilePhotoNotFound:
			return nil, status.Errorf(codes.NotFound, "profile photo not found")
		case service.ErrUnauthorized:
			return nil, status.Errorf(codes.PermissionDenied, "unauthorized: profile photo does not belong to user")
		default:
			return nil, status.Errorf(codes.Internal, "failed to delete profile photo: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}
