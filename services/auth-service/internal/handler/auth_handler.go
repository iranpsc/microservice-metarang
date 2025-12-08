package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"metargb/auth-service/internal/repository"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

type authHandler struct {
	pb.UnimplementedAuthServiceServer
	authService service.AuthService
	tokenRepo   repository.TokenRepository
}

func RegisterAuthHandler(grpcServer *grpc.Server, authService service.AuthService, tokenRepo repository.TokenRepository) {
	pb.RegisterAuthServiceServer(grpcServer, &authHandler{
		authService: authService,
		tokenRepo:   tokenRepo,
	})
}

func (h *authHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	url, err := h.authService.Register(ctx, req.BackUrl, req.Referral)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "registration failed: %v", err)
	}

	return &pb.RegisterResponse{
		Url: url,
	}, nil
}

func (h *authHandler) Redirect(ctx context.Context, req *pb.RedirectRequest) (*pb.RedirectResponse, error) {
	url, _, err := h.authService.Redirect(ctx, req.BackUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "redirect failed: %v", err)
	}

	// TODO: Store state and back_url in cache (Redis)
	// For now, we're returning the URL and the state would be managed by Kong/gateway

	return &pb.RedirectResponse{
		Url: url,
	}, nil
}

func (h *authHandler) Callback(ctx context.Context, req *pb.CallbackRequest) (*pb.CallbackResponse, error) {
	// TODO: Retrieve cached state from Redis
	// For now, we'll assume state is validated by the caller
	cachedState := req.State // This should come from cache

	result, err := h.authService.Callback(ctx, req.State, req.Code, cachedState)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "callback failed: %v", err)
	}

	return &pb.CallbackResponse{
		Token:       result.Token,
		ExpiresAt:   result.ExpiresAt,
		RedirectUrl: result.RedirectURL,
	}, nil
}

func (h *authHandler) GetMe(ctx context.Context, req *pb.GetMeRequest) (*pb.UserResponse, error) {
	userDetails, err := h.authService.GetMe(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
	}

	response := &pb.UserResponse{
		Id:                         userDetails.ID,
		Name:                       userDetails.Name,
		Token:                      userDetails.Token,
		AccessToken:                userDetails.AccessToken,
		AutomaticLogout:            userDetails.AutomaticLogout,
		Code:                       userDetails.Code,
		Image:                      userDetails.Image,
		Notifications:              userDetails.Notifications,
		SocrePercentageToNextLevel: userDetails.ScorePercentageToNextLevel, // TYPO PRESERVED!
		UnasnweredQuestionsCount:   userDetails.UnansweredQuestionsCount,   // TYPO PRESERVED!
		HourlyProfitTimePercentage: userDetails.HourlyProfitTimePercentage,
		VerifiedKyc:                userDetails.VerifiedKYC,
		Birthdate:                  userDetails.Birthdate,
	}

	if userDetails.Level != nil {
		response.Level = &pb.Level{
			Id:          userDetails.Level.ID,
			Title:       userDetails.Level.Title,
			Description: userDetails.Level.Description,
			Score:       userDetails.Level.Score,
		}
	}

	return response, nil
}

func (h *authHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*emptypb.Empty, error) {
	// Validate token and get user
	user, err := h.tokenRepo.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Extract IP and UserAgent from request context (if available)
	// TODO: Extract from gRPC metadata
	ip := ""
	userAgent := ""

	if err := h.authService.Logout(ctx, user.ID, ip, userAgent); err != nil {
		return nil, status.Errorf(codes.Internal, "logout failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *authHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	user, err := h.authService.ValidateToken(ctx, req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: user.ID,
		Email:  user.Email,
	}, nil
}

func (h *authHandler) RequestAccountSecurity(ctx context.Context, req *pb.RequestAccountSecurityRequest) (*emptypb.Empty, error) {
	if err := h.authService.RequestAccountSecurity(ctx, req.UserId, req.TimeMinutes, req.Phone); err != nil {
		return nil, mapAccountSecurityError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *authHandler) VerifyAccountSecurity(ctx context.Context, req *pb.VerifyAccountSecurityRequest) (*emptypb.Empty, error) {
	if err := h.authService.VerifyAccountSecurity(ctx, req.UserId, req.Code, req.Ip, req.UserAgent); err != nil {
		return nil, mapAccountSecurityError(err)
	}
	return &emptypb.Empty{}, nil
}

func mapAccountSecurityError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidUnlockDuration),
		errors.Is(err, service.ErrInvalidPhoneFormat),
		errors.Is(err, service.ErrPhoneRequired),
		errors.Is(err, service.ErrInvalidOTPCode),
		errors.Is(err, service.ErrPhoneAlreadyTaken):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, service.ErrAccountSecurityNotFound):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, service.ErrUserNotFound):
		return status.Errorf(codes.NotFound, "%v", err)
	case errors.Is(err, service.ErrAccountSecurityAlreadyUnlocked):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "account security operation failed: %v", err)
	}
}
