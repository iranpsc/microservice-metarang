package service

import (
	"context"
	"fmt"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

type UserService interface {
	GetUser(ctx context.Context, userID uint64) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uint64, name, email, phone string) (*models.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUser(ctx context.Context, userID uint64) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uint64, name, email, phone string) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	user.Name = name
	user.Email = email
	user.Phone = phone

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

