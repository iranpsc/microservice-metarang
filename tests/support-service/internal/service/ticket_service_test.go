package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/support-service/internal/models"
)

// mockTicketRepository implements TicketRepository for testing
type mockTicketRepository struct {
	tickets             map[uint64]*models.TicketWithRelations
	responses           map[uint64][]models.TicketResponse
	createCount         int
	updateCount         int
	createResponseCount int
	getByIDFunc         func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error)
	getByUserIDFunc     func(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error)
}

func newMockTicketRepository() *mockTicketRepository {
	return &mockTicketRepository{
		tickets:   make(map[uint64]*models.TicketWithRelations),
		responses: make(map[uint64][]models.TicketResponse),
	}
}

func (m *mockTicketRepository) Create(ctx context.Context, ticket *models.Ticket) (*models.Ticket, error) {
	m.createCount++
	id := uint64(len(m.tickets) + 1)
	ticket.ID = id
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	// Convert to TicketWithRelations for storage
	twr := &models.TicketWithRelations{
		Ticket:     *ticket,
		SenderName: "Test User",
		SenderCode: "hm-1234567",
	}
	m.tickets[id] = twr
	return ticket, nil
}

func (m *mockTicketRepository) GetByID(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, ticketID)
	}
	ticket, exists := m.tickets[ticketID]
	if !exists {
		return nil, nil
	}
	// Copy responses if any
	if responses, ok := m.responses[ticketID]; ok {
		ticket.Responses = responses
	}
	return ticket, nil
}

func (m *mockTicketRepository) GetByUserID(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(ctx, userID, page, perPage, received)
	}
	var result []*models.TicketWithRelations
	for _, ticket := range m.tickets {
		if received {
			if ticket.ReceiverID != nil && *ticket.ReceiverID == userID {
				result = append(result, ticket)
			}
		} else {
			if ticket.UserID == userID {
				result = append(result, ticket)
			}
		}
	}
	return result, len(result), nil
}

