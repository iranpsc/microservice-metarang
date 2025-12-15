package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
	"metargb/shared/pkg/helpers"
)

type userEventsHandler struct {
	pb.UnimplementedUserEventsServiceServer
	service  service.UserEventsService
	userRepo interface {
		FindByID(ctx context.Context, id uint64) (*models.User, error)
	}
}

func RegisterUserEventsHandler(grpcServer *grpc.Server, userEventsService service.UserEventsService, userRepo interface {
	FindByID(ctx context.Context, id uint64) (*models.User, error)
}) {
	pb.RegisterUserEventsServiceServer(grpcServer, &userEventsHandler{
		service:  userEventsService,
		userRepo: userRepo,
	})
}

// NewUserEventsHandler creates a new user events handler
func NewUserEventsHandler(service service.UserEventsService, userRepo interface {
	FindByID(ctx context.Context, id uint64) (*models.User, error)
}) *userEventsHandler {
	return &userEventsHandler{
		service:  service,
		userRepo: userRepo,
	}
}

// ListUserEvents handles GET /api/events
func (h *userEventsHandler) ListUserEvents(ctx context.Context, req *pb.ListUserEventsRequest) (*pb.ListUserEventsResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}

	events, nextPageURL, prevPageURL, err := h.service.ListUserEvents(ctx, req.UserId, page)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user events: %v", err)
	}

	// Convert to proto format
	eventResources := make([]*pb.UserEventResource, 0, len(events))
	for _, event := range events {
		eventResource := h.convertUserEventToResource(event, false) // false = don't include report
		eventResources = append(eventResources, eventResource)
	}

	// Build pagination meta
	pagination := &pb.PaginationMeta{
		CurrentPage: page,
		NextPageUrl: nextPageURL,
		PrevPageUrl: prevPageURL,
	}

	return &pb.ListUserEventsResponse{
		Data:       eventResources,
		Pagination: pagination,
	}, nil
}

