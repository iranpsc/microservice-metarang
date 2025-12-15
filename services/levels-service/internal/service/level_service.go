package service

import (
	"context"
	"fmt"

	"metargb/levels-service/internal/repository"
	pb "metargb/shared/pb/levels"
)

type LevelService struct {
	levelRepo   *repository.LevelRepository
	userLogRepo *repository.UserLogRepository
}

func NewLevelService(levelRepo *repository.LevelRepository, userLogRepo *repository.UserLogRepository) *LevelService {
	return &LevelService{
		levelRepo:   levelRepo,
		userLogRepo: userLogRepo,
	}
}

// GetUserLevel retrieves user's current level and progression
// Implements Laravel: UserController@getLevel
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
	// Implements Laravel: $latestLevel->getScorePercentageToNextLevel($user)
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
// Implements Laravel: LevelController@index
func (s *LevelService) GetAllLevels(ctx context.Context) ([]*pb.Level, error) {
	return s.levelRepo.GetAllLevels(ctx)
}

// GetLevel retrieves a specific level with all relationships
// Implements Laravel: LevelController@show
func (s *LevelService) GetLevel(ctx context.Context, levelID uint64, levelSlug string) (*pb.Level, error) {
	if levelID > 0 {
		return s.levelRepo.FindByID(ctx, levelID)
	}
	return s.levelRepo.FindBySlug(ctx, levelSlug)
}

// GetLevelGeneralInfo retrieves general info for a level
// Implements Laravel: LevelController@getGeneralInfo
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
// Implements Laravel: LevelController@gem
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
// Implements Laravel: LevelController@gift
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
// Implements Laravel: LevelController@licenses
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
// Implements Laravel: LevelController@prizes
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
// Implements Laravel: LevelPrizePolicy@recievePrize and UserObserver prize award logic
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

	// TODO: Call commercial service to increment wallet
	// This matches Laravel's prize award logic in UserObserver:
	// $wallet->increment('psc', ($levelPrize->psc / Variable::getRate('psc')));
	// $wallet->increment('blue', $levelPrize->blue);
	// $wallet->increment('red', $levelPrize->red);
	// $wallet->increment('yellow', $levelPrize->yellow);
	// $wallet->update(['effect' => $levelPrize->effect]);
	// $wallet->increment('satisfaction', $levelPrize->satisfaction);

	// Record that prize has been received
	if err := s.levelRepo.RecordReceivedPrize(ctx, userID, prize.Id); err != nil {
		return fmt.Errorf("failed to record received prize: %w", err)
	}

	return nil
}
