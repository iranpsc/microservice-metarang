package handler

import (
	"context"

	"metargb/features-service/internal/service"
	pb "metargb/shared/pb/features"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeatureHandler struct {
	pb.UnimplementedFeatureServiceServer
	service *service.FeatureService
}

func NewFeatureHandler(service *service.FeatureService) *FeatureHandler {
	return &FeatureHandler{
		service: service,
	}
}

// ListFeatures retrieves features within a bounding box
func (h *FeatureHandler) ListFeatures(ctx context.Context, req *pb.ListFeaturesRequest) (*pb.FeaturesResponse, error) {
	// Validate points (must have exactly 4 points for bbox)
	if len(req.Points) != 4 {
		return nil, status.Errorf(codes.InvalidArgument, "exactly 4 points required for bounding box")
	}

	features, err := h.service.ListFeatures(ctx, req.Points, req.LoadBuildings, req.UserFeaturesLocation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list features: %v", err)
	}

	return &pb.FeaturesResponse{
		Features: features,
	}, nil
}

// GetFeature retrieves a single feature by ID with all relations
func (h *FeatureHandler) GetFeature(ctx context.Context, req *pb.GetFeatureRequest) (*pb.FeatureResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}

	feature, err := h.service.GetFeature(ctx, req.FeatureId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "feature not found: %v", err)
	}

	return &pb.FeatureResponse{
		Feature: feature,
	}, nil
}

// UpdateFeature updates feature properties
func (h *FeatureHandler) UpdateFeature(ctx context.Context, req *pb.UpdateFeatureRequest) (*pb.FeatureResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}

	feature, err := h.service.UpdateFeature(ctx, req.FeatureId, req.Properties)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update feature: %v", err)
	}

	return &pb.FeatureResponse{
		Feature: feature,
	}, nil
}

// AddFeatureImages adds images to a feature
func (h *FeatureHandler) AddFeatureImages(ctx context.Context, req *pb.AddFeatureImagesRequest) (*pb.FeatureResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}

	if len(req.ImageUrls) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "at least one image_url is required")
	}

	feature, err := h.service.AddFeatureImages(ctx, req.FeatureId, req.ImageUrls)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add images: %v", err)
	}

	return &pb.FeatureResponse{
		Feature: feature,
	}, nil
}

// GetMyFeatures retrieves all features owned by a user
func (h *FeatureHandler) GetMyFeatures(ctx context.Context, req *pb.GetMyFeaturesRequest) (*pb.FeaturesResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	features, err := h.service.GetMyFeatures(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user features: %v", err)
	}

	return &pb.FeaturesResponse{
		Features: features,
	}, nil
}

