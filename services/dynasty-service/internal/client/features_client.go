package client

import (
	"context"
	"fmt"
	"time"

	pb "metarang/shared/pb/features"
	grpcutil "metarang/shared/pkg/grpc"

	"google.golang.org/grpc"
)

// FeaturesClient wraps gRPC client for Features Service
type FeaturesClient struct {
	featureClient pb.FeatureServiceClient
	conn          *grpc.ClientConn
}

// NewFeaturesClient creates a new Features Service client
func NewFeaturesClient(address string) (*FeaturesClient, error) {
	conn, err := grpcutil.DialContextWithTimeout(address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to features service at %s: %w", address, err)
	}

	return &FeaturesClient{
		featureClient: pb.NewFeatureServiceClient(conn),
		conn:          conn,
	}, nil
}

// NewFeaturesClientFromGRPC builds a FeaturesClient from an existing gRPC stub (used in tests).
func NewFeaturesClientFromGRPC(featureClient pb.FeatureServiceClient, conn *grpc.ClientConn) *FeaturesClient {
	return &FeaturesClient{
		featureClient: featureClient,
		conn:          conn,
	}
}

// Close closes the gRPC connection
func (c *FeaturesClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetFeature retrieves feature information by ID
func (c *FeaturesClient) GetFeature(ctx context.Context, featureID uint64) (*pb.Feature, error) {
	req := &pb.GetFeatureRequest{
		FeatureId: featureID,
	}

	resp, err := c.featureClient.GetFeature(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	return resp.Feature, nil
}

// GetMyFeatures retrieves all features owned by a user
func (c *FeaturesClient) GetMyFeatures(ctx context.Context, userID uint64) ([]*pb.Feature, error) {
	req := &pb.GetMyFeaturesRequest{
		UserId: userID,
	}

	resp, err := c.featureClient.GetMyFeatures(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user features: %w", err)
	}

	return resp.Features, nil
}

// ListMyFeatures retrieves user features with pagination
func (c *FeaturesClient) ListMyFeatures(ctx context.Context, userID uint64, page int32) ([]*pb.Feature, error) {
	req := &pb.ListMyFeaturesRequest{
		UserId: userID,
		Page:   page,
	}

	resp, err := c.featureClient.ListMyFeatures(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list user features: %w", err)
	}

	return resp.Data, nil
}
