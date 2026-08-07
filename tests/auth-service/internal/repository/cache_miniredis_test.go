package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"metarang/auth-service/internal/repository"
)

func TestCacheRepository_Miniredis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()
	repo := repository.NewCacheRepository(client)
	ctx := context.Background()
	ttl := time.Minute

	require.NoError(t, repo.SetState(ctx, "st1", ttl))
	ok, err := repo.GetState(ctx, "st1")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.GetState(ctx, "st1")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, repo.SetRedirectTo(ctx, "st2", "https://r", ttl))
	v, err := repo.GetRedirectTo(ctx, "st2")
	require.NoError(t, err)
	require.Equal(t, "https://r", v)

	require.NoError(t, repo.SetBackURL(ctx, "st3", "https://b", ttl))
	v, err = repo.GetBackURL(ctx, "st3")
	require.NoError(t, err)
	require.Equal(t, "https://b", v)

	ok, err = repo.TryAcquireAccountSecurityVerificationSlot(ctx, 7, time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.TryAcquireAccountSecurityVerificationSlot(ctx, 7, time.Minute)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, repo.SetWeb3LinkNonce(ctx, 1, "0xabc", "n1", ttl))
	n, err := repo.PullWeb3LinkNonce(ctx, 1, "0xabc")
	require.NoError(t, err)
	require.Equal(t, "n1", n)
	n, err = repo.PullWeb3LinkNonce(ctx, 1, "0xabc")
	require.NoError(t, err)
	require.Equal(t, "", n)

	require.NoError(t, repo.SetWeb3SecurityNonce(ctx, 1, "0xabc", "n2", ttl))
	n, err = repo.PullWeb3SecurityNonce(ctx, 1, "0xabc")
	require.NoError(t, err)
	require.Equal(t, "n2", n)
}
