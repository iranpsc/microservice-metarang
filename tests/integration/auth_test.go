package integration

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "metargb/shared/pb/auth"
)

const authServiceAddr = "localhost:50051"

func TestAuthFlow(t *testing.T) {
	// Connect to auth service
	conn, err := grpc.Dial(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Register", func(t *testing.T) {
		req := &pb.RegisterRequest{
			BackUrl:  "https://example.com",
			Referral: "",
		}

		resp, err := client.Register(ctx, req)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if resp.Url == "" {
			t.Error("Expected non-empty URL")
		}

		t.Logf("Register URL: %s", resp.Url)
	})

	t.Run("Redirect", func(t *testing.T) {
		req := &pb.RedirectRequest{
			BackUrl: "https://example.com",
		}

		resp, err := client.Redirect(ctx, req)
		if err != nil {
			t.Fatalf("Redirect failed: %v", err)
		}

		if resp.Url == "" {
			t.Error("Expected non-empty URL")
		}

		t.Logf("Redirect URL: %s", resp.Url)
	})

	// Note: Callback and GetMe tests require actual OAuth flow
	// These should be tested in E2E tests with mock OAuth server
}

func TestValidateToken(t *testing.T) {
	conn, err := grpc.Dial(authServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuthServiceClient(conn)
	ctx := context.Background()

	t.Run("InvalidToken", func(t *testing.T) {
		req := &pb.ValidateTokenRequest{
			Token: "invalid-token-12345",
		}

		resp, err := client.ValidateToken(ctx, req)
		if err != nil {
			t.Logf("Expected error for invalid token: %v", err)
		}

		if resp != nil && resp.Valid {
			t.Error("Invalid token should not be valid")
		}
	})
}

