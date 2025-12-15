package handler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/support"
	"metargb/support-service/internal/models"
)

// mockTicketService implements TicketService for testing
type mockTicketService struct {
	createTicketFunc func(ctx context.Context, userID uint64, title, content, attachment string, receiverID *uint64, department *string) (*models.TicketWithRelations, error)
	getTicketsFunc   func(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error)
	getTicketFunc    func(ctx context.Context, ticketID, userID uint64) (*models.TicketWithRelations, error)
}

func (m *mockTicketService) CreateTicket(ctx context.Context, userID uint64, title, content, attachment string, receiverID *uint64, department *string) (*models.TicketWithRelations, error) {
	if m.createTicketFunc != nil {
		return m.createTicketFunc(ctx, userID, title, content, attachment, receiverID, department)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketService) GetTickets(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error) {
	if m.getTicketsFunc != nil {
		return m.getTicketsFunc(ctx, userID, page, perPage, received)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *mockTicketService) GetTicket(ctx context.Context, ticketID, userID uint64) (*models.TicketWithRelations, error) {
	if m.getTicketFunc != nil {
		return m.getTicketFunc(ctx, ticketID, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTicketService) UpdateTicket(ctx context.Context, ticketID, userID uint64, title, content, attachment string) (*models.TicketWithRelations, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTicketService) AddResponse(ctx context.Context, ticketID, userID uint64, response, attachment, userName string) (*models.TicketWithRelations, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTicketService) CloseTicket(ctx context.Context, ticketID, userID uint64) (*models.TicketWithRelations, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTicketService) CheckAuthorization(ctx context.Context, ticketID, userID uint64, action string) error {
	return nil
}

func TestTicketHandler_CreateTicket(t *testing.T) {
	ctx := context.Background()

	t.Run("missing user_id", func(t *testing.T) {
		mockService := &mockTicketService{}
		handler := NewTicketHandler(mockService)

		req := &pb.CreateTicketRequest{
			Title:   "Test",
			Content: "Content",
		}

		_, err := handler.CreateTicket(ctx, req)
		if err == nil {
			t.Error("Expected error for missing user_id")
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", status.Code(err))
		}
	})

	t.Run("missing title", func(t *testing.T) {
		mockService := &mockTicketService{}
		handler := NewTicketHandler(mockService)

		req := &pb.CreateTicketRequest{
			UserId:  1,
			Content: "Content",
		}

		_, err := handler.CreateTicket(ctx, req)
		if err == nil {
			t.Error("Expected error for missing title")
		}
	})

	t.Run("successful creation", func(t *testing.T) {
		mockService := &mockTicketService{}
		mockService.createTicketFunc = func(ctx context.Context, userID uint64, title, content, attachment string, receiverID *uint64, department *string) (*models.TicketWithRelations, error) {
			return &models.TicketWithRelations{
				Ticket: models.Ticket{
					ID:      1,
					Title:   title,
					Content: content,
					UserID:  userID,
					Status:  models.TicketStatusNew,
					Code:    123456,
				},
				SenderName: "Test User",
				SenderCode: "hm-1234567",
			}, nil
		}

		handler := NewTicketHandler(mockService)

		req := &pb.CreateTicketRequest{
			UserId:     1,
			Title:      "Test Ticket",
			Content:    "Test Content",
			ReceiverId: 2,
		}

		resp, err := handler.CreateTicket(ctx, req)
		if err != nil {
			t.Fatalf("CreateTicket failed: %v", err)
		}

		if resp.Id != 1 {
			t.Errorf("Expected ID 1, got %d", resp.Id)
		}
		if resp.Title != "Test Ticket" {
			t.Errorf("Expected title 'Test Ticket', got %s", resp.Title)
		}
	})
}

func TestTicketHandler_GetTickets(t *testing.T) {
	ctx := context.Background()

	t.Run("missing user_id", func(t *testing.T) {
		mockService := &mockTicketService{}
		handler := NewTicketHandler(mockService)

		req := &pb.GetTicketsRequest{}

		_, err := handler.GetTickets(ctx, req)
		if err == nil {
			t.Error("Expected error for missing user_id")
		}
	})

	t.Run("successful get", func(t *testing.T) {
		mockService := &mockTicketService{}
		mockService.getTicketsFunc = func(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error) {
			return []*models.TicketWithRelations{
				{
					Ticket: models.Ticket{
						ID:      1,
						Title:   "Ticket 1",
						Content: "Content 1",
						UserID:  userID,
					},
					SenderName: "User",
					SenderCode: "hm-1234567",
				},
			}, 1, nil
		}

		handler := NewTicketHandler(mockService)

		req := &pb.GetTicketsRequest{
			UserId: 1,
		}

		resp, err := handler.GetTickets(ctx, req)
		if err != nil {
			t.Fatalf("GetTickets failed: %v", err)
		}

		if len(resp.Tickets) != 1 {
			t.Errorf("Expected 1 ticket, got %d", len(resp.Tickets))
		}
	})
}
