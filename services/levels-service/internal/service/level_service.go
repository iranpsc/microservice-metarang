package service

import (
	"context"
	"fmt"

	"metarang/levels-service/internal/client"
	pb "metarang/shared/pb/levels"
)

type levelRepository interface {
	GetUserLatestLevel(ctx context.Context, userID uint64) (*pb.Level, error)
	GetLevelsBelowScore(ctx context.Context, score int32) ([]*pb.Level, error)
	GetNextLevel(ctx context.Context, currentScore int32) (*pb.Level, error)
	GetAllLevels(ctx context.Context) ([]*pb.Level, error)
	FindByID(ctx context.Context, id uint64) (*pb.Level, error)
	FindBySlug(ctx context.Context, slug string) (*pb.Level, error)
	GetLevelGeneralInfo(ctx context.Context, levelID uint64) (*pb.LevelGeneralInfo, error)
	GetLevelGem(ctx context.Context, levelID uint64) (*pb.LevelGem, error)
	GetLevelGift(ctx context.Context, levelID uint64) (*pb.LevelGift, error)
	GetLevelLicenses(ctx context.Context, levelID uint64) (*pb.LevelLicense, error)
	GetLevelPrize(ctx context.Context, levelID uint64) (*pb.LevelPrize, error)
	HasUserReceivedPrize(ctx context.Context, userID, prizeID uint64) (bool, error)
	RecordReceivedPrize(ctx context.Context, userID, prizeID uint64) error
	GetVariableRate(ctx context.Context, name string) (float64, error)
}

type userLogRepository interface {
	GetUserScore(ctx context.Context, userID uint64) (int32, error)
}

type LevelService struct {
	levelRepo        levelRepository
	userLogRepo      userLogRepository
	commercialClient client.CommercialClient
	defaultPSCRate   float64
}

func NewLevelService(
	levelRepo levelRepository,
	userLogRepo userLogRepository,
	commercialClient client.CommercialClient,
) *LevelService {
	return &LevelService{
		levelRepo:        levelRepo,
		userLogRepo:      userLogRepo,
		commercialClient: commercialClient,
		defaultPSCRate:   30000,
	}
}

// GetUserLevel retrieves user's current level and progression
func (s *LevelService) GetUserLevel(ctx context.Context, userID uint64) (*pb.UserLevelResponse, error) {
	latestLevel, err := s.levelRepo.GetUserLatestLevel(ctx, userID)
	if err != nil {
		// User has no level yet
		return &pb.UserLevelResponse{
			LatestLevel:                nil,
			PreviousLevels:             []*pb.Level{},
			ScorePercentageToNextLevel: 0,
			UserScore:                  0,
		}, nil
	}

	// Get previous levels (all levels below current level's score)
	previousLevels, err := s.levelRepo.GetLevelsBelowScore(ctx, latestLevel.Score)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous levels: %w", err)
	}

	// Get user score
	userScore, err := s.userLogRepo.GetUserScore(ctx, userID)
	if err != nil {
		userScore = 0
	}

	// Calculate percentage to next level
	nextLevel, err := s.levelRepo.GetNextLevel(ctx, latestLevel.Score)
	scorePercentage := int32(0)
	if err == nil && nextLevel != nil {
		scorePercentage = int32((userScore * 100) / nextLevel.Score)
	}

	return &pb.UserLevelResponse{
		LatestLevel:                latestLevel,
		PreviousLevels:             previousLevels,
		ScorePercentageToNextLevel: scorePercentage,
		UserScore:                  userScore,
	}, nil
}

// GetAllLevels retrieves all levels
func (s *LevelService) GetAllLevels(ctx context.Context) ([]*pb.Level, error) {
	return s.levelRepo.GetAllLevels(ctx)
}

// GetLevel retrieves a specific level with all relationships
func (s *LevelService) GetLevel(ctx context.Context, levelID uint64, levelSlug string) (*pb.Level, error) {
	if levelID > 0 {
		return s.levelRepo.FindByID(ctx, levelID)
	}
	return s.levelRepo.FindBySlug(ctx, levelSlug)
}