// GetUserEvent handles GET /api/events/{userEvent}
func (h *userEventsHandler) GetUserEvent(ctx context.Context, req *pb.GetUserEventRequest) (*pb.GetUserEventResponse, error) {
	event, report, responses, err := h.service.GetUserEvent(ctx, req.UserId, req.EventId)
	if err != nil {
		if err == service.ErrUserEventNotFound {
			return nil, status.Errorf(codes.NotFound, "user event not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user event: %v", err)
	}

	// Convert to proto format with report
	eventResource := h.convertUserEventToResource(event, true) // true = include report
	if report != nil {
		eventResource.Report = h.convertUserEventReportToResource(report, responses)
	}

	return &pb.GetUserEventResponse{
		Data: eventResource,
	}, nil
}

// ReportUserEvent handles POST /api/events/report/{userEvent}
func (h *userEventsHandler) ReportUserEvent(ctx context.Context, req *pb.ReportUserEventRequest) (*pb.UserEventReportResponse, error) {
	// Validate event_description
	if req.EventDescription == "" {
		return nil, status.Errorf(codes.InvalidArgument, "event_description is required")
	}
	if len(req.EventDescription) > 500 {
		return nil, status.Errorf(codes.InvalidArgument, "event_description exceeds maximum length of 500 characters")
	}

	var suspeciousCitizen *string
	if req.SuspeciousCitizen != "" {
		suspeciousCitizen = &req.SuspeciousCitizen
	}

	report, err := h.service.ReportUserEvent(ctx, req.UserId, req.EventId, suspeciousCitizen, req.EventDescription)
	if err != nil {
		switch err {
		case service.ErrUserEventNotFound:
			return nil, status.Errorf(codes.NotFound, "user event not found")
		case service.ErrUserEventReportExists:
			return nil, status.Errorf(codes.AlreadyExists, "user event report already exists")
		case service.ErrInvalidCitizenCode:
			return nil, status.Errorf(codes.InvalidArgument, "invalid citizen code")
		case service.ErrEventDescriptionTooLong:
			return nil, status.Errorf(codes.InvalidArgument, "event_description exceeds maximum length of 500 characters")
		default:
			return nil, status.Errorf(codes.Internal, "failed to create report: %v", err)
		}
	}

	// Get empty responses for new report
	reportResource := h.convertUserEventReportToResource(report, nil)

	return &pb.UserEventReportResponse{
		Data: reportResource,
	}, nil
}

// SendReportResponse handles POST /api/events/report/response/{userEvent}
func (h *userEventsHandler) SendReportResponse(ctx context.Context, req *pb.SendReportResponseRequest) (*pb.UserEventReportResponseResponse, error) {
	// Validate response
	if req.Response == "" {
		return nil, status.Errorf(codes.InvalidArgument, "response is required")
	}
	if len(req.Response) > 300 {
		return nil, status.Errorf(codes.InvalidArgument, "response exceeds maximum length of 300 characters")
	}

	// Get user name for responser_name
	user, err := h.userRepo.FindByID(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	reportResponse, err := h.service.SendReportResponse(ctx, req.UserId, req.EventId, user.Name, req.Response)
	if err != nil {
		switch err {
		case service.ErrUserEventNotFound:
			return nil, status.Errorf(codes.NotFound, "user event not found")
		case service.ErrUserEventReportNotFound:
			return nil, status.Errorf(codes.NotFound, "user event report not found")
		case service.ErrResponseTooLong:
			return nil, status.Errorf(codes.InvalidArgument, "response exceeds maximum length of 300 characters")
		default:
			return nil, status.Errorf(codes.Internal, "failed to send report response: %v", err)
		}
	}

	responseResource := h.convertUserEventReportResponseToResource(reportResponse)

	return &pb.UserEventReportResponseResponse{
		Data: responseResource,
	}, nil
}

// CloseEventReport handles POST /api/events/report/close/{userEvent}
func (h *userEventsHandler) CloseEventReport(ctx context.Context, req *pb.CloseEventReportRequest) (*emptypb.Empty, error) {
	err := h.service.CloseEventReport(ctx, req.UserId, req.EventId)
	if err != nil {
		switch err {
		case service.ErrUserEventNotFound:
			return nil, status.Errorf(codes.NotFound, "user event not found")
		case service.ErrUserEventReportNotFound:
			return nil, status.Errorf(codes.NotFound, "user event report not found")
		default:
			return nil, status.Errorf(codes.Internal, "failed to close event report: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}

// Helper functions to convert models to proto resources

func (h *userEventsHandler) convertUserEventToResource(event *models.UserEvent, includeReport bool) *pb.UserEventResource {
	// Convert status: 1 = "موفق", 0 = "ناموفق" (Persian: successful or unsuccessful)
	statusStr := "ناموفق"
	if event.Status == 1 {
		statusStr = "موفق"
	}

	resource := &pb.UserEventResource{
		Id:     event.ID,
		Event:  event.Event,
		Ip:     event.IP,
		Device: event.Device,
		Status: statusStr,
		Date:   helpers.FormatJalaliDate(event.CreatedAt),
		Time:   helpers.FormatJalaliTime(event.CreatedAt),
	}

	// Report is only included when includeReport is true (for GetUserEvent)
	// It will be set by the caller if needed

	return resource
}

func (h *userEventsHandler) convertUserEventReportToResource(report *models.UserEventReport, responses []*models.UserEventReportResponse) *pb.UserEventReportResource {
	resource := &pb.UserEventReportResource{
		Id:               report.ID,
		EventDescription: report.EventDescription,
		Status:           report.Status,
		Closed:           report.Closed,
		Date:             helpers.FormatJalaliDate(report.CreatedAt),
		Time:             helpers.FormatJalaliTime(report.CreatedAt),
	}

	// Handle suspecious_citizen (nullable)
	if report.SuspeciousCitizen.Valid {
		resource.SuspeciousCitizen = report.SuspeciousCitizen.String
	}

	// Convert responses
	if responses != nil {
		responseResources := make([]*pb.UserEventReportResponseResource, 0, len(responses))
		for _, response := range responses {
			responseResources = append(responseResources, h.convertUserEventReportResponseToResource(response))
		}
		resource.Responses = responseResources
	}

	return resource
}

func (h *userEventsHandler) convertUserEventReportResponseToResource(response *models.UserEventReportResponse) *pb.UserEventReportResponseResource {
	return &pb.UserEventReportResponseResource{
		Id:            response.ID,
		ResponserName: response.ResponserName,
		Response:      response.Response,
		Date:          helpers.FormatJalaliDate(response.CreatedAt),
		Time:          helpers.FormatJalaliTime(response.CreatedAt),
	}
}
