package handler

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
	"metargb/shared/pkg/helpers"
)

type citizenHandler struct {
	pb.UnimplementedCitizenServiceServer
	citizenService service.CitizenService
}

func RegisterCitizenHandler(grpcServer *grpc.Server, citizenService service.CitizenService) {
	pb.RegisterCitizenServiceServer(grpcServer, &citizenHandler{
		citizenService: citizenService,
	})
}

// GetCitizenProfile returns the public profile for a citizen identified by code
func (h *citizenHandler) GetCitizenProfile(ctx context.Context, req *pb.GetCitizenProfileRequest) (*pb.CitizenProfileResponse, error) {
	if req.Code == "" {
		locale := "en" // TODO: Get locale from config or context
		t := helpers.GetLocaleTranslations(locale)
		validationErrors := map[string]string{
			"code": fmt.Sprintf(t.Required, "code"),
		}
		return nil, returnValidationError(validationErrors)
	}

	profile, err := h.citizenService.GetCitizenProfile(ctx, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get citizen profile: %v", err)
	}
	if profile == nil {
		return nil, status.Errorf(codes.NotFound, "citizen not found")
	}

	response := &pb.CitizenProfileResponse{
		Code:     profile.Code,
		Name:     profile.Name,
		Position: profile.Position,
		Score:    profile.Score,
	}

	// Convert profile photos
	for _, photo := range profile.ProfilePhotos {
		response.ProfilePhotos = append(response.ProfilePhotos, &pb.ProfilePhoto{
			Id:  photo.ID,
			Url: photo.URL,
		})
	}

	// Convert KYC data
	if profile.KYC != nil {
		kyc := &pb.CitizenKYC{
			Fname: profile.KYC.Fname,
			Lname: profile.KYC.Lname,
		}

		// Add nationality flag URL if privacy allows
		if profile.Privacy == nil || h.checkPrivacy(profile.Privacy, "nationality") {
			kyc.Nationality = "/uploads/flags/iran.svg" // Default flag URL
		}

		// Add birth date if privacy allows
		if !profile.KYC.Birthdate.IsZero() && (profile.Privacy == nil || h.checkPrivacy(profile.Privacy, "birth_date")) {
			kyc.BirthDate = formatJalaliDate(profile.KYC.Birthdate)
		}

		// Add phone if privacy allows
		if profile.Phone != "" && (profile.Privacy == nil || h.checkPrivacy(profile.Privacy, "phone")) {
			kyc.Phone = profile.Phone
		}

		// Add email if privacy allows
		if profile.Email != "" && (profile.Privacy == nil || h.checkPrivacy(profile.Privacy, "email")) {
			kyc.Email = profile.Email
		}

		// Add address if privacy allows
		if profile.KYC.Address != "" && (profile.Privacy == nil || h.checkPrivacy(profile.Privacy, "address")) {
			kyc.Address = profile.KYC.Address
		}

		response.Kyc = kyc
	}

	// Add registered_at in Jalali format
	response.RegisteredAt = formatJalaliDate(profile.RegisteredAt)

	// Convert personal info (customs)
	if profile.PersonalInfo != nil {
		customs := &pb.CitizenCustoms{
			Occupation: profile.PersonalInfo.Occupation,
			Education:  profile.PersonalInfo.Education,
			Prediction: profile.PersonalInfo.Prediction,
		}

		// Convert passions map (passion_key -> icon_url)
		if profile.PersonalInfo.Passions != nil {
			customs.Passions = make(map[string]string)
			for key, enabled := range profile.PersonalInfo.Passions {
				if enabled {
					// Generate icon URL (this would typically come from a config or service)
					customs.Passions[key] = fmt.Sprintf("/uploads/passions/%s.svg", key)
				}
			}
		}

		response.Customs = customs
	}

	// Add score percentage to next level (would need level service call)
	// For now, set to 0
	response.ScorePercentageToNextLevel = 0

	// Add current_level if privacy allows
	if profile.CurrentLevel != nil {
		response.CurrentLevel = &pb.CitizenLevel{
			Id:          profile.CurrentLevel.ID,
			Title:       profile.CurrentLevel.Title,
			Description: profile.CurrentLevel.Description,
			Score:       profile.CurrentLevel.Score,
		}
	}

	// Add achieved_levels if privacy allows
	for _, level := range profile.AchievedLevels {
		response.AchievedLevels = append(response.AchievedLevels, &pb.CitizenLevel{
			Id:          level.ID,
			Title:       level.Title,
			Description: level.Description,
			Score:       level.Score,
		})
	}

	// Add avatar if privacy allows
	if profile.Avatar != "" {
		response.Avatar = profile.Avatar
	}

	return response, nil
}

