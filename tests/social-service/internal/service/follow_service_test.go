package service

import (
	"context"
	"errors"
	"testing"

	"metargb/social-service/internal/repository"
)

// Mock repositories
type mockFollowRepository struct {
	createFunc       func(ctx context.Context, followerID, followingID uint64) error
	deleteFunc       func(ctx context.Context, followerID, followingID uint64) error
	existsFunc       func(ctx context.Context, followerID, followingID uint64) (bool, error)
	getFollowersFunc func(ctx context.Context, userID uint64) ([]uint64, error)
	getFollowingFunc func(ctx context.Context, userID uint64) ([]uint64, error)
}

func (m *mockFollowRepository) Create(ctx context.Context, followerID, followingID uint64) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, followerID, followingID)
	}
	return errors.New("not implemented")
}

func (m *mockFollowRepository) Delete(ctx context.Context, followerID, followingID uint64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, followerID, followingID)
	}
	return errors.New("not implemented")
}

func (m *mockFollowRepository) Exists(ctx context.Context, followerID, followingID uint64) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, followerID, followingID)
	}
	return false, errors.New("not implemented")
}

func (m *mockFollowRepository) GetFollowers(ctx context.Context, userID uint64) ([]uint64, error) {
	if m.getFollowersFunc != nil {
		return m.getFollowersFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFollowRepository) GetFollowing(ctx context.Context, userID uint64) ([]uint64, error) {
	if m.getFollowingFunc != nil {
		return m.getFollowingFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

type mockUserRepository struct {
	getUserBasicInfoFunc func(ctx context.Context, userID uint64) (*repository.UserBasicInfo, error)
	getUserLevelFunc     func(ctx context.Context, userID uint64) (string, error)
	getProfilePhotosFunc func(ctx context.Context, userID uint64) ([]string, error)
	isUserOnlineFunc     func(ctx context.Context, userID uint64) (bool, error)
}

func (m *mockUserRepository) GetUserBasicInfo(ctx context.Context, userID uint64) (*repository.UserBasicInfo, error) {
	if m.getUserBasicInfoFunc != nil {
		return m.getUserBasicInfoFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) GetUserLevel(ctx context.Context, userID uint64) (string, error) {
	if m.getUserLevelFunc != nil {
		return m.getUserLevelFunc(ctx, userID)
	}
	return "", errors.New("not implemented")
}

func (m *mockUserRepository) GetProfilePhotos(ctx context.Context, userID uint64) ([]string, error) {
	if m.getProfilePhotosFunc != nil {
		return m.getProfilePhotosFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) IsUserOnline(ctx context.Context, userID uint64) (bool, error) {
	if m.isUserOnlineFunc != nil {
		return m.isUserOnlineFunc(ctx, userID)
	}
	return false, errors.New("not implemented")
}

func TestFollowService_Follow(t *testing.T) {
	ctx := context.Background()

	t.Run("successful follow", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		followRepo.existsFunc = func(ctx context.Context, followerID, followingID uint64) (bool, error) {
			return false, nil // Not already following
		}
		followRepo.createFunc = func(ctx context.Context, followerID, followingID uint64) error {
			return nil
		}

		userRepo := &mockUserRepository{}

		service := NewFollowService(followRepo, userRepo)
		err := service.Follow(ctx, 1, 2)

		if err != nil {
			t.Fatalf("Follow failed: %v", err)
		}
	})

	t.Run("cannot follow self", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		userRepo := &mockUserRepository{}

		service := NewFollowService(followRepo, userRepo)
		err := service.Follow(ctx, 1, 1)

		if err == nil {
			t.Fatal("Expected error when following self")
		}
		if err != ErrCannotFollowSelf {
			t.Fatalf("Expected ErrCannotFollowSelf, got: %v", err)
		}
	})

	t.Run("already following", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		followRepo.existsFunc = func(ctx context.Context, followerID, followingID uint64) (bool, error) {
			return true, nil // Already following
		}

		userRepo := &mockUserRepository{}

		service := NewFollowService(followRepo, userRepo)
		err := service.Follow(ctx, 1, 2)

		if err == nil {
			t.Fatal("Expected error when already following")
		}
		if err != ErrAlreadyFollowing {
			t.Fatalf("Expected ErrAlreadyFollowing, got: %v", err)
		}
	})
}

func TestFollowService_GetFollowers(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get followers", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		followRepo.getFollowersFunc = func(ctx context.Context, userID uint64) ([]uint64, error) {
			return []uint64{2, 3}, nil
		}

		userRepo := &mockUserRepository{}
		userRepo.getUserBasicInfoFunc = func(ctx context.Context, userID uint64) (*repository.UserBasicInfo, error) {
			return &repository.UserBasicInfo{
				ID:   userID,
				Name: "User",
				Code: "USER",
			}, nil
		}
		userRepo.getUserLevelFunc = func(ctx context.Context, userID uint64) (string, error) {
			return "level1", nil
		}
		userRepo.getProfilePhotosFunc = func(ctx context.Context, userID uint64) ([]string, error) {
			return []string{"photo1.jpg"}, nil
		}
		userRepo.isUserOnlineFunc = func(ctx context.Context, userID uint64) (bool, error) {
			return true, nil
		}

		service := NewFollowService(followRepo, userRepo)
		followers, err := service.GetFollowers(ctx, 1)

		if err != nil {
			t.Fatalf("GetFollowers failed: %v", err)
		}
		if len(followers) != 2 {
			t.Fatalf("Expected 2 followers, got %d", len(followers))
		}
		if followers[0].ID != 2 {
			t.Fatalf("Expected follower ID 2, got %d", followers[0].ID)
		}
	})
}

func TestFollowService_Unfollow(t *testing.T) {
	ctx := context.Background()

	t.Run("successful unfollow", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		followRepo.deleteFunc = func(ctx context.Context, followerID, followingID uint64) error {
			return nil
		}

		userRepo := &mockUserRepository{}

		service := NewFollowService(followRepo, userRepo)
		err := service.Unfollow(ctx, 1, 2)

		if err != nil {
			t.Fatalf("Unfollow failed: %v", err)
		}
	})
}

func TestFollowService_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("successful remove", func(t *testing.T) {
		followRepo := &mockFollowRepository{}
		followRepo.deleteFunc = func(ctx context.Context, followerID, followingID uint64) error {
			return nil
		}

		userRepo := &mockUserRepository{}

		service := NewFollowService(followRepo, userRepo)
		err := service.Remove(ctx, 1, 2)

		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})
}