func (m *mockTicketRepository) Update(ctx context.Context, ticket *models.Ticket) error {
	m.updateCount++
	if existing, ok := m.tickets[ticket.ID]; ok {
		existing.Ticket = *ticket
		existing.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("ticket not found")
}

func (m *mockTicketRepository) UpdateStatus(ctx context.Context, ticketID uint64, status int32) error {
	if ticket, ok := m.tickets[ticketID]; ok {
		ticket.Status = status
		ticket.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("ticket not found")
}

func (m *mockTicketRepository) GetResponsesByTicketID(ctx context.Context, ticketID uint64) ([]models.TicketResponse, error) {
	return m.responses[ticketID], nil
}

func (m *mockTicketRepository) CreateResponse(ctx context.Context, response *models.TicketResponse) (*models.TicketResponse, error) {
	m.createResponseCount++
	id := uint64(len(m.responses[response.TicketID]) + 1)
	response.ID = id
	response.CreatedAt = time.Now()
	response.UpdatedAt = time.Now()
	m.responses[response.TicketID] = append(m.responses[response.TicketID], *response)
	return response, nil
}

func (m *mockTicketRepository) CheckUserOwnership(ctx context.Context, ticketID, userID uint64) (bool, error) {
	ticket, ok := m.tickets[ticketID]
	if !ok {
		return false, nil
	}
	return ticket.UserID == userID || (ticket.ReceiverID != nil && *ticket.ReceiverID == userID), nil
}

func (m *mockTicketRepository) GetTicketSenderReceiver(ctx context.Context, ticketID uint64) (senderID, receiverID uint64, err error) {
	ticket, ok := m.tickets[ticketID]
	if !ok {
		return 0, 0, errors.New("ticket not found")
	}
	senderID = ticket.UserID
	if ticket.ReceiverID != nil {
		receiverID = *ticket.ReceiverID
	}
	return senderID, receiverID, nil
}

func TestTicketService_CreateTicket(t *testing.T) {
	ctx := context.Background()
	repo := newMockTicketRepository()
	service := NewTicketService(repo, "")

	t.Run("successful creation", func(t *testing.T) {
		userID := uint64(1)
		receiverID := uint64(2)
		title := "Test Ticket"
		content := "Test Content"

		ticket, err := service.CreateTicket(ctx, userID, title, content, "", &receiverID, nil)
		if err != nil {
			t.Fatalf("CreateTicket failed: %v", err)
		}

		if ticket.Title != title {
			t.Errorf("Expected title %s, got %s", title, ticket.Title)
		}
		if ticket.Status != models.TicketStatusNew {
			t.Errorf("Expected status %d, got %d", models.TicketStatusNew, ticket.Status)
		}
		if ticket.Code == 0 {
			t.Error("Expected code to be generated")
		}
	})

	t.Run("creation with department", func(t *testing.T) {
		userID := uint64(1)
		department := models.DeptTechnicalSupport
		title := "Department Ticket"
		content := "Department Content"

		ticket, err := service.CreateTicket(ctx, userID, title, content, "", nil, &department)
		if err != nil {
			t.Fatalf("CreateTicket failed: %v", err)
		}

		if ticket.Department == nil || *ticket.Department != department {
			t.Errorf("Expected department %s, got %v", department, ticket.Department)
		}
	})
}

func TestTicketService_GetTickets(t *testing.T) {
	ctx := context.Background()
	repo := newMockTicketRepository()
	service := NewTicketService(repo, "")

	// Create test tickets
	userID := uint64(1)
	receiverID := uint64(2)
	_, _ = service.CreateTicket(ctx, userID, "Ticket 1", "Content 1", "", &receiverID, nil)
	_, _ = service.CreateTicket(ctx, userID, "Ticket 2", "Content 2", "", nil, nil)

	t.Run("get sent tickets", func(t *testing.T) {
		tickets, total, err := service.GetTickets(ctx, userID, 1, 10, false)
		if err != nil {
			t.Fatalf("GetTickets failed: %v", err)
		}

		if len(tickets) != 2 {
			t.Errorf("Expected 2 tickets, got %d", len(tickets))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})
}

func TestTicketService_AddResponse(t *testing.T) {
	ctx := context.Background()
	repo := newMockTicketRepository()
	service := NewTicketService(repo, "")

	userID := uint64(1)
	receiverID := uint64(2)
	ticket, _ := service.CreateTicket(ctx, userID, "Test", "Content", "", &receiverID, nil)

	t.Run("successful response", func(t *testing.T) {
		response := "Test Response"
		updatedTicket, err := service.AddResponse(ctx, ticket.ID, receiverID, response, "", "Receiver")
		if err != nil {
			t.Fatalf("AddResponse failed: %v", err)
		}

		if updatedTicket.Status != models.TicketStatusAnswered {
			t.Errorf("Expected status %d, got %d", models.TicketStatusAnswered, updatedTicket.Status)
		}
		if len(updatedTicket.Responses) != 1 {
			t.Errorf("Expected 1 response, got %d", len(updatedTicket.Responses))
		}
	})

	t.Run("response to closed ticket fails", func(t *testing.T) {
		// Close ticket first
		_, _ = service.CloseTicket(ctx, ticket.ID, userID)

		_, err := service.AddResponse(ctx, ticket.ID, receiverID, "Response", "", "Receiver")
		if err == nil {
			t.Error("Expected error when responding to closed ticket")
		}
	})
}

func TestTicketService_CloseTicket(t *testing.T) {
	ctx := context.Background()
	repo := newMockTicketRepository()
	service := NewTicketService(repo, "")

	userID := uint64(1)
	receiverID := uint64(2)
	ticket, _ := service.CreateTicket(ctx, userID, "Test", "Content", "", &receiverID, nil)

	t.Run("successful close", func(t *testing.T) {
		closedTicket, err := service.CloseTicket(ctx, ticket.ID, userID)
		if err != nil {
			t.Fatalf("CloseTicket failed: %v", err)
		}

		if closedTicket.Status != models.TicketStatusClosed {
			t.Errorf("Expected status %d, got %d", models.TicketStatusClosed, closedTicket.Status)
		}
	})

	t.Run("close by non-sender fails", func(t *testing.T) {
		ticket2, _ := service.CreateTicket(ctx, userID, "Test 2", "Content", "", &receiverID, nil)

		err := service.CheckAuthorization(ctx, ticket2.ID, receiverID, "close")
		if err == nil {
			t.Error("Expected error when non-sender tries to close")
		}
	})
}
