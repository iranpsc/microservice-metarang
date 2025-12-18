package handler

import (
	"context"
	"testing"

	"metargb/financial-service/internal/service"
	pb "metargb/shared/pb/financial"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockOrderService struct {
	createOrderLink string
	createOrderErr  error
	callbackURL     string
	callbackErr     error
}

func (m *mockOrderService) CreateOrder(ctx context.Context, userID uint64, amount int32, asset string) (string, error) {
	return m.createOrderLink, m.createOrderErr
}

func (m *mockOrderService) HandleCallback(ctx context.Context, orderID uint64, status int32, token int64, additionalParams map[string]string) (string, error) {
	return m.callbackURL, m.callbackErr
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name        string
		req         *pb.CreateOrderRequest
		serviceLink string
		serviceErr  error
		expectError bool
		expectCode  codes.Code
	}{
		{
			name: "successful order creation",
			req: &pb.CreateOrderRequest{
				UserId: 1,
				Amount: 10,
				Asset:  "psc",
			},
			serviceLink: "https://pec.shaparak.ir/NewIPG/?Token=12345",
			expectError: false,
		},
		{
			name: "invalid amount",
			req: &pb.CreateOrderRequest{
				UserId: 1,
				Amount: 0,
				Asset:  "psc",
			},
			expectError: true,
			expectCode:  codes.InvalidArgument,
		},
		{
			name: "service error",
			req: &pb.CreateOrderRequest{
				UserId: 1,
				Amount: 10,
				Asset:  "psc",
			},
			serviceErr:  service.ErrPaymentFailed,
			expectError: true,
			expectCode:  codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockOrderService{
				createOrderLink: tt.serviceLink,
				createOrderErr:  tt.serviceErr,
			}
			handler := NewOrderHandler(mockService)

			resp, err := handler.CreateOrder(context.Background(), tt.req)

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
				if resp.Link != tt.serviceLink {
					t.Errorf("expected link %s, got %s", tt.serviceLink, resp.Link)
				}
			}
		})
	}
}
