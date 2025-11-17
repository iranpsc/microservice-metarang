package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "metargb/shared/pb/auth"
	"metargb/auth-service/internal/service"
)

type userHandler struct {
	pb.UnimplementedUserServiceServer
	userService service.UserService
}

func RegisterUserHandler(grpcServer *grpc.Server, userService service.UserService) {
	pb.RegisterUserServiceServer(grpcServer, &userHandler{
		userService: userService,
	})
}

func (h *userHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	user, err := h.userService.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	response := &pb.User{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Code:      user.Code,
		Score:     user.Score,
		Ip:        user.IP,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}

	if user.ReferrerID.Valid {
		response.ReferrerId = uint64(user.ReferrerID.Int64)
	}

	if user.LastSeen.Valid {
		response.LastSeen = timestamppb.New(user.LastSeen.Time)
	}

	if user.EmailVerifiedAt.Valid {
		response.EmailVerifiedAt = timestamppb.New(user.EmailVerifiedAt.Time)
	}

	if user.PhoneVerifiedAt.Valid {
		response.PhoneVerifiedAt = timestamppb.New(user.PhoneVerifiedAt.Time)
	}

	if user.AccessToken.Valid {
		response.AccessToken = user.AccessToken.String
	}

	if user.RefreshToken.Valid {
		response.RefreshToken = user.RefreshToken.String
	}

	if user.TokenType.Valid {
		response.TokenType = user.TokenType.String
	}

	if user.ExpiresIn.Valid {
		response.ExpiresIn = user.ExpiresIn.Int64
	}

	return response, nil
}

func (h *userHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.User, error) {
	user, err := h.userService.UpdateProfile(ctx, req.UserId, req.Name, req.Email, req.Phone)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update profile: %v", err)
	}

	response := &pb.User{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Code:      user.Code,
		Score:     user.Score,
		Ip:        user.IP,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}

	return response, nil
}

func (h *userHandler) GetUserWallet(ctx context.Context, req *pb.GetUserWalletRequest) (*pb.UserWalletResponse, error) {
	// This should call the Commercial service's WalletService
	// For now, return a placeholder error
	return nil, status.Errorf(codes.Unimplemented, "wallet service not yet implemented - should call Commercial service")
}

func (h *userHandler) GetUserLevel(ctx context.Context, req *pb.GetUserLevelRequest) (*pb.UserLevelResponse, error) {
	// This should call the Levels service
	// For now, return a placeholder error
	return nil, status.Errorf(codes.Unimplemented, "levels service not yet implemented - should call Levels service")
}

