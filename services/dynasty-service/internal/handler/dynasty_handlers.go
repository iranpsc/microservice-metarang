package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/service"
	commonpb "metargb/shared/pb/common"
	dynastypb "metargb/shared/pb/dynasty"
	"metargb/shared/pkg/helpers"
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
	permissionService  *service.PermissionService
	userSearchService  *service.UserSearchService
}

// NewDynastyHandler creates a new handler with all services
func NewDynastyHandler(
	dynastyService *service.DynastyService,
	joinRequestService *service.JoinRequestService,
	familyService *service.FamilyService,
	prizeService *service.PrizeService,
	permissionService *service.PermissionService,
	userSearchService *service.UserSearchService,
) *DynastyHandler {
	return &DynastyHandler{
		dynastyService:     dynastyService,
		joinRequestService: joinRequestService,
		familyService:      familyService,
		prizeService:       prizeService,
		permissionService:  permissionService,
		userSearchService:  userSearchService,
	}
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

// ============================================================================
// Dynasty Service Methods
// ============================================================================

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

// ============================================================================
// Join Request Service Methods
// ============================================================================

func (h *DynastyHandler) SendJoinRequest(ctx context.Context, req *dynastypb.SendJoinRequestRequest) (*dynastypb.JoinRequestResponse, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	var permissions *models.ChildPermission
	if req.Permissions != nil && req.Relationship == "offspring" {
		permissions = &models.ChildPermission{
			Verified: req.Permissions.Verified,
			BFR:      req.Permissions.BFR,
			SF:       req.Permissions.SF,
			W:        req.Permissions.W,
			JU:       req.Permissions.JU,
			DM:       req.Permissions.DM,
			PIUP:     req.Permissions.PIUP,
			PITC:     req.Permissions.PITC,
			PIC:      req.Permissions.PIC,
			ESOO:     req.Permissions.ESOO,
			COTB:     req.Permissions.COTB,
		}
	}

	var message *string
	if req.Message != "" {
		message = &req.Message
	}

	joinRequest, err := h.joinRequestService.SendJoinRequest(ctx, req.FromUserId, req.ToUserId, req.Relationship, message, permissions)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Get user info
	toUserInfo, _ := h.joinRequestService.GetUserBasicInfo(ctx, req.ToUserId)

	// Get prize for relationship
	prize, _ := h.joinRequestService.GetPrizeByRelationship(ctx, req.Relationship)

	return buildJoinRequestResponse(joinRequest, toUserInfo, prize), nil
}

func (h *DynastyHandler) GetSentRequests(ctx context.Context, req *dynastypb.GetSentRequestsRequest) (*dynastypb.JoinRequestsResponse, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		page = req.Pagination.Page
		perPage = req.Pagination.PerPage
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	requests, total, err := h.joinRequestService.GetSentRequests(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var responses []*dynastypb.JoinRequestResponse
	for _, req := range requests {
		toUserInfo, _ := h.joinRequestService.GetUserBasicInfo(ctx, req.ToUser)
		prize, _ := h.joinRequestService.GetPrizeByRelationship(ctx, req.Relationship)
		responses = append(responses, buildJoinRequestResponse(req, toUserInfo, prize))
	}

	return &dynastypb.JoinRequestsResponse{
		Requests: responses,
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}, nil
}

func (h *DynastyHandler) GetReceivedRequests(ctx context.Context, req *dynastypb.GetReceivedRequestsRequest) (*dynastypb.JoinRequestsResponse, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		page = req.Pagination.Page
		perPage = req.Pagination.PerPage
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	requests, total, err := h.joinRequestService.GetReceivedRequests(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var responses []*dynastypb.JoinRequestResponse
	for _, req := range requests {
		fromUserInfo, _ := h.joinRequestService.GetUserBasicInfo(ctx, req.FromUser)
		prize, _ := h.joinRequestService.GetPrizeByRelationship(ctx, req.Relationship)
		responses = append(responses, buildJoinRequestResponse(req, fromUserInfo, prize))
	}

	return &dynastypb.JoinRequestsResponse{
		Requests: responses,
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}, nil
}

func (h *DynastyHandler) GetJoinRequest(ctx context.Context, req *dynastypb.GetJoinRequestRequest) (*dynastypb.JoinRequestResponse, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	joinRequest, err := h.joinRequestService.GetJoinRequest(ctx, req.RequestId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Get user info (from or to depending on who is viewing)
	var userInfo *models.UserBasic
	if joinRequest.FromUser == req.UserId {
		userInfo, _ = h.joinRequestService.GetUserBasicInfo(ctx, joinRequest.ToUser)
	} else {
		userInfo, _ = h.joinRequestService.GetUserBasicInfo(ctx, joinRequest.FromUser)
	}

	prize, _ := h.joinRequestService.GetPrizeByRelationship(ctx, joinRequest.Relationship)

	return buildJoinRequestResponse(joinRequest, userInfo, prize), nil
}

func (h *DynastyHandler) AcceptJoinRequest(ctx context.Context, req *dynastypb.AcceptJoinRequestRequest) (*commonpb.Empty, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	err := h.joinRequestService.AcceptJoinRequest(ctx, req.RequestId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &commonpb.Empty{}, nil
}

func (h *DynastyHandler) RejectJoinRequest(ctx context.Context, req *dynastypb.RejectJoinRequestRequest) (*commonpb.Empty, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	err := h.joinRequestService.RejectJoinRequest(ctx, req.RequestId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &commonpb.Empty{}, nil
}

func (h *DynastyHandler) DeleteJoinRequest(ctx context.Context, req *dynastypb.DeleteJoinRequestRequest) (*commonpb.Empty, error) {
	if h.joinRequestService == nil {
		return nil, status.Errorf(codes.Internal, "join request service not initialized")
	}

	err := h.joinRequestService.DeleteJoinRequest(ctx, req.RequestId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &commonpb.Empty{}, nil
}

// ============================================================================
// Family Service Methods
// ============================================================================

func (h *DynastyHandler) GetFamily(ctx context.Context, req *dynastypb.GetFamilyRequest) (*dynastypb.FamilyResponse, error) {
	if h.familyService == nil {
		return nil, status.Errorf(codes.Internal, "family service not initialized")
	}

	family, err := h.familyService.GetFamily(ctx, req.FamilyId, req.DynastyId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	// Get members
	members, _, err := h.familyService.GetFamilyMembers(ctx, family.ID, 1, 1000)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var protoMembers []*dynastypb.FamilyMember
	for _, m := range members {
		userInfo, _ := h.familyService.GetUserBasicInfo(ctx, m.UserID)
		protoMembers = append(protoMembers, &dynastypb.FamilyMember{
			Id:           m.ID,
			UserId:       m.UserID,
			Relationship: m.Relationship,
			UserInfo:     buildUserBasic(userInfo),
			CreatedAt:    formatJalaliDate(m.CreatedAt),
		})
	}

	return &dynastypb.FamilyResponse{
		Id:        family.ID,
		DynastyId: family.DynastyID,
		Members:   protoMembers,
	}, nil
}

func (h *DynastyHandler) GetFamilyMembers(ctx context.Context, req *dynastypb.GetFamilyMembersRequest) (*dynastypb.FamilyMembersResponse, error) {
	if h.familyService == nil {
		return nil, status.Errorf(codes.Internal, "family service not initialized")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		page = req.Pagination.Page
		perPage = req.Pagination.PerPage
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	members, total, err := h.familyService.GetFamilyMembers(ctx, req.FamilyId, page, perPage)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var protoMembers []*dynastypb.FamilyMember
	for _, m := range members {
		userInfo, _ := h.familyService.GetUserBasicInfo(ctx, m.UserID)
		protoMembers = append(protoMembers, &dynastypb.FamilyMember{
			Id:           m.ID,
			UserId:       m.UserID,
			Relationship: m.Relationship,
			UserInfo:     buildUserBasic(userInfo),
			CreatedAt:    formatJalaliDate(m.CreatedAt),
		})
	}

	return &dynastypb.FamilyMembersResponse{
		Members: protoMembers,
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}, nil
}

func (h *DynastyHandler) SetChildPermissions(ctx context.Context, req *dynastypb.SetChildPermissionsRequest) (*commonpb.Empty, error) {
	if h.permissionService == nil {
		return nil, status.Errorf(codes.Internal, "permission service not initialized")
	}

	// API spec: POST /api/dynasty/children/{user} with {permission: "BFR", status: true}
	// Updates a single permission flag
	// The grpc-gateway sends a ChildPermissions object with only one permission set
	locale := "en" // TODO: Get locale from config or context
	if req.Permissions == nil {
		validationErrors := validateRequired("permissions", nil, locale)
		return nil, returnValidationError(validationErrors)
	}

	// Determine which permission is being set and its status
	var permission string
	var statusVal bool
	found := false

	// Check each permission flag to find the one being set
	// The gateway sets only one permission at a time
	if req.Permissions.BFR {
		permission = "BFR"
		statusVal = req.Permissions.BFR
		found = true
	} else if req.Permissions.SF {
		permission = "SF"
		statusVal = req.Permissions.SF
		found = true
	} else if req.Permissions.W {
		permission = "W"
		statusVal = req.Permissions.W
		found = true
	} else if req.Permissions.JU {
		permission = "JU"
		statusVal = req.Permissions.JU
		found = true
	} else if req.Permissions.DM {
		permission = "DM"
		statusVal = req.Permissions.DM
		found = true
	} else if req.Permissions.PIUP {
		permission = "PIUP"
		statusVal = req.Permissions.PIUP
		found = true
	} else if req.Permissions.PITC {
		permission = "PITC"
		statusVal = req.Permissions.PITC
		found = true
	} else if req.Permissions.PIC {
		permission = "PIC"
		statusVal = req.Permissions.PIC
		found = true
	} else if req.Permissions.ESOO {
		permission = "ESOO"
		statusVal = req.Permissions.ESOO
		found = true
	} else if req.Permissions.COTB {
		permission = "COTB"
		statusVal = req.Permissions.COTB
		found = true
	}

	if !found {
		validationErrors := make(map[string]string)
		t := helpers.GetLocaleTranslations(locale)
		validationErrors["permissions"] = fmt.Sprintf(t.Required, "permissions")
		return nil, returnValidationError(validationErrors)
	}

	err := h.permissionService.UpdateChildPermission(ctx, req.ParentUserId, req.ChildUserId, permission, statusVal)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &commonpb.Empty{}, nil
}

// ============================================================================
// Dynasty Prize Service Methods
// ============================================================================

func (h *DynastyHandler) GetPrizes(ctx context.Context, req *dynastypb.GetPrizesRequest) (*dynastypb.PrizesResponse, error) {
	if h.prizeService == nil {
		return nil, status.Errorf(codes.Internal, "prize service not initialized")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		page = req.Pagination.Page
		perPage = req.Pagination.PerPage
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}

	prizes, total, err := h.prizeService.GetUserReceivedPrizes(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var protoPrizes []*dynastypb.DynastyPrize
	for _, prize := range prizes {
		if prize.Prize != nil {
			protoPrizes = append(protoPrizes, buildDynastyPrize(prize.Prize))
		}
	}

	return &dynastypb.PrizesResponse{
		Prizes: protoPrizes,
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}, nil
}

func (h *DynastyHandler) GetPrize(ctx context.Context, req *dynastypb.GetPrizeRequest) (*dynastypb.PrizeResponse, error) {
	if h.prizeService == nil {
		return nil, status.Errorf(codes.Internal, "prize service not initialized")
	}

	receivedPrize, err := h.prizeService.GetReceivedPrize(ctx, req.PrizeId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	if receivedPrize == nil || receivedPrize.Prize == nil {
		return nil, status.Errorf(codes.NotFound, "prize not found")
	}

	return &dynastypb.PrizeResponse{
		Prize: buildDynastyPrize(receivedPrize.Prize),
	}, nil
}

func (h *DynastyHandler) ClaimPrize(ctx context.Context, req *dynastypb.ClaimPrizeRequest) (*commonpb.Empty, error) {
	if h.prizeService == nil {
		return nil, status.Errorf(codes.Internal, "prize service not initialized")
	}

	err := h.prizeService.ClaimPrize(ctx, req.PrizeId, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &commonpb.Empty{}, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func formatJalaliDate(t time.Time) string {
	return helpers.FormatJalaliDate(t)
}

func formatJalaliDateTime(t time.Time) string {
	return helpers.FormatJalaliDateTime(t)
}

func formatJalaliTime(t time.Time) string {
	return helpers.FormatJalaliTime(t)
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildDynastyFeature(details map[string]interface{}, memberCount int32, updatedAt time.Time) *dynastypb.DynastyFeature {
	if details == nil {
		return nil
	}

	profitIncrease := "0"
	if stability, ok := details["stability"].(string); ok {
		if stabilityInt, err := strconv.ParseFloat(stability, 64); err == nil && stabilityInt > 10000 {
			profitIncrease = fmt.Sprintf("%.3f", stabilityInt/10000-1)
		}
	}

	return &dynastypb.DynastyFeature{
		Id:                    getUint64(details["id"]),
		PropertiesId:          getString(details["properties_id"]),
		Area:                  getString(details["area"]),
		Density:               getString(details["density"]),
		FeatureProfitIncrease: profitIncrease,
		FamilyMembersCount:    memberCount,
		LastUpdated:           formatJalaliDateTime(updatedAt),
	}
}

func buildAvailableFeatures(features []map[string]interface{}) []*dynastypb.AvailableFeature {
	var result []*dynastypb.AvailableFeature
	for _, f := range features {
		result = append(result, &dynastypb.AvailableFeature{
			Id:           getUint64(f["id"]),
			PropertiesId: getString(f["properties_id"]),
			Density:      getString(f["density"]),
			Stability:    getString(f["stability"]),
			Area:         getString(f["area"]),
		})
	}
	return result
}

func buildJoinRequestResponse(req *models.JoinRequest, userInfo *models.UserBasic, prize *models.DynastyPrize) *dynastypb.JoinRequestResponse {
	resp := &dynastypb.JoinRequestResponse{
		Id:           req.ID,
		FromUser:     req.FromUser,
		ToUser:       req.ToUser,
		Status:       int32(req.Status),
		Relationship: req.Relationship,
		CreatedAt:    formatJalaliDate(req.CreatedAt),
	}

	if req.Message != nil {
		resp.Message = *req.Message
	}

	if userInfo != nil {
		resp.ToUserInfo = buildUserBasic(userInfo)
	}

	if prize != nil {
		resp.RequestPrize = buildDynastyPrize(prize)
	}

	return resp
}

func buildDynastyPrize(prize *models.DynastyPrize) *dynastypb.DynastyPrize {
	return &dynastypb.DynastyPrize{
		Id:                         prize.ID,
		Member:                     prize.Member,
		Satisfaction:               fmt.Sprintf("%.0f", prize.Satisfaction*100),
		IntroductionProfitIncrease: fmt.Sprintf("%.0f", prize.IntroductionProfitIncrease*100),
		AccumulatedCapitalReserve:  fmt.Sprintf("%.0f", prize.AccumulatedCapitalReserve*100),
		DataStorage:                fmt.Sprintf("%.0f", prize.DataStorage*100),
		Psc:                        int32(prize.PSC),
	}
}

func buildUserBasic(user *models.UserBasic) *commonpb.UserBasic {
	if user == nil {
		return nil
	}
	return &commonpb.UserBasic{
		Id:           user.ID,
		Code:         user.Code,
		Name:         user.Name,
		ProfilePhoto: stringOrEmpty(user.ProfilePhoto),
	}
}

func getUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case uint64:
		return val
	case int64:
		return uint64(val)
	case int:
		return uint64(val)
	case uint:
		return uint64(val)
	case float64:
		return uint64(val)
	default:
		return 0
	}
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func mapServiceError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Map common errors to gRPC status codes
	switch {
	case contains(errStr, "not found"):
		return status.Errorf(codes.NotFound, "%s", errStr)
	case contains(errStr, "unauthorized") || contains(errStr, "permission denied"):
		return status.Errorf(codes.PermissionDenied, "%s", errStr)
	case contains(errStr, "invalid") || contains(errStr, "validation"):
		return status.Errorf(codes.InvalidArgument, "%s", errStr)
	case contains(errStr, "already exists") || contains(errStr, "duplicate"):
		return status.Errorf(codes.AlreadyExists, "%s", errStr)
	default:
		return status.Errorf(codes.Internal, "%s", errStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOfSubstring(s, substr) >= 0)))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
