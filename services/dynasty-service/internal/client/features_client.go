package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "metargb/shared/pb/features"
)

// FeaturesClient wraps gRPC client for Features Service
type FeaturesClient struct {
	featureClient pb.FeatureServiceClient
	conn          *grpc.ClientConn
}

// NewFeaturesClient creates a new Features Service client
func NewFeaturesClient(address string) (*FeaturesClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to features service at %s: %w", address, err)
	}

	return &FeaturesClient{
		featureClient: pb.NewFeatureServiceClient(conn),
		conn:          conn,
	}, nil
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

