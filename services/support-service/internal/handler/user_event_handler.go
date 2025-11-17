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
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	event, err := h.userEventService.CreateUserEvent(ctx, req.UserId, req.Title, req.Description, req.EventDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user event: %v", err)
	}

	return convertUserEventToProto(event), nil
}

func (h *UserEventHandler) GetUserEvents(ctx context.Context, req *pb.GetUserEventsRequest) (*pb.UserEventsResponse, error) {
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
	if req.EventId == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
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
	if req.EventId == 0 {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}
	if req.EventDescription == "" {
		return nil, status.Error(codes.InvalidArgument, "event_description is required")
	}

	report, err := h.userEventService.ReportUserEvent(ctx, req.EventId, req.SuspiciousCitizen, req.EventDescription)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to report user event: %v", err)
	}

	return convertUserEventReportToProto(report), nil
}

func (h *UserEventHandler) SendEventReportResponse(ctx context.Context, req *pb.SendEventReportResponseRequest) (*pbCommon.Empty, error) {
	if req.ReportId == 0 {
		return nil, status.Error(codes.InvalidArgument, "report_id is required")
	}
	if req.Response == "" {
		return nil, status.Error(codes.InvalidArgument, "response is required")
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