// GetLevelGeneralInfo retrieves general info for a level
func (s *LevelService) GetLevelGeneralInfo(ctx context.Context, levelID uint64, levelSlug string) (*pb.LevelGeneralInfo, error) {
	// Get level first if only slug is provided
	if levelID == 0 && levelSlug != "" {
		level, err := s.levelRepo.FindBySlug(ctx, levelSlug)
		if err != nil {
			return nil, err
		}
		levelID = level.Id
	}

	return s.levelRepo.GetLevelGeneralInfo(ctx, levelID)
}

// GetLevelGem retrieves gem info for a level
func (s *LevelService) GetLevelGem(ctx context.Context, levelID uint64, levelSlug string) (*pb.LevelGem, error) {
	// Get level first if only slug is provided
	if levelID == 0 && levelSlug != "" {
		level, err := s.levelRepo.FindBySlug(ctx, levelSlug)
		if err != nil {
			return nil, err
		}
		levelID = level.Id
	}

	return s.levelRepo.GetLevelGem(ctx, levelID)
}

// GetLevelGift retrieves gift info for a level
func (s *LevelService) GetLevelGift(ctx context.Context, levelID uint64, levelSlug string) (*pb.LevelGift, error) {
	// Get level first if only slug is provided
	if levelID == 0 && levelSlug != "" {
		level, err := s.levelRepo.FindBySlug(ctx, levelSlug)
		if err != nil {
			return nil, err
		}
		levelID = level.Id
	}

	return s.levelRepo.GetLevelGift(ctx, levelID)
}

// GetLevelLicenses retrieves license info for a level
func (s *LevelService) GetLevelLicenses(ctx context.Context, levelID uint64, levelSlug string) (*pb.LevelLicense, error) {
	// Get level first if only slug is provided
	if levelID == 0 && levelSlug != "" {
		level, err := s.levelRepo.FindBySlug(ctx, levelSlug)
		if err != nil {
			return nil, err
		}
		levelID = level.Id
	}

	return s.levelRepo.GetLevelLicenses(ctx, levelID)
}

// GetLevelPrizes retrieves prizes for a level
func (s *LevelService) GetLevelPrizes(ctx context.Context, levelID uint64, levelSlug string) (*pb.LevelPrize, error) {
	// Get level first if only slug is provided
	if levelID == 0 && levelSlug != "" {
		level, err := s.levelRepo.FindBySlug(ctx, levelSlug)
		if err != nil {
			return nil, err
		}
		levelID = level.Id
	}

	prize, err := s.levelRepo.GetLevelPrize(ctx, levelID)
	if err != nil {
		return nil, err
	}
	// prize can be nil if not found (allowed per API docs)
	return prize, nil
}

// ClaimPrize allows user to claim prize (future implementation with wallet service integration)
func (s *LevelService) ClaimPrize(ctx context.Context, userID, levelID uint64) error {
	// Get the level prize
	prize, err := s.levelRepo.GetLevelPrize(ctx, levelID)
	if err != nil {
		return fmt.Errorf("failed to get level prize: %w", err)
	}
	if prize == nil {
		return fmt.Errorf("prize not found for level")
	}

	// Check if user has already received this prize
	hasReceived, err := s.levelRepo.HasUserReceivedPrize(ctx, userID, prize.Id)
	if err != nil {
		return fmt.Errorf("failed to check prize status: %w", err)
	}

	if hasReceived {
		return fmt.Errorf("prize already claimed")
	}

	pscRate, err := s.levelRepo.GetVariableRate(ctx, "psc")
	if err != nil || pscRate <= 0 {
		pscRate = s.defaultPSCRate
	}
	if err := applyLevelPrizeBalances(ctx, s.commercialClient, userID, prize, pscRate); err != nil {
		return fmt.Errorf("failed to apply level prize balances: %w", err)
	}

	// Record that prize has been received
	if err := s.levelRepo.RecordReceivedPrize(ctx, userID, prize.Id); err != nil {
		return fmt.Errorf("failed to record received prize: %w", err)
	}

	return nil
}
