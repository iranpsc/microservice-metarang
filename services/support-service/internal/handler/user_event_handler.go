package handler

import (
	"context"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/service"
	"metargb/support-service/internal/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pbCommon "metargb/shared/pb/common"
	pb "metargb/shared/pb/support"
)

type UserEventHandler struct {
	pb.UnimplementedUserEventReportServiceServer
	userEventService service.UserEventService
}

func NewUserEventHandler(userEventService service.UserEventService) *UserEventHandler {
	return &UserEventHandler{
		userEventService: userEventService,
	}
}

func RegisterUserEventHandler(grpcServer *grpc.Server, userEventService service.UserEventService) {
	handler := NewUserEventHandler(userEventService)
	pb.RegisterUserEventReportServiceServer(grpcServer, handler)
}

func (h *UserEventHandler) CreateUserEvent(ctx context.Context, req *pb.CreateUserEventRequest) (*pb.UserEventResponse, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := mergeValidationErrors(
		validateRequired("user_id", req.UserId, locale),
		validateRequired("title", req.Title, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	event, err := h.userEventService.CreateUserEvent(ctx, req.UserId, req.Title, req.Description, req.EventDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user event: %v", err)
	}

	return convertUserEventToProto(event), nil
}

func (h *UserEventHandler) GetUserEvents(ctx context.Context, req *pb.GetUserEventsRequest) (*pb.UserEventsResponse, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := validateRequired("user_id", req.UserId, locale)
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

	events, total, err := h.userEventService.GetUserEvents(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user events: %v", err)
	}

	response := &pb.UserEventsResponse{
		Events: make([]*pb.UserEventResponse, len(events)),
		Pagination: &pbCommon.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       int32(total),
			LastPage:    int32((total + int(perPage) - 1) / int(perPage)),
		},
	}

	for i, event := range events {
		response.Events[i] = convertUserEventToProto(event)
	}

	return response, nil
}

func (h *UserEventHandler) GetUserEvent(ctx context.Context, req *pb.GetUserEventRequest) (*pb.UserEventResponse, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := validateRequired("event_id", req.EventId, locale)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	event, err := h.userEventService.GetUserEvent(ctx, req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user event: %v", err)
	}

	if event == nil {
		return nil, status.Error(codes.NotFound, "user event not found")
	}

	return convertUserEventWithReportToProto(event), nil
}

func (h *UserEventHandler) ReportUserEvent(ctx context.Context, req *pb.ReportUserEventRequest) (*pb.UserEventReportResponse, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := mergeValidationErrors(
		validateRequired("event_id", req.EventId, locale),
		validateRequired("event_description", req.EventDescription, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	report, err := h.userEventService.ReportUserEvent(ctx, req.EventId, req.SuspiciousCitizen, req.EventDescription)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to report user event: %v", err)
	}

	return convertUserEventReportToProto(report), nil
}

func (h *UserEventHandler) SendEventReportResponse(ctx context.Context, req *pb.SendEventReportResponseRequest) (*pbCommon.Empty, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := mergeValidationErrors(
		validateRequired("report_id", req.ReportId, locale),
		validateRequired("response", req.Response, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	// Get responder name (should query user service in production)
	responderName := "Admin"

	err := h.userEventService.SendEventReportResponse(ctx, req.ReportId, responderName, req.Response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

	return &pbCommon.Empty{}, nil
}

// Helper functions to convert models to proto
func convertUserEventToProto(event *models.UserEvent) *pb.UserEventResponse {
	return &pb.UserEventResponse{
		Id:          event.ID,
		UserId:      event.UserID,
		Title:       event.Event, // Using Event field as Title
		Description: "",          // Not stored in Laravel UserEvent
		EventDate:   utils.FormatJalaliDate(event.CreatedAt),
		CreatedAt:   utils.FormatJalaliDateTime(event.CreatedAt),
	}
}

func convertUserEventWithReportToProto(event *models.UserEventWithReport) *pb.UserEventResponse {
	response := &pb.UserEventResponse{
		Id:          event.ID,
		UserId:      event.UserID,
		Title:       event.Event,
		Description: "",
		EventDate:   utils.FormatJalaliDate(event.CreatedAt),
		CreatedAt:   utils.FormatJalaliDateTime(event.CreatedAt),
	}

	return response
}

func convertUserEventReportToProto(report *models.UserEventReport) *pb.UserEventReportResponse {
	response := &pb.UserEventReportResponse{
		Id:               report.ID,
		EventId:          report.UserEventID,
		EventDescription: report.EventDescription,
		CreatedAt:        utils.FormatJalaliDateTime(report.CreatedAt),
	}

	if report.SuspeciousCitizen != nil {
		response.SuspiciousCitizen = *report.SuspeciousCitizen
	}

	return response
}
