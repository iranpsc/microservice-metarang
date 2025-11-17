package handler

import (
	"context"

	"metargb/levels-service/internal/service"
	pb "metargb/shared/pb/levels"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LevelHandler struct {
	pb.UnimplementedLevelServiceServer
	service *service.LevelService
}

func NewLevelHandler(service *service.LevelService) *LevelHandler {
	return &LevelHandler{
		service: service,
	}
}

// GetUserLevel retrieves user's current level and progression
// Implements Laravel's UserController@getLevel
func (h *LevelHandler) GetUserLevel(ctx context.Context, req *pb.GetUserLevelRequest) (*pb.UserLevelResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	userLevel, err := h.service.GetUserLevel(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user level: %v", err)
	}

	return userLevel, nil
}

// GetAllLevels retrieves all levels in the system
// Implements Laravel's LevelController@index (V2)
func (h *LevelHandler) GetAllLevels(ctx context.Context, req *pb.GetAllLevelsRequest) (*pb.LevelsResponse, error) {
	levels, err := h.service.GetAllLevels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get all levels: %v", err)
	}

	return &pb.LevelsResponse{
		Levels: levels,
	}, nil
}

// GetLevel retrieves a specific level by ID or slug
// Implements Laravel's LevelController@show (V2)
func (h *LevelHandler) GetLevel(ctx context.Context, req *pb.GetLevelRequest) (*pb.LevelResponse, error) {
	if req.LevelId == 0 && req.LevelSlug == "" {
		return nil, status.Errorf(codes.InvalidArgument, "level_id or level_slug is required")
	}

	level, err := h.service.GetLevel(ctx, req.LevelId, req.LevelSlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "level not found: %v", err)
	}

	return &pb.LevelResponse{
		Level: level,
	}, nil
}

// GetLevelGeneralInfo retrieves general information for a level
// Implements Laravel's LevelController@getGeneralInfo (V2)
func (h *LevelHandler) GetLevelGeneralInfo(ctx context.Context, req *pb.GetLevelGeneralInfoRequest) (*pb.LevelGeneralInfoResponse, error) {
	if req.LevelId == 0 && req.LevelSlug == "" {
		return nil, status.Errorf(codes.InvalidArgument, "level_id or level_slug is required")
	}

	generalInfo, err := h.service.GetLevelGeneralInfo(ctx, req.LevelId, req.LevelSlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "general info not found: %v", err)
	}

	return &pb.LevelGeneralInfoResponse{
		GeneralInfo: generalInfo,
	}, nil
}

// GetLevelGem retrieves gem information for a level
// Implements Laravel's LevelController@gem (V2)
func (h *LevelHandler) GetLevelGem(ctx context.Context, req *pb.GetLevelGemRequest) (*pb.LevelGemResponse, error) {
	if req.LevelId == 0 && req.LevelSlug == "" {
		return nil, status.Errorf(codes.InvalidArgument, "level_id or level_slug is required")
	}

	gem, err := h.service.GetLevelGem(ctx, req.LevelId, req.LevelSlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gem not found: %v", err)
	}

	return &pb.LevelGemResponse{
		Gem: gem,
	}, nil
}

// GetLevelGift retrieves gift information for a level
// Implements Laravel's LevelController@gift (V2)
func (h *LevelHandler) GetLevelGift(ctx context.Context, req *pb.GetLevelGiftRequest) (*pb.LevelGiftResponse, error) {
	if req.LevelId == 0 && req.LevelSlug == "" {
		return nil, status.Errorf(codes.InvalidArgument, "level_id or level_slug is required")
	}

	gift, err := h.service.GetLevelGift(ctx, req.LevelId, req.LevelSlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gift not found: %v", err)
	}

	return &pb.LevelGiftResponse{
		Gift: gift,
	}, nil
}

// GetLevelLicenses retrieves license information for a level
// Implements Laravel's LevelController@licenses (V2)
func (h *LevelHandler) GetLevelLicenses(ctx context.Context, req *pb.GetLevelLicensesRequest) (*pb.LevelLicensesResponse, error) {
	if req.LevelId == 0 && req.LevelSlug == "" {
		return nil, status.Errorf(codes.InvalidArgument, "level_id or level_slug is required")
	}

	licenses, err := h.service.GetLevelLicenses(ctx, req.LevelId, req.LevelSlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "licenses not found: %v", err)
	}

	return &pb.LevelLicensesResponse{
		Licenses: licenses,
	}, nil
}

// GetLevelPrizes retrieves prizes for a specific level
// Implements Laravel's LevelController@prizes (V2)
func (h *LevelHandler) GetLevelPrizes(ctx context.Context, req *pb.GetLevelPrizesRequest) (*pb.LevelPrizesResponse, error) {
	if req.LevelId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "level_id is required")
	}

	prize, err := h.service.GetLevelPrizes(ctx, req.LevelId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "level prize not found: %v", err)
	}

	return &pb.LevelPrizesResponse{
		Prize: prize,
	}, nil
}

// ClaimPrize allows user to claim prize for reaching a level
// Implements Laravel's prize claiming logic with policy check
func (h *LevelHandler) ClaimPrize(ctx context.Context, req *pb.ClaimPrizeRequest) (*pb.ClaimPrizeResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}
	if req.LevelId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "level_id is required")
	}

	err := h.service.ClaimPrize(ctx, req.UserId, req.LevelId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to claim prize: %v", err)
	}

	return &pb.ClaimPrizeResponse{
		Success: true,
		Message: "Prize claimed successfully",
	}, nil
}
