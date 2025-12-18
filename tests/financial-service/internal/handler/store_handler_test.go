package handler

import (
	"context"
	"errors"
	"testing"

	"metargb/financial-service/internal/service"
	pb "metargb/shared/pb/financial"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStoreService struct {
	packages []*service.PackageResource
	err      error
}

func (m *mockStoreService) GetStorePackages(ctx context.Context, codes []string) ([]*service.PackageResource, error) {
	return m.packages, m.err
}

func TestStoreHandler_GetStorePackages(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.GetStorePackagesRequest
		packages    []*service.PackageResource
		serviceErr  error
		expectError bool
		expectCode  codes.Code
	}{
		{
			name: "successful package retrieval",
			req: &pb.GetStorePackagesRequest{
				Codes: []string{"PACK1", "PACK2"},
			},
			packages: []*service.PackageResource{
				{ID: 1, Code: "PACK1", Asset: "psc", Amount: 100, UnitPrice: 1000},
				{ID: 2, Code: "PACK2", Asset: "red", Amount: 50, UnitPrice: 2000},
			},
			expectError: false,
		},
		{
			name: "insufficient codes",
			req: &pb.GetStorePackagesRequest{
				Codes: []string{"PACK1"},
			},
			expectError: true,
			expectCode:  codes.InvalidArgument,
		},
		{
			name: "service error",
			req: &pb.GetStorePackagesRequest{
				Codes: []string{"PACK1", "PACK2"},
			},
			serviceErr:  errors.New("database error"),
			expectError: true,
			expectCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockStoreService{
				packages: tt.packages,
				err:      tt.serviceErr,
			}
			handler := NewStoreHandler(mockService)

			resp, err := handler.GetStorePackages(context.Background(), tt.req)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.expectCode != codes.OK {
					st, ok := status.FromError(err)
					if !ok || st.Code() != tt.expectCode {
						t.Errorf("expected error code %v, got %v", tt.expectCode, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(resp.Packages) != len(tt.packages) {
					t.Errorf("expected %d packages, got %d", len(tt.packages), len(resp.Packages))
				}
			}
		})
	}
}
