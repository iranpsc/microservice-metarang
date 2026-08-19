package handler

import (
	"context"
	"strings"

	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/internal/utils"

	pbCommon "metarang/shared/pb/common"
	pb "metarang/shared/pb/support"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func RegisterTicketHandler(grpcServer *grpc.Server, ticketService service.TicketService) *TicketHandler {
	handler := NewTicketHandler(ticketService)
	pb.RegisterTicketServiceServer(grpcServer, handler)
	return handler
}

func (h *TicketHandler) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.TicketResponse, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("user_id", req.UserId, locale),
		ValidateRequired("title", req.Title, locale),
		ValidateRequired("content", req.Content, locale),
		ValidateMaxLen("title", req.Title, 250, locale),
		ValidateMaxLen("content", req.Content, 500, locale),
	)
	if req.ReceiverId == 0 && req.Department == "" {
		validationErrors = mergeValidationErrors(validationErrors, map[string]string{
			"reciever": "Either reciever or department is required",
		})
	}
	if req.ReceiverId != 0 && req.Department != "" {
		validationErrors = mergeValidationErrors(validationErrors, map[string]string{
			"department": "Cannot specify both reciever and department",
		})
	}
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
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
		return nil, MapServiceError(err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) GetTickets(ctx context.Context, req *pb.GetTicketsRequest) (*pb.TicketsResponse, error) {
	locale := handlerLocale(ctx)
	validationErrors := ValidateRequired("user_id", req.UserId, locale)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
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

	received := req.Received

	tickets, total, err := h.ticketService.GetTickets(ctx, req.UserId, page, perPage, received)
	if err != nil {
		return nil, MapServiceError(err)
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
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("ticket_id", req.TicketId, locale),
		ValidateRequired("user_id", req.UserId, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	ticket, err := h.ticketService.GetTicket(ctx, req.TicketId, req.UserId)
	if err != nil {
		return nil, MapServiceError(err)
	}

	if ticket == nil {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) UpdateTicket(ctx context.Context, req *pb.UpdateTicketRequest) (*pb.TicketResponse, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("ticket_id", req.TicketId, locale),
		ValidateRequired("user_id", req.UserId, locale),
		ValidateRequired("title", req.Title, locale),
		ValidateRequired("content", req.Content, locale),
		ValidateMaxLen("title", req.Title, 250, locale),
		ValidateMaxLen("content", req.Content, 500, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	ticket, err := h.ticketService.UpdateTicket(ctx, req.TicketId, req.UserId, req.Title, req.Content, req.Attachment)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) AddResponse(ctx context.Context, req *pb.AddResponseRequest) (*pb.TicketResponse, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("ticket_id", req.TicketId, locale),
		ValidateRequired("user_id", req.UserId, locale),
		ValidateRequired("response", req.Response, locale),
		ValidateMaxLen("response", req.Response, 500, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	userName := strings.TrimSpace(req.UserName)
	if userName == "" {
		userName = "User"
	}

	ticket, err := h.ticketService.AddResponse(ctx, req.TicketId, req.UserId, req.Response, req.Attachment, userName)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return convertTicketToProto(ticket), nil
}

func (h *TicketHandler) CloseTicket(ctx context.Context, req *pb.CloseTicketRequest) (*pb.TicketResponse, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("ticket_id", req.TicketId, locale),
		ValidateRequired("user_id", req.UserId, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	ticket, err := h.ticketService.CloseTicket(ctx, req.TicketId, req.UserId)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return convertTicketToProto(ticket), nil
}

func convertTicketToProto(ticket *models.TicketWithRelations) *pb.TicketResponse {
	response := &pb.TicketResponse{
		Id:         ticket.ID,
		Title:      ticket.Title,
		Content:    ticket.Content,
		Attachment: ticket.Attachment,
		Code:       ticket.Code,
		Status:     ticket.Status,
		Importance: ticket.Importance,
		CreatedAt:  utils.FormatJalaliDateTime(ticket.CreatedAt),
		UpdatedAt:  utils.FormatJalaliDateTime(ticket.UpdatedAt),
	}

	if ticket.Department != nil {
		response.Department = *ticket.Department
	}

	response.Sender = &pbCommon.UserBasic{
		Id:   ticket.UserID,
		Code: ticket.SenderCode,
		Name: ticket.SenderName,
	}
	if ticket.SenderProfilePhoto != nil {
		response.Sender.ProfilePhoto = *ticket.SenderProfilePhoto
	}

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
