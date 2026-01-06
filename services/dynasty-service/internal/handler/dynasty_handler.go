package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/dynasty-service/internal/service"
	dynastypb "metargb/shared/pb/dynasty"
)

// DynastyHandler handles DynastyService gRPC methods
type DynastyHandler struct {
	dynastypb.UnimplementedDynastyServiceServer
	dynastyService *service.DynastyService
}

// NewDynastyHandler creates a new dynasty handler
func NewDynastyHandler(dynastyService *service.DynastyService) *DynastyHandler {
	return &DynastyHandler{
		dynastyService: dynastyService,
	}
}

// CreateDynasty creates a new dynasty for a user with the specified feature
func (h *DynastyHandler) CreateDynasty(ctx context.Context, req *dynastypb.CreateDynastyRequest) (*dynastypb.DynastyResponse, error) {
	if h.dynastyService == nil {
		return nil, status.Errorf(codes.Internal, "dynasty service not initialized")
	}

	dynasty, family, err := h.dynastyService.CreateDynasty(ctx, req.UserId, req.FeatureId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Get feature details
	featureDetails, err := h.dynastyService.GetFeatureDetails(ctx, dynasty.FeatureID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get feature details: %v", err)
	}

	// Get user features
	userFeatures, err := h.dynastyService.GetUserFeatures(ctx, req.UserId, dynasty.FeatureID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user features: %v", err)
	}

	// Get profile photo
	profilePhoto, _ := h.dynastyService.GetUserProfilePhoto(ctx, req.UserId)

	// Get family member count
	memberCount, _ := h.dynastyService.GetFamilyMemberCount(ctx, family.ID)

	// Build response
	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		FamilyId:       family.ID,
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
		ProfileImage:   stringOrEmpty(profilePhoto),
		DynastyFeature: buildDynastyFeature(featureDetails, memberCount, dynasty.UpdatedAt),
		Features:       buildAvailableFeatures(userFeatures),
	}

	return response, nil
}

// GetUserDynasty retrieves a user's dynasty or returns available features if none exists
func (h *DynastyHandler) GetUserDynasty(ctx context.Context, req *dynastypb.GetUserDynastyRequest) (*dynastypb.DynastyResponse, error) {
	if h.dynastyService == nil {
		return nil, status.Errorf(codes.Internal, "dynasty service not initialized")
	}

	dynasty, err := h.dynastyService.GetDynastyByUserID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dynasty: %v", err)
	}

	if dynasty == nil {
		// Return available features when no dynasty exists
		userFeatures, _ := h.dynastyService.GetUserFeatures(ctx, req.UserId, 0)

		return &dynastypb.DynastyResponse{
			UserHasDynasty: false,
			Features:       buildAvailableFeatures(userFeatures),
			// Note: intro prizes would need to be added to proto
		}, nil
	}

	// Get family
	family, _ := h.dynastyService.GetFamilyByDynastyID(ctx, dynasty.ID)
	if family == nil {
		return nil, status.Errorf(codes.Internal, "family not found for dynasty")
	}

	// Get feature details
	featureDetails, _ := h.dynastyService.GetFeatureDetails(ctx, dynasty.FeatureID)
	userFeatures, _ := h.dynastyService.GetUserFeatures(ctx, req.UserId, dynasty.FeatureID)
	profilePhoto, _ := h.dynastyService.GetUserProfilePhoto(ctx, req.UserId)
	memberCount, _ := h.dynastyService.GetFamilyMemberCount(ctx, family.ID)

	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		FamilyId:       family.ID,
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
		ProfileImage:   stringOrEmpty(profilePhoto),
		DynastyFeature: buildDynastyFeature(featureDetails, memberCount, dynasty.UpdatedAt),
		Features:       buildAvailableFeatures(userFeatures),
	}

	return response, nil
}

// GetDynasty retrieves a dynasty by ID
func (h *DynastyHandler) GetDynasty(ctx context.Context, req *dynastypb.GetDynastyRequest) (*dynastypb.DynastyResponse, error) {
	if h.dynastyService == nil {
		return nil, status.Errorf(codes.Internal, "dynasty service not initialized")
	}

	dynasty, err := h.dynastyService.GetDynastyByID(ctx, req.DynastyId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "dynasty not found")
	}

	// Get family
	family, _ := h.dynastyService.GetFamilyByDynastyID(ctx, dynasty.ID)
	if family == nil {
		return nil, status.Errorf(codes.Internal, "family not found")
	}

	// Get feature details
	featureDetails, _ := h.dynastyService.GetFeatureDetails(ctx, dynasty.FeatureID)
	userFeatures, _ := h.dynastyService.GetUserFeatures(ctx, dynasty.UserID, dynasty.FeatureID)
	profilePhoto, _ := h.dynastyService.GetUserProfilePhoto(ctx, dynasty.UserID)
	memberCount, _ := h.dynastyService.GetFamilyMemberCount(ctx, family.ID)

	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		FamilyId:       family.ID,
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
		ProfileImage:   stringOrEmpty(profilePhoto),
		DynastyFeature: buildDynastyFeature(featureDetails, memberCount, dynasty.UpdatedAt),
		Features:       buildAvailableFeatures(userFeatures),
	}

	return response, nil
}

// UpdateDynastyFeature updates a dynasty's feature
func (h *DynastyHandler) UpdateDynastyFeature(ctx context.Context, req *dynastypb.UpdateDynastyFeatureRequest) (*dynastypb.DynastyResponse, error) {
	if h.dynastyService == nil {
		return nil, status.Errorf(codes.Internal, "dynasty service not initialized")
	}

	err := h.dynastyService.UpdateDynastyFeature(ctx, req.DynastyId, req.FeatureId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Return updated dynasty
	return h.GetDynasty(ctx, &dynastypb.GetDynastyRequest{DynastyId: req.DynastyId})
}

