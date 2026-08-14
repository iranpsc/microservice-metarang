package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/repository"
	"metarang/social-service/internal/service"
	"metarang/social-service/internal/testutil"
)

func TestFollowService_GetFollowers_RepoError(t *testing.T) {
	fr := &testutil.MockFollowRepository{
		GetFollowersFunc: func(context.Context, uint64) ([]uint64, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, nil, nil)
	_, err := svc.GetFollowers(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get followers")
}

func TestFollowService_GetFollowing_RepoError(t *testing.T) {
	fr := &testutil.MockFollowRepository{
		GetFollowingFunc: func(context.Context, uint64) ([]uint64, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, nil, nil)
	_, err := svc.GetFollowing(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get following")
}

func TestFollowService_EmptyLists(t *testing.T) {
	fr := &testutil.MockFollowRepository{
		GetFollowersFunc: func(context.Context, uint64) ([]uint64, error) {
			return []uint64{}, nil
		},
		GetFollowingFunc: func(context.Context, uint64) ([]uint64, error) {
			return []uint64{}, nil
		},
	}
	svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, nil, nil)

	followers, err := svc.GetFollowers(context.Background(), 1)
	require.NoError(t, err)
	require.Empty(t, followers)

	following, err := svc.GetFollowing(context.Background(), 1)
	require.NoError(t, err)
	require.Empty(t, following)
}

func TestFollowService_GetFollowers_SkipsNilAndUserInfoError(t *testing.T) {
	fr := &testutil.MockFollowRepository{
		GetFollowersFunc: func(context.Context, uint64) ([]uint64, error) {
			return []uint64{10, 11, 12}, nil
		},
		ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
			return false, nil
		},
	}
	ur := &testutil.MockUserRepository{
		GetUserBasicInfoFunc: func(_ context.Context, userID uint64) (*repository.UserBasicInfo, error) {
			switch userID {
			case 10:
				return nil, errors.New("user lookup failed")
			case 11:
				return nil, nil
			default:
				return &repository.UserBasicInfo{ID: 12, Name: "Kept", Code: "k", ProfilePhoto: "http://kept"}, nil
			}
		},
	}
	svc := service.NewFollowService(fr, ur, nil, nil)
	list, err := svc.GetFollowers(context.Background(), 99)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, uint64(12), list[0].ID)
	require.Equal(t, "http://kept", list[0].ProfilePhoto)
}

func TestFollowService_BuildFollowResource_OptionalLevelOnlineAndExistsErrors(t *testing.T) {
	t.Run("level and online errors continue", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			GetFollowersFunc: func(context.Context, uint64) ([]uint64, error) {
				return []uint64{5}, nil
			},
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, nil
			},
		}
		ur := &testutil.MockUserRepository{
			GetUserBasicInfoFunc: func(context.Context, uint64) (*repository.UserBasicInfo, error) {
				return &repository.UserBasicInfo{ID: 5, Name: "U", Code: "c", ProfilePhoto: "http://p"}, nil
			},
			GetUserLevelFunc: func(context.Context, uint64) (string, error) {
				return "", errors.New("level down")
			},
			IsUserOnlineFunc: func(context.Context, uint64) (bool, error) {
				return false, errors.New("online down")
			},
		}
		svc := service.NewFollowService(fr, ur, nil, nil)
		list, err := svc.GetFollowers(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "", list[0].Level)
		require.False(t, list[0].Online)
		require.Equal(t, "http://p", list[0].ProfilePhoto)
	})

	t.Run("following exists error skips user", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			GetFollowingFunc: func(context.Context, uint64) ([]uint64, error) {
				return []uint64{8, 9}, nil
			},
			ExistsFunc: func(_ context.Context, followerID, followingID uint64) (bool, error) {
				if followingID == 8 {
					return false, errors.New("exists following")
				}
				return false, nil
			},
		}
		ur := &testutil.MockUserRepository{
			GetUserBasicInfoFunc: func(_ context.Context, userID uint64) (*repository.UserBasicInfo, error) {
				return &repository.UserBasicInfo{ID: userID, Name: "N", Code: "c"}, nil
			},
		}
		svc := service.NewFollowService(fr, ur, nil, nil)
		list, err := svc.GetFollowing(context.Background(), 1)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, uint64(9), list[0].ID)
	})

	t.Run("follower exists error skips user", func(t *testing.T) {
		var existsCalls int
		fr := &testutil.MockFollowRepository{
			GetFollowersFunc: func(context.Context, uint64) ([]uint64, error) {
				return []uint64{4}, nil
			},
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				existsCalls++
				if existsCalls == 2 {
					return false, errors.New("exists follower")
				}
				return false, nil
			},
		}
		ur := &testutil.MockUserRepository{
			GetUserBasicInfoFunc: func(context.Context, uint64) (*repository.UserBasicInfo, error) {
				return &repository.UserBasicInfo{ID: 4, Name: "N", Code: "c"}, nil
			},
		}
		svc := service.NewFollowService(fr, ur, nil, nil)
		list, err := svc.GetFollowers(context.Background(), 1)
		require.NoError(t, err)
		require.Empty(t, list)
	})
}

func TestFollowService_Follow_ErrorPaths(t *testing.T) {
	t.Run("exists error", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, errors.New("exists failed")
			},
		}
		svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, nil, nil)
		err := svc.Follow(context.Background(), 1, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to check follow relationship")
	})

	t.Run("auth client nil", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, nil
			},
		}
		svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, nil, nil)
		err := svc.Follow(context.Background(), 1, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "auth service client is not configured")
	})

	t.Run("can follow error", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, nil
			},
		}
		auth := &testutil.MockAuthClient{
			CanFollowFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, errors.New("auth rpc")
			},
		}
		svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, auth, nil)
		err := svc.Follow(context.Background(), 1, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to check profile limitation")
	})

	t.Run("create error", func(t *testing.T) {
		fr := &testutil.MockFollowRepository{
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, nil
			},
			CreateFunc: func(context.Context, uint64, uint64) error {
				return errors.New("insert failed")
			},
		}
		auth := &testutil.MockAuthClient{
			CanFollowFunc: func(context.Context, uint64, uint64) (bool, error) {
				return true, nil
			},
		}
		svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, auth, nil)
		err := svc.Follow(context.Background(), 1, 2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create follow relationship")
	})

	t.Run("levels record follower error still succeeds", func(t *testing.T) {
		var created bool
		fr := &testutil.MockFollowRepository{
			ExistsFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, nil
			},
			CreateFunc: func(context.Context, uint64, uint64) error {
				created = true
				return nil
			},
		}
		auth := &testutil.MockAuthClient{
			CanFollowFunc: func(context.Context, uint64, uint64) (bool, error) {
				return true, nil
			},
		}
		levels := &testutil.MockLevelsClient{
			RecordFollowerFunc: func(context.Context, uint64) error {
				return errors.New("levels down")
			},
		}
		svc := service.NewFollowService(fr, &testutil.MockUserRepository{}, auth, levels)
		require.NoError(t, svc.Follow(context.Background(), 1, 2))
		require.True(t, created)
	})
}
