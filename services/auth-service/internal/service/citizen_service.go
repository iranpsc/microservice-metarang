package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
	"metargb/shared/pkg/helpers"
)

type CitizenService interface {
	GetCitizenProfile(ctx context.Context, code string) (*models.CitizenProfile, error)
	GetCitizenReferrals(ctx context.Context, code string, search string, page int32) ([]*models.CitizenReferral, *models.PaginationMeta, error)
	GetCitizenReferralChart(ctx context.Context, code string, rangeType string) (*models.ReferralChartData, error)
}

type citizenService struct {
	citizenRepo repository.CitizenRepository
	userRepo    repository.UserRepository
}

func NewCitizenService(citizenRepo repository.CitizenRepository, userRepo repository.UserRepository) CitizenService {
	return &citizenService{
		citizenRepo: citizenRepo,
		userRepo:    userRepo,
	}
}

// GetCitizenProfile retrieves a citizen's public profile with privacy filtering
func (s *citizenService) GetCitizenProfile(ctx context.Context, code string) (*models.CitizenProfile, error) {
	profile, err := s.citizenRepo.GetCitizenByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get citizen profile: %w", err)
	}
	if profile == nil {
		return nil, nil // Not found
	}

	// Get levels if privacy allows
	if s.checkPrivacy(profile.Privacy, "level") {
		currentLevel, achievedLevels, err := s.citizenRepo.GetCitizenLevels(ctx, profile.ID)
		if err == nil {
			profile.CurrentLevel = currentLevel
			profile.AchievedLevels = achievedLevels
		}
	}

	// Set avatar URL if privacy allows
	if s.checkPrivacy(profile.Privacy, "avatar") {
		// Avatar URL format: /uploads/avatars/{user_id}.svg
		profile.Avatar = fmt.Sprintf("/uploads/avatars/%d.svg", profile.ID)
	}

	// Apply privacy filtering
	s.applyPrivacyFilters(profile)

	return profile, nil
}

// GetCitizenReferrals retrieves referrals for a citizen with pagination and search
func (s *citizenService) GetCitizenReferrals(ctx context.Context, code string, search string, page int32) ([]*models.CitizenReferral, *models.PaginationMeta, error) {
	// Get user by code to get referrer_id
	user, err := s.userRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, nil, nil // Not found
	}

	// Get referrals
	referrals, meta, err := s.citizenRepo.GetCitizenReferrals(ctx, user.ID, search, int(page), 10)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get referrals: %w", err)
	}

	// Get referral orders for each referral
	for _, referral := range referrals {
		orders, err := s.citizenRepo.GetCitizenReferralOrders(ctx, referral.ID)
		if err == nil {
			referral.ReferrerOrders = orders
		}
	}

	return referrals, meta, nil
}

// GetCitizenReferralChart retrieves referral chart data for a citizen
func (s *citizenService) GetCitizenReferralChart(ctx context.Context, code string, rangeType string) (*models.ReferralChartData, error) {
	// Get user by code to get referrer_id
	user, err := s.userRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, nil // Not found
	}

	// Validate and normalize range type
	if rangeType == "" {
		rangeType = "daily"
	}
	rangeType = strings.ToLower(rangeType)
	if rangeType != "daily" && rangeType != "weekly" && rangeType != "monthly" && rangeType != "yearly" {
		rangeType = "daily"
	}

	// Get chart data
	chartData, err := s.citizenRepo.GetCitizenReferralChartData(ctx, user.ID, rangeType)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart data: %w", err)
	}

	// Chart labels will be converted to Jalali format in the handler
	// The repository returns Gregorian dates that need conversion

	return chartData, nil
}

// applyPrivacyFilters applies privacy settings to filter profile data
func (s *citizenService) applyPrivacyFilters(profile *models.CitizenProfile) {
	if profile.Privacy == nil {
		// Default: show all if no privacy settings
		return
	}

	// Filter KYC data based on privacy flags
	if profile.KYC != nil {
		if !s.checkPrivacy(profile.Privacy, "nationality") {
			// Don't show nationality
		}
		if !s.checkPrivacy(profile.Privacy, "fname") {
			profile.KYC.Fname = ""
		}
		if !s.checkPrivacy(profile.Privacy, "lname") {
			profile.KYC.Lname = ""
		}
		if !s.checkPrivacy(profile.Privacy, "birth_date") {
			profile.KYC.Birthdate = time.Time{}
		}
		if !s.checkPrivacy(profile.Privacy, "phone") {
			profile.Phone = ""
		}
		if !s.checkPrivacy(profile.Privacy, "email") {
			profile.Email = ""
		}
		if !s.checkPrivacy(profile.Privacy, "address") {
			profile.KYC.Address = ""
		}
	}

	// Filter profile photos
	if !s.checkPrivacy(profile.Privacy, "profile_photos") {
		profile.ProfilePhotos = []*models.ProfilePhoto{}
	}

	// Filter score and level data
	if !s.checkPrivacy(profile.Privacy, "score") {
		profile.Score = 0
	}

	// Filter avatar
	if !s.checkPrivacy(profile.Privacy, "avatar") {
		profile.Avatar = ""
	}

	// Filter level data
	if !s.checkPrivacy(profile.Privacy, "level") {
		profile.CurrentLevel = nil
		profile.AchievedLevels = []*models.CitizenLevel{}
	}
}

// checkPrivacy checks if a privacy flag allows showing the field
// Returns true if the field should be shown (privacy allows it)
func (s *citizenService) checkPrivacy(privacy map[string]bool, field string) bool {
	if privacy == nil {
		return true // Default: show all
	}

	// Check if field exists in privacy map
	// If it exists and is false, hide it
	// If it exists and is true, show it
	// If it doesn't exist, default to showing it
	if value, exists := privacy[field]; exists {
		return value
	}

	// Default: show if not explicitly set
	return true
}

// FormatJalaliDateTime formats a time.Time to Jalali format Y-m-d H:i:s
func FormatJalaliDateTime(t time.Time) string {
	return helpers.FormatJalaliDateTime(t)
}

// FormatJalaliDate formats a time.Time to Jalali format Y/m/d
func FormatJalaliDate(t time.Time) string {
	return helpers.FormatJalaliDate(t)
}
