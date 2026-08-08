package client

import (
	"context"
	"fmt"
	"time"

	pb "metarang/shared/pb/levels"
	grpcutil "metarang/shared/pkg/grpc"

	"google.golang.org/grpc"
)

// LevelsClient wraps gRPC client for Levels Service.
type LevelsClient struct {
	levelClient pb.LevelServiceClient
	conn        *grpc.ClientConn
}

// NewLevelsClient creates a new Levels Service client.
func NewLevelsClient(address string) (*LevelsClient, error) {
	conn, err := grpcutil.DialContextWithTimeout(address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to levels service at %s: %w", address, err)
	}

	return &LevelsClient{
		levelClient: pb.NewLevelServiceClient(conn),
		conn:        conn,
	}, nil
}

// NewLevelsClientFromGRPC builds a LevelsClient from an existing stub (tests).
func NewLevelsClientFromGRPC(levelClient pb.LevelServiceClient, conn *grpc.ClientConn) *LevelsClient {
	return &LevelsClient{
		levelClient: levelClient,
		conn:        conn,
	}
}

// Close closes the gRPC connection.
func (c *LevelsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetUserLevel fetches the user's level ladder from levels-service.
func (c *LevelsClient) GetUserLevel(ctx context.Context, userID uint64) (*pb.UserLevelResponse, error) {
	resp, err := c.levelClient.GetUserLevel(ctx, &pb.GetUserLevelRequest{UserId: userID})
	if err != nil {
		return nil, fmt.Errorf("failed to get user level: %w", err)
	}
	return resp, nil
}

// GetLevelGem fetches gem details for a level.
func (c *LevelsClient) GetLevelGem(ctx context.Context, levelID uint64) (*pb.LevelGem, error) {
	resp, err := c.levelClient.GetLevelGem(ctx, &pb.GetLevelGemRequest{LevelId: levelID})
	if err != nil {
		return nil, fmt.Errorf("failed to get level gem: %w", err)
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Gem, nil
}