// GetCitizenReferrals lists referrals for a citizen with pagination
func (h *citizenHandler) GetCitizenReferrals(ctx context.Context, req *pb.GetCitizenReferralsRequest) (*pb.CitizenReferralsResponse, error) {
	if req.Code == "" {
		locale := "en" // TODO: Get locale from config or context
		t := helpers.GetLocaleTranslations(locale)
		validationErrors := map[string]string{
			"code": fmt.Sprintf(t.Required, "code"),
		}
		return nil, returnValidationError(validationErrors)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	referrals, meta, err := h.citizenService.GetCitizenReferrals(ctx, req.Code, req.Search, page)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get referrals: %v", err)
	}
	if referrals == nil {
		// User not found
		return &pb.CitizenReferralsResponse{
			Data: []*pb.CitizenReferral{},
			Meta: &pb.PaginationMeta{
				CurrentPage: 1,
			},
		}, nil
	}

	response := &pb.CitizenReferralsResponse{
		Meta: &pb.PaginationMeta{
			CurrentPage: meta.CurrentPage,
			NextPageUrl: meta.NextPageURL,
			PrevPageUrl: meta.PrevPageURL,
		},
	}

	// Convert referrals
	for _, ref := range referrals {
		pbRef := &pb.CitizenReferral{
			Id:   ref.ID,
			Code: ref.Code,
			Name: ref.Name,
		}

		if ref.Image != "" {
			pbRef.Image = ref.Image
		}

		// Convert referrer orders
		for _, order := range ref.ReferrerOrders {
			pbRef.ReferrerOrders = append(pbRef.ReferrerOrders, &pb.ReferrerOrder{
				Id:        order.ID,
				Amount:    order.Amount,
				CreatedAt: formatJalaliDateTime(order.CreatedAt),
			})
		}

		response.Data = append(response.Data, pbRef)
	}

	return response, nil
}

// GetCitizenReferralChart provides aggregated referral analytics
func (h *citizenHandler) GetCitizenReferralChart(ctx context.Context, req *pb.GetCitizenReferralChartRequest) (*pb.CitizenReferralChartResponse, error) {
	if req.Code == "" {
		locale := "en" // TODO: Get locale from config or context
		t := helpers.GetLocaleTranslations(locale)
		validationErrors := map[string]string{
			"code": fmt.Sprintf(t.Required, "code"),
		}
		return nil, returnValidationError(validationErrors)
	}

	rangeType := req.Range
	if rangeType == "" {
		rangeType = "daily"
	}

	chartData, err := h.citizenService.GetCitizenReferralChart(ctx, req.Code, rangeType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chart data: %v", err)
	}
	if chartData == nil {
		// User not found
		return &pb.CitizenReferralChartResponse{
			Data: &pb.ReferralChartData{
				TotalReferralsCount:       "0",
				TotalReferralOrdersAmount: "0",
				ChartData:                 []*pb.ChartDataPoint{},
			},
		}, nil
	}

	response := &pb.CitizenReferralChartResponse{
		Data: &pb.ReferralChartData{
			TotalReferralsCount:       chartData.TotalReferralsCount,
			TotalReferralOrdersAmount: chartData.TotalReferralOrdersAmount,
		},
	}

	// Convert chart data points
	for _, point := range chartData.ChartData {
		response.Data.ChartData = append(response.Data.ChartData, &pb.ChartDataPoint{
			Label:       point.Label,
			Count:       point.Count,
			TotalAmount: point.TotalAmount,
		})
	}

	return response, nil
}

// Helper functions

func formatJalaliDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	// Use the helper from service package
	return service.FormatJalaliDate(t)
}

func formatJalaliDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	// Format as Y-m-d H:i:s (Jalali)
	return service.FormatJalaliDateTime(t)
}

func (h *citizenHandler) checkPrivacy(privacy map[string]bool, field string) bool {
	if privacy == nil {
		return true
	}
	if value, exists := privacy[field]; exists {
		return value
	}
	return true
}
