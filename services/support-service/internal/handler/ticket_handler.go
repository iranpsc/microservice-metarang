package handler

import (
	"context"
	"fmt"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/service"
	"metargb/support-service/internal/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pbCommon "metargb/shared/pb/common"
	pb "metargb/shared/pb/support"
)

type TicketHandler struct {
	pb.UnimplementedTicketServiceServer
	ticketService service.TicketService
}

func NewTicketHandler(ticketService service.TicketService) *TicketHandler {
	return &TicketHandler{
		ticketService: ticketService,
	}
}

func RegisterTicketHandler(grpcServer *grpc.Server, ticketService service.TicketService) {
	handler := NewTicketHandler(ticketService)
	pb.RegisterTicketServiceServer(grpcServer, handler)
}

func (h *TicketHandler) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.TicketResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	// Validate that either receiver_id or department is provided (matching Laravel validation)
	if req.ReceiverId == 0 && req.Department == "" {
		return nil, status.Error(codes.InvalidArgument, "either receiver_id or department is required")
	}
	if req.ReceiverId != 0 && req.Department != "" {
		return nil, status.Error(codes.InvalidArgument, "cannot specify both receiver_id and department")
	}

	var receiverID *uint64
	if req.ReceiverId > 0 {
		receiverID = &req.ReceiverId
	}

	var department *string
	if req.Department != "" {
		department = &req.Department
	}

	ticket, err := h.ticketService.CreateTicket(ctx, req.UserId, req.Title, req.Content, req.Attachment, receiverID, department)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create ticket: %v", err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) GetTickets(ctx context.Context, req *pb.GetTicketsRequest) (*pb.TicketsResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			perPage = req.Pagination.PerPage
		}
	}

	// In Laravel, 'received' parameter determines if we get tickets where user is receiver
	// The proto doesn't have this, so we'll check status_filter or always get user's sent tickets
	// For now, we'll get sent tickets by default (matching Laravel default behavior)
	received := false

	tickets, total, err := h.ticketService.GetTickets(ctx, req.UserId, page, perPage, received)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tickets: %v", err)
	}

	response := &pb.TicketsResponse{
		Tickets: make([]*pb.TicketResponse, len(tickets)),
		Pagination: &pbCommon.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       int32(total),
			LastPage:    int32((total + int(perPage) - 1) / int(perPage)),
		},
	}

	for i, ticket := range tickets {
		response.Tickets[i] = convertTicketToProto(ticket)
	}

	return response, nil
}

func (h *TicketHandler) GetTicket(ctx context.Context, req *pb.GetTicketRequest) (*pb.TicketResponse, error) {
	if req.TicketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	ticket, err := h.ticketService.GetTicket(ctx, req.TicketId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get ticket: %v", err)
	}

	if ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) UpdateTicket(ctx context.Context, req *pb.UpdateTicketRequest) (*pb.TicketResponse, error) {
	if req.TicketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}

	ticket, err := h.ticketService.UpdateTicket(ctx, req.TicketId, req.UserId, req.Title, req.Content, req.Attachment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update ticket: %v", err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) AddResponse(ctx context.Context, req *pb.AddResponseRequest) (*pb.TicketResponse, error) {
	if req.TicketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Response == "" {
		return nil, status.Error(codes.InvalidArgument, "response is required")
	}

	// We need to get the user's name - for now we'll use empty string
	// In production, this should query the user service
	userName := fmt.Sprintf("User_%d", req.UserId)

	ticket, err := h.ticketService.AddResponse(ctx, req.TicketId, req.UserId, req.Response, req.Attachment, userName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add response: %v", err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) CloseTicket(ctx context.Context, req *pb.CloseTicketRequest) (*pb.TicketResponse, error) {
	if req.TicketId == 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	ticket, err := h.ticketService.CloseTicket(ctx, req.TicketId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close ticket: %v", err)
	}

	return convertTicketToProto(ticket), nil
}

// Helper function to convert ticket model to proto response
func convertTicketToProto(ticket *models.TicketWithRelations) *pb.TicketResponse {
	response := &pb.TicketResponse{
		Id:         ticket.ID,
		Title:      ticket.Title,
		Content:    ticket.Content,
		Attachment: ticket.Attachment,
		Code:       ticket.Code,
		Status:     ticket.Status,
		Importance: ticket.Importance,
		CreatedAt:  utils.FormatJalaliDate(ticket.CreatedAt),
		UpdatedAt:  utils.FormatJalaliDate(ticket.UpdatedAt),
	}

	if ticket.Department != nil {
		response.Department = *ticket.Department
	}

	// Sender info
	response.Sender = &pbCommon.UserBasic{
		Id:   ticket.UserID,
		Code: ticket.SenderCode,
		Name: ticket.SenderName,
	}
	if ticket.SenderProfilePhoto != nil {
		response.Sender.ProfilePhoto = *ticket.SenderProfilePhoto
	}

	// Receiver info
	if ticket.ReceiverID != nil {
		response.Receiver = &pbCommon.UserBasic{
			Id: *ticket.ReceiverID,
		}
		if ticket.ReceiverName != nil {
			response.Receiver.Name = *ticket.ReceiverName
		}
		if ticket.ReceiverCode != nil {
			response.Receiver.Code = *ticket.ReceiverCode
		}
		if ticket.ReceiverProfilePhoto != nil {
			response.Receiver.ProfilePhoto = *ticket.ReceiverProfilePhoto
		}
	}

	// Responses
	response.Responses = make([]*pb.TicketResponseItem, len(ticket.Responses))
	for i, resp := range ticket.Responses {
		response.Responses[i] = &pb.TicketResponseItem{
			Id:            resp.ID,
			TicketId:      resp.TicketID,
			Response:      resp.Response,
			Attachment:    resp.Attachment,
			ResponserName: resp.ResponserName,
			ResponserId:   resp.ResponserID,
			CreatedAt:     utils.FormatJalaliDateTime(resp.CreatedAt),
		}
	}

	return response
}
