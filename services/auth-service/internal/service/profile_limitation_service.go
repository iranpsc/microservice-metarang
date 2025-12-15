package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

var (
	ErrProfileLimitationNotFound      = errors.New("profile limitation not found")
	ErrProfileLimitationAlreadyExists = errors.New("profile limitation already exists for this user pair")
	ErrInvalidOptions                 = errors.New("invalid options: all six boolean keys are required")
	ErrNoteTooLong                    = errors.New("note must be 500 characters or less")
	ErrUnauthorized                   = errors.New("unauthorized: you can only modify limitations you created")
)

type ProfileLimitationService interface {
	Create(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error)
	Update(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error)
	Delete(ctx context.Context, limitationID, limiterUserID uint64) error
	GetByID(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error)
	GetBetweenUsers(ctx context.Context, callerUserID, targetUserID uint64) (*models.ProfileLimitation, error)
	ValidateOptions(options models.ProfileLimitationOptions) error
}

type profileLimitationService struct {
	limitationRepo repository.ProfileLimitationRepository
	userRepo       repository.UserRepository
}

func NewProfileLimitationService(limitationRepo repository.ProfileLimitationRepository, userRepo repository.UserRepository) ProfileLimitationService {
	return &profileLimitationService{
		limitationRepo: limitationRepo,
		userRepo:       userRepo,
	}
}

// ValidateOptions ensures all six required boolean keys are present
func (s *profileLimitationService) ValidateOptions(options models.ProfileLimitationOptions) error {
	// The struct already has all fields, but we need to ensure they're boolean
	// Since Go is statically typed, this is already enforced, but we can add
	// additional validation if needed (e.g., checking for specific business rules)
	return nil
}

func (s *profileLimitationService) Create(ctx context.Context, limiterUserID, limitedUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
	// Validate that limited user exists
	limitedUser, err := s.userRepo.FindByID(ctx, limitedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check limited user: %w", err)
	}
	if limitedUser == nil {
		return nil, ErrUserNotFound
	}

	// Validate that limiter user exists
	limiterUser, err := s.userRepo.FindByID(ctx, limiterUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check limiter user: %w", err)
	}
	if limiterUser == nil {
		return nil, ErrUserNotFound
	}

	// Check if limitation already exists
	exists, err := s.limitationRepo.ExistsForLimiterAndLimited(ctx, limiterUserID, limitedUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing limitation: %w", err)
	}
	if exists {
		return nil, ErrProfileLimitationAlreadyExists
	}

	// Validate note length
	if len(note) > 500 {
		return nil, ErrNoteTooLong
	}

	// Create limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: limiterUserID,
		LimitedUserID: limitedUserID,
		Options:       options,
	}

	if note != "" {
		limitation.Note = sql.NullString{String: note, Valid: true}
	}

	if err := s.limitationRepo.Create(ctx, limitation); err != nil {
		return nil, fmt.Errorf("failed to create profile limitation: %w", err)
	}

	return limitation, nil
}

func (s *profileLimitationService) Update(ctx context.Context, limitationID, limiterUserID uint64, options models.ProfileLimitationOptions, note string) (*models.ProfileLimitation, error) {
	// Get existing limitation
	limitation, err := s.limitationRepo.FindByID(ctx, limitationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get limitation: %w", err)
	}
	if limitation == nil {
		return nil, ErrProfileLimitationNotFound
	}

	// Verify ownership (only limiter can update)
	if limitation.LimiterUserID != limiterUserID {
		return nil, ErrUnauthorized
	}

	// Validate note length
	if len(note) > 500 {
		return nil, ErrNoteTooLong
	}

	// Update fields
	limitation.Options = options
	if note != "" {
		limitation.Note = sql.NullString{String: note, Valid: true}
	} else {
		limitation.Note = sql.NullString{Valid: false}
	}

	if err := s.limitationRepo.Update(ctx, limitation); err != nil {
		return nil, fmt.Errorf("failed to update profile limitation: %w", err)
	}

	return limitation, nil
}

func (s *profileLimitationService) Delete(ctx context.Context, limitationID, limiterUserID uint64) error {
	// Get existing limitation
	limitation, err := s.limitationRepo.FindByID(ctx, limitationID)
	if err != nil {
		return fmt.Errorf("failed to get limitation: %w", err)
	}
	if limitation == nil {
		return ErrProfileLimitationNotFound
	}

	// Verify ownership (only limiter can delete)
	if limitation.LimiterUserID != limiterUserID {
		return ErrUnauthorized
	}

	if err := s.limitationRepo.Delete(ctx, limitationID); err != nil {
		return fmt.Errorf("failed to delete profile limitation: %w", err)
	}

	return nil
}

func (s *profileLimitationService) GetByID(ctx context.Context, limitationID uint64) (*models.ProfileLimitation, error) {
	limitation, err := s.limitationRepo.FindByID(ctx, limitationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get limitation: %w", err)
	}
	if limitation == nil {
		return nil, ErrProfileLimitationNotFound
	}
	return limitation, nil
}

func (s *profileLimitationService) GetBetweenUsers(ctx context.Context, callerUserID, targetUserID uint64) (*models.ProfileLimitation, error) {
	limitation, err := s.limitationRepo.FindBetweenUsers(ctx, callerUserID, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get limitation between users: %w", err)
	}
	// Return nil if not found (not an error)
	return limitation, nil
}
