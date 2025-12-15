package service

import (
	"context"
)

// HelperService provides helper methods that integrate with other microservices
// These methods implement the Laravel helper functions that require cross-service calls
type HelperService interface {
	// GetUnansweredQuestionsCount calls Levels service to get unanswered questions count
	GetUnansweredQuestionsCount(ctx context.Context, userID uint64) (int32, error)

	// GetHourlyProfitTimePercentage calls Features service to get hourly profit percentage
	GetHourlyProfitTimePercentage(ctx context.Context, userID uint64) (float64, error)

	// GetScorePercentageToNextLevel calls Levels service to calculate score percentage
	GetScorePercentageToNextLevel(ctx context.Context, userID uint64, currentScore int32) (float64, error)

	// GetUserLevel calls Levels service to get user's current level
	GetUserLevel(ctx context.Context, userID uint64) (*LevelInfo, error)
}

type helperService struct {
	levelsServiceAddr   string
	featuresServiceAddr string
}

// NewHelperService creates a new helper service
func NewHelperService(levelsAddr, featuresAddr string) HelperService {
	return &helperService{
		levelsServiceAddr:   levelsAddr,
		featuresServiceAddr: featuresAddr,
	}
}

// NOTE: These are currently stub implementations that return default values
// TODO: Implement actual gRPC calls once the Levels and Features services
// have the required RPC methods defined in their proto files:
// - levels.proto: GetUnansweredQuestionsCount, GetScorePercentageToNextLevel, GetUserLevel
// - features.proto: GetHourlyProfitTimePercentage

// GetUnansweredQuestionsCount implements the Laravel getUnansweredQuestionsCount helper
// Calls the Levels service to get count of questions user hasn't answered
func (s *helperService) GetUnansweredQuestionsCount(ctx context.Context, userID uint64) (int32, error) {
	// TODO: Implement actual gRPC call to Levels service
	// The Levels service needs to implement this RPC method first
	return 0, nil
}

// GetHourlyProfitTimePercentage implements the Laravel hourlyProfitInfo helper
// Calls the Features service to calculate time percentage for hourly profit
func (s *helperService) GetHourlyProfitTimePercentage(ctx context.Context, userID uint64) (float64, error) {
	// TODO: Implement actual gRPC call to Features service
	// The Features service needs to implement this RPC method first
	return 0.0, nil
}

// GetScorePercentageToNextLevel implements the Laravel getScorePercentageToNextLevel helper
// Calls the Levels service to calculate percentage of score needed for next level
func (s *helperService) GetScorePercentageToNextLevel(ctx context.Context, userID uint64, currentScore int32) (float64, error) {
	// TODO: Implement actual gRPC call to Levels service
	// The Levels service needs to implement this RPC method first
	return 0.0, nil
}

// GetUserLevel calls Levels service to get user's current level
func (s *helperService) GetUserLevel(ctx context.Context, userID uint64) (*LevelInfo, error) {
	// TODO: Implement actual gRPC call to Levels service
	// The Levels service needs to implement this RPC method first
	return nil, nil
}
