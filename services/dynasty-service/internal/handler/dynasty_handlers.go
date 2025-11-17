package handler

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dynastypb "metargb/shared/pb/dynasty"
	"metargb/dynasty-service/internal/service"
)

type DynastyHandler struct {
	dynastypb.UnimplementedDynastyServiceServer
	dynastypb.UnimplementedJoinRequestServiceServer
	dynastypb.UnimplementedFamilyServiceServer
	dynastypb.UnimplementedDynastyPrizeServiceServer
	
	dynastyService     *service.DynastyService
	joinRequestService *service.JoinRequestService
	familyService      *service.FamilyService
	prizeService       *service.PrizeService
}

func RegisterDynastyHandler(grpcServer *grpc.Server, dynastyService *service.DynastyService) {
	handler := &DynastyHandler{dynastyService: dynastyService}
	dynastypb.RegisterDynastyServiceServer(grpcServer, handler)
}

func RegisterJoinRequestHandler(grpcServer *grpc.Server, joinRequestService *service.JoinRequestService) {
	handler := &DynastyHandler{joinRequestService: joinRequestService}
	dynastypb.RegisterJoinRequestServiceServer(grpcServer, handler)
}

func RegisterFamilyHandler(grpcServer *grpc.Server, familyService *service.FamilyService) {
	handler := &DynastyHandler{familyService: familyService}
	dynastypb.RegisterFamilyServiceServer(grpcServer, handler)
}

func RegisterPrizeHandler(grpcServer *grpc.Server, prizeService *service.PrizeService) {
	handler := &DynastyHandler{prizeService: prizeService}
	dynastypb.RegisterDynastyPrizeServiceServer(grpcServer, handler)
}

// Dynasty Service Methods

func (h *DynastyHandler) CreateDynasty(ctx context.Context, req *dynastypb.CreateDynastyRequest) (*dynastypb.DynastyResponse, error) {
	dynasty, family, err := h.dynastyService.CreateDynasty(ctx, req.UserId, req.FeatureId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create dynasty: %v", err)
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

	// Build response
	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		FamilyId:       family.ID,
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
		ProfileImage:   stringOrEmpty(profilePhoto),
		DynastyFeature: buildDynastyFeature(featureDetails, 1), // 1 member initially
		Features:       buildAvailableFeatures(userFeatures),
	}

	return response, nil
}

func (h *DynastyHandler) GetUserDynasty(ctx context.Context, req *dynastypb.GetUserDynastyRequest) (*dynastypb.DynastyResponse, error) {
	dynasty, err := h.dynastyService.GetDynastyByUserID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get dynasty: %v", err)
	}

	if dynasty == nil {
		return &dynastypb.DynastyResponse{UserHasDynasty: false}, nil
	}

	// Similar response building as CreateDynasty...
	featureDetails, _ := h.dynastyService.GetFeatureDetails(ctx, dynasty.FeatureID)
	userFeatures, _ := h.dynastyService.GetUserFeatures(ctx, req.UserId, dynasty.FeatureID)
	profilePhoto, _ := h.dynastyService.GetUserProfilePhoto(ctx, req.UserId)

	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		FamilyId:       0, // Would need to fetch from family repo
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
		ProfileImage:   stringOrEmpty(profilePhoto),
		DynastyFeature: buildDynastyFeature(featureDetails, 1),
		Features:       buildAvailableFeatures(userFeatures),
	}

	return response, nil
}

func (h *DynastyHandler) GetDynasty(ctx context.Context, req *dynastypb.GetDynastyRequest) (*dynastypb.DynastyResponse, error) {
	dynasty, err := h.dynastyService.GetDynastyByID(ctx, req.DynastyId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "dynasty not found")
	}

	// Build full response...
	response := &dynastypb.DynastyResponse{
		UserHasDynasty: true,
		Id:             dynasty.ID,
		CreatedAt:      formatJalaliDate(dynasty.CreatedAt),
	}

	return response, nil
}

func (h *DynastyHandler) UpdateDynastyFeature(ctx context.Context, req *dynastypb.UpdateDynastyFeatureRequest) (*dynastypb.DynastyResponse, error) {
	err := h.dynastyService.UpdateDynastyFeature(ctx, req.DynastyId, req.FeatureId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update dynasty feature: %v", err)
	}

	// Return updated dynasty
	return h.GetDynasty(ctx, &dynastypb.GetDynastyRequest{DynastyId: req.DynastyId})
}

// Helper functions

func formatJalaliDate(t time.Time) string {
	// TODO: Implement actual Jalali date conversion
	// For now, return formatted Gregorian date
	return t.Format("2006/01/02")
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildDynastyFeature(details map[string]interface{}, memberCount int32) *dynastypb.DynastyFeature {
	if details == nil {
		return nil
	}

	_ = fmt.Sprintf("%v", details["stability"]) // stability - reserved for future use
	profitIncrease := "0"
	// Calculate profit increase from stability (if > 10000)
	// profitIncrease = (stability / 10000 - 1) formatted to 3 decimals

	return &dynastypb.DynastyFeature{
		Id:                    details["id"].(uint64),
		PropertiesId:          details["properties_id"].(string),
		Area:                  details["area"].(string),
		Density:               details["density"].(string),
		FeatureProfitIncrease: profitIncrease,
		FamilyMembersCount:    memberCount,
		LastUpdated:           formatJalaliDate(time.Now()),
	}
}

func buildAvailableFeatures(features []map[string]interface{}) []*dynastypb.AvailableFeature {
	var result []*dynastypb.AvailableFeature
	for _, f := range features {
		result = append(result, &dynastypb.AvailableFeature{
			Id:           f["id"].(uint64),
			PropertiesId: f["properties_id"].(string),
			Density:      f["density"].(string),
			Stability:    f["stability"].(string),
			Area:         f["area"].(string),
		})
	}
	return result
}

