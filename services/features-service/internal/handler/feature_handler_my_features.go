package handler

import (
	"context"
	"fmt"
	"strings"

	pb "metargb/shared/pb/features"
	"metargb/shared/pkg/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListMyFeatures handles GET /api/my-features
// Returns paginated list of authenticated user's features (5 per page)
func (h *FeatureHandler) ListMyFeatures(ctx context.Context, req *pb.ListMyFeaturesRequest) (*pb.ListMyFeaturesResponse, error) {
	// Get authenticated user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: authentication required")
	}

	// Use authenticated user ID (ignore req.UserId from path)
	page := req.Page
	if page < 1 {
		page = 1
	}

	features, err := h.service.ListMyFeatures(ctx, user.UserID, page)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list features: %v", err)
	}

	// Build pagination links (simple pagination - no total counts)
	basePath := "/api/my-features"
	links := &pb.PaginationLinks{
		First: fmt.Sprintf("%s?page=1", basePath),
		Last:  "", // Unknown without total
		Prev:  "",
		Next:  "",
	}

	if page > 1 {
		links.Prev = fmt.Sprintf("%s?page=%d", basePath, page-1)
	}

	// If we got 5 results, there might be a next page
	if len(features) == 5 {
		links.Next = fmt.Sprintf("%s?page=%d", basePath, page+1)
	}

	meta := &pb.SimplePaginationMeta{
		CurrentPage: page,
		Path:        basePath,
		PerPage:     5,
	}

	return &pb.ListMyFeaturesResponse{
		Data:  features,
		Links: links,
		Meta:  meta,
	}, nil
}

// GetMyFeature handles GET /api/my-features/{user}/features/{feature}
// Returns a single feature with all relations (properties, images, latestTraded, geometry)
func (h *FeatureHandler) GetMyFeature(ctx context.Context, req *pb.GetMyFeatureRequest) (*pb.FeatureResponse, error) {
	// Get authenticated user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: authentication required")
	}

	// Verify scoped binding: feature must belong to user_id from path
	if req.UserId != user.UserID {
		return nil, status.Errorf(codes.PermissionDenied, "feature does not belong to user")
	}

	feature, err := h.service.GetMyFeature(ctx, req.UserId, req.FeatureId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "feature not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get feature: %v", err)
	}

	return &pb.FeatureResponse{
		Feature: feature,
	}, nil
}

// AddMyFeatureImages handles POST /api/my-features/{user}/add-image/{feature}
// Uploads images and attaches them to a feature
func (h *FeatureHandler) AddMyFeatureImages(ctx context.Context, req *pb.AddMyFeatureImagesRequest) (*pb.FeatureResponse, error) {
	// Get authenticated user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: authentication required")
	}

	// Verify scoped binding
	if req.UserId != user.UserID {
		return nil, status.Errorf(codes.PermissionDenied, "feature does not belong to user")
	}

	// Validate images
	if len(req.ImageData) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one image is required")
	}

	// TODO: Upload images to storage service and get URLs
	// For now, we'll expect the grpc-gateway to handle file uploads and pass URLs
	// This is a placeholder - the actual implementation should handle file uploads
	imageURLs := make([]string, 0, len(req.ImageData))
	for i, imgData := range req.ImageData {
		// In a real implementation, upload to storage service
		// For now, create a placeholder URL
		// The grpc-gateway should handle actual file uploads
		_ = imgData // Use image data
		_ = req.Filenames[i]
		_ = req.ContentTypes[i]
		// Placeholder - should be replaced with actual storage URL
		imageURLs = append(imageURLs, fmt.Sprintf("uploads/features/%d/image_%d.jpg", req.FeatureId, i+1))
	}

	feature, err := h.service.AddMyFeatureImages(ctx, req.UserId, req.FeatureId, imageURLs)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "feature not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to add images: %v", err)
	}

	return &pb.FeatureResponse{
		Feature: feature,
	}, nil
}

// RemoveMyFeatureImage handles POST /api/my-features/{user}/remove-image/{feature}/image/{image}
// Removes a single image from a feature
func (h *FeatureHandler) RemoveMyFeatureImage(ctx context.Context, req *pb.RemoveMyFeatureImageRequest) (*emptypb.Empty, error) {
	// Get authenticated user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: authentication required")
	}

	// Verify scoped binding
	if req.UserId != user.UserID {
		return nil, status.Errorf(codes.PermissionDenied, "feature does not belong to user")
	}

	err = h.service.RemoveMyFeatureImage(ctx, req.UserId, req.FeatureId, req.ImageId)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "image not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove image: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// UpdateMyFeature handles POST /api/my-features/{user}/features/{feature}
// Updates minimum price percentage and recalculates pricing
func (h *FeatureHandler) UpdateMyFeature(ctx context.Context, req *pb.UpdateMyFeatureRequest) (*emptypb.Empty, error) {
	// Get authenticated user from context
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized: authentication required")
	}

	// Verify scoped binding
	if req.UserId != user.UserID {
		return nil, status.Errorf(codes.PermissionDenied, "feature does not belong to user")
	}

	// Validate minimum_price_percentage
	if req.MinimumPricePercentage < 80 {
		return nil, status.Errorf(codes.InvalidArgument, "minimum_price_percentage must be at least 80")
	}

	err = h.service.UpdateMyFeature(ctx, req.UserId, req.FeatureId, req.MinimumPricePercentage)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Errorf(codes.NotFound, "feature not found")
		}
		// Check for validation errors from pricing service
		if strings.Contains(err.Error(), "حداقل درصد") {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to update feature: %v", err)
	}

	return &emptypb.Empty{}, nil
}
