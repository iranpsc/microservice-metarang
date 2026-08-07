package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
)

func TestHelperService_StubPathsAndCtor(t *testing.T) {
	// Non-empty addrs exercise dialer/interceptor setup; refused connections fail fast.
	hs := service.NewHelperService("127.0.0.1:1", "127.0.0.1:1", "127.0.0.1:1")
	require.NotNil(t, hs)
	ctx := context.Background()

	pct, err := hs.GetHourlyProfitTimePercentage(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 0.0, pct)

	pct, err = hs.GetScorePercentageToNextLevel(ctx, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 0.0, pct)

	lvl, err := hs.GetUserLevel(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, lvl)

	_, err = hs.GetUserWallet(ctx, 1)
	require.Error(t, err)

	require.Error(t, hs.CreateWallet(ctx, 1))
	require.Error(t, hs.CreateUserVariables(ctx, 1))
	require.NoError(t, hs.Close())

	empty := service.NewHelperService("", "", "")
	require.NotNil(t, empty)
	_, err = empty.GetUserWallet(ctx, 1)
	require.Error(t, err)
	require.NoError(t, empty.Close())
}

func TestNewUserServiceAndResolvePhotoURL(t *testing.T) {
	users := newFakeUserRepository(map[uint64]*models.User{1: {ID: 1}})
	svc := service.NewUserService(users)
	require.NotNil(t, svc)
	u, err := svc.GetUser(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), u.ID)

	photo := service.NewProfilePhotoService(nil, nil, "https://gw")
	require.Equal(t, "https://gw/a.png", photo.ResolvePhotoURL("/a.png"))
}
