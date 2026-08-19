package client

import (
	"context"
	"fmt"
	"time"

	pb "metarang/shared/pb/levels"
	grpcutil "metarang/shared/pkg/grpc"

	"google.golang.org/grpc"
)

// LevelsClient wraps gRPC clients for Levels Service
type LevelsClient interface {
	// RecordFollower asks levels-service to update the user's followers_count
	RecordFollower(ctx context.Context, userID uint64) error
	Close() error
}

type levelsClient struct {
	activityClient pb.ActivityServiceClient
	conn           *grpc.ClientConn
}

// NewLevelsClient creates a new Levels Service client
func NewLevelsClient(address string) (LevelsClient, error) {
	conn, err := grpcutil.DialContextWithTimeout(address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to levels service at %s: %w", address, err)
	}

	return &levelsClient{
		activityClient: pb.NewActivityServiceClient(conn),
		conn:           conn,
	}, nil
}

// NewLevelsClientFromGRPC builds a LevelsClient from an existing gRPC stub (used in tests).
func NewLevelsClientFromGRPC(activityClient pb.ActivityServiceClient) LevelsClient {
	return &levelsClient{
		activityClient: activityClient,
	}
}

// Close closes the gRPC connection
func (c *levelsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// RecordFollower records a follower change for the given user
func (c *levelsClient) RecordFollower(ctx context.Context, userID uint64) error {
	resp, err := c.activityClient.RecordFollower(ctx, &pb.RecordFollowerRequest{UserId: userID})
	if err != nil {
		return fmt.Errorf("failed to record follower: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("record follower failed for user %d", userID)
	}
	return nil
}
