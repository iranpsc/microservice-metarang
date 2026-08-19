package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/tests/internal/testutil"
)

func TestTicketService_CreateTicket_RepoAndGetByIDErrors(t *testing.T) {
	repo := &testutil.MockTicketRepo{
		CreateFunc: func(ctx context.Context, ticket *models.Ticket) (*models.Ticket, error) {
			return nil, errors.New("insert failed")
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	_, err := svc.CreateTicket(context.Background(), 1, "t", "c", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create ticket") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		CreateFunc: func(ctx context.Context, ticket *models.Ticket) (*models.Ticket, error) {
			ticket.ID = 3
			return ticket, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return nil, errors.New("lookup failed")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.CreateTicket(context.Background(), 1, "t", "c", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to get created ticket") {
		t.Fatalf("err=%v", err)
	}
}

func TestTicketService_CreateTicket_WithReceiverReturnsWithoutWaitingNotification(t *testing.T) {
	rid := uint64(20)
	repo := &testutil.MockTicketRepo{
		CreateFunc: func(ctx context.Context, ticket *models.Ticket) (*models.Ticket, error) {
			if ticket.ReceiverID == nil || *ticket.ReceiverID != rid {
				t.Fatalf("receiver=%v", ticket.ReceiverID)
			}
			ticket.ID = 8
			return ticket, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			tk := ticketFull(ticketID, 10, models.TicketStatusNew)
			tk.ReceiverID = &rid
			return tk, nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	got, err := svc.CreateTicket(context.Background(), 10, "t", "c", "", &rid, nil)
	if err != nil || got.ID != 8 {
		t.Fatalf("err=%v got=%+v", err, got)
	}
}

func TestTicketService_GetTickets_PaginationDefaults(t *testing.T) {
	var gotPage, gotPer int32
	repo := &testutil.MockTicketRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64, page, perPage int32, received bool) ([]*models.TicketWithRelations, int, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	_, _, err := svc.GetTickets(context.Background(), 1, 0, -5, false)
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != 1 || gotPer != 10 {
		t.Fatalf("page=%d per=%d", gotPage, gotPer)
	}
}

func TestTicketService_GetTicket_ViewAllowedForReceiverAndSenderReceiverError(t *testing.T) {
	repo := &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return ticketFull(ticketID, 1, models.TicketStatusNew), nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	got, err := svc.GetTicket(context.Background(), 9, 2)
	if err != nil || got == nil {
		t.Fatalf("receiver view err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 0, 0, errors.New("missing ticket")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.GetTicket(context.Background(), 9, 1)
	if err == nil || !strings.Contains(err.Error(), "failed to get ticket info") {
		t.Fatalf("err=%v", err)
	}
}

func TestTicketService_UpdateTicket_GetByIDUpdateErrorsAndAttachmentRules(t *testing.T) {
	tk := ticketFull(5, 7, models.TicketStatusAnswered)
	tk.Attachment = "old.png"
	var updated *models.Ticket
	repo := &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 7, 8, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return tk, nil
		},
		UpdateFunc: func(ctx context.Context, ticket *models.Ticket) error {
			cp := *ticket
			updated = &cp
			tk.Title = ticket.Title
			tk.Content = ticket.Content
			tk.Attachment = ticket.Attachment
			tk.Status = ticket.Status
			return nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	_, err := svc.UpdateTicket(context.Background(), 5, 7, "x", "y", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated == nil || updated.Attachment != "old.png" || updated.Status != models.TicketStatusNew {
		t.Fatalf("empty attachment must keep old file and reset status: %+v", updated)
	}

	_, err = svc.UpdateTicket(context.Background(), 5, 7, "x", "y", "new.png")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Attachment != "new.png" {
		t.Fatalf("non-empty attachment must overwrite: %+v", updated)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 7, 8, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return nil, errors.New("db")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.UpdateTicket(context.Background(), 5, 7, "x", "y", "")
	if err == nil || !strings.Contains(err.Error(), "failed to get ticket") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 7, 8, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return ticketFull(5, 7, models.TicketStatusNew), nil
		},
		UpdateFunc: func(ctx context.Context, ticket *models.Ticket) error {
			return errors.New("write failed")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.UpdateTicket(context.Background(), 5, 7, "x", "y", "")
	if err == nil || !strings.Contains(err.Error(), "failed to update ticket") {
		t.Fatalf("err=%v", err)
	}
}

func TestTicketService_AddResponse_UnauthorizedGetByIDAndRepoErrors(t *testing.T) {
	repo := &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	_, err := svc.AddResponse(context.Background(), 3, 99, "hi", "", "X")
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return nil, errors.New("lookup")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.AddResponse(context.Background(), 3, 1, "hi", "", "X")
	if err == nil || !strings.Contains(err.Error(), "failed to get ticket") {
		t.Fatalf("err=%v", err)
	}

	tk := ticketFull(3, 1, models.TicketStatusNew)
	calls := 0
	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return tk, nil
		},
		CreateResponseFunc: func(ctx context.Context, response *models.TicketResponse) (*models.TicketResponse, error) {
			return nil, errors.New("insert response")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.AddResponse(context.Background(), 3, 1, "hi", "", "X")
	if err == nil || !strings.Contains(err.Error(), "failed to create response") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return tk, nil
		},
		CreateResponseFunc: func(ctx context.Context, response *models.TicketResponse) (*models.TicketResponse, error) {
			return response, nil
		},
		UpdateStatusFunc: func(ctx context.Context, ticketID uint64, status int32) error {
			return errors.New("status")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.AddResponse(context.Background(), 3, 1, "hi", "", "X")
	if err == nil || !strings.Contains(err.Error(), "failed to update ticket status") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			calls++
			if calls == 1 {
				return tk, nil
			}
			return nil, errors.New("reload")
		},
		CreateResponseFunc: func(ctx context.Context, response *models.TicketResponse) (*models.TicketResponse, error) {
			return response, nil
		},
		UpdateStatusFunc: func(ctx context.Context, ticketID uint64, status int32) error {
			return nil
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.AddResponse(context.Background(), 3, 1, "hi", "", "X")
	if err == nil || !strings.Contains(err.Error(), "failed to get updated ticket") {
		t.Fatalf("err=%v", err)
	}
}

func TestTicketService_CloseTicket_UnauthorizedGetByIDAndStatusError(t *testing.T) {
	repo := &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 9, 0, nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	_, err := svc.CloseTicket(context.Background(), 6, 1)
	if err == nil || !strings.Contains(err.Error(), "only ticket sender can close") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 9, 0, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return nil, errors.New("lookup")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.CloseTicket(context.Background(), 6, 9)
	if err == nil || !strings.Contains(err.Error(), "failed to get ticket") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 9, 0, nil
		},
		GetByIDFunc: func(ctx context.Context, ticketID uint64) (*models.TicketWithRelations, error) {
			return ticketFull(6, 9, models.TicketStatusNew), nil
		},
		UpdateStatusFunc: func(ctx context.Context, ticketID uint64, status int32) error {
			return errors.New("close failed")
		},
	}
	svc = service.NewTicketService(repo, "127.0.0.1:1")
	_, err = svc.CloseTicket(context.Background(), 6, 9)
	if err == nil || !strings.Contains(err.Error(), "failed to close ticket") {
		t.Fatalf("err=%v", err)
	}
}

func TestTicketService_CheckAuthorization_UnknownActionAndUpdateSenderOnly(t *testing.T) {
	repo := &testutil.MockTicketRepo{
		GetTicketSenderReceiverFunc: func(ctx context.Context, ticketID uint64) (uint64, uint64, error) {
			return 1, 2, nil
		},
	}
	svc := service.NewTicketService(repo, "127.0.0.1:1")
	err := svc.CheckAuthorization(context.Background(), 1, 1, "explode")
	if err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("err=%v", err)
	}
	if err := svc.CheckAuthorization(context.Background(), 1, 1, "update"); err != nil {
		t.Fatal(err)
	}
	err = svc.CheckAuthorization(context.Background(), 1, 2, "update")
	if err == nil || !strings.Contains(err.Error(), "only ticket sender can update") {
		t.Fatalf("err=%v", err)
	}
	if err := svc.CheckAuthorization(context.Background(), 1, 2, "respond"); err != nil {
		t.Fatal(err)
	}
	err = svc.CheckAuthorization(context.Background(), 1, 9, "view")
	if err == nil || !strings.Contains(err.Error(), "permission to view") {
		t.Fatalf("err=%v", err)
	}
}
