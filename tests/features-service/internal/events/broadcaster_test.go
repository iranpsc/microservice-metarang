package events_test

import (
	"context"
	"testing"
	"time"

	"metarang/features-service/internal/events"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisBroadcaster_ConnectionFailure(t *testing.T) {
	_, err := events.NewRedisBroadcaster("127.0.0.1:1", "", "feature-events")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

func TestBroadcastFeatureStatusChanged_Error(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond})
	b := events.NewRedisBroadcasterFromClient(rdb, "feature-events")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := b.BroadcastFeatureStatusChanged(ctx, 9, "a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish")
}

func TestClose_NilAndClient(t *testing.T) {
	var nilBroadcaster events.RedisBroadcaster
	require.NoError(t, nilBroadcaster.Close())

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	b := events.NewRedisBroadcasterFromClient(rdb, "ch")
	require.NoError(t, b.Close())
}
