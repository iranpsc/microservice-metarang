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
		ValidateRequired("user_id", req.UserId, locale),
		ValidateRequired("title", req.Title, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	event, err := h.userEventService.CreateUserEvent(ctx, req.UserId, req.Title, req.Description, req.EventDate)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return convertUserEventToProto(event), nil
}

func (h *UserEventHandler) GetUserEvents(ctx context.Context, req *pb.GetUserEventsRequest) (*pb.UserEventsResponse, error) {
	locale := "en" // TODO: Get locale from config or context
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

	events, total, err := h.userEventService.GetUserEvents(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, MapServiceError(err)
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
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("event_id", req.EventId, locale),
		ValidateRequired("user_id", req.UserId, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	event, err := h.userEventService.GetUserEvent(ctx, req.EventId, req.UserId)
	if err != nil {
		return nil, MapServiceError(err)
	}

	if event == nil {
		return nil, status.Error(codes.NotFound, "user event not found")
	}

	return convertUserEventWithReportToProto(event), nil
}

func (h *UserEventHandler) ReportUserEvent(ctx context.Context, req *pb.ReportUserEventRequest) (*pb.UserEventReportResponse, error) {
	locale := "en" // TODO: Get locale from config or context
	validationErrors := mergeValidationErrors(
		ValidateRequired("event_id", req.EventId, locale),
		ValidateRequired("event_description", req.EventDescription, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	report, err := h.userEventService.ReportUserEvent(ctx, req.EventId, req.SuspiciousCitizen, req.EventDescription)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return convertUserEventReportToProto(report), nil
}

func (h *UserEventHandler) SendEventReportResponse(ctx context.Context, req *pb.SendEventReportResponseRequest) (*pb.SendEventReportResponseReply, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("event_id", req.EventId, locale),
		ValidateRequired("response", req.Response, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	responderName := strings.TrimSpace(req.ResponderName)
	if responderName == "" {
		responderName = "Admin"
	}

	created, err := h.userEventService.SendEventReportResponse(ctx, req.EventId, responderName, req.Response)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return &pb.SendEventReportResponseReply{
		Id:            created.ID,
		ResponserName: created.ResponserName,
		Response:      created.Response,
		Date:          utils.FormatJalaliDate(created.CreatedAt),
		Time:          utils.FormatJalaliTime(created.CreatedAt),
	}, nil
}

func (h *UserEventHandler) CloseUserEventReport(ctx context.Context, req *pb.CloseUserEventReportRequest) (*pbCommon.Empty, error) {
	locale := handlerLocale(ctx)
	validationErrors := mergeValidationErrors(
		ValidateRequired("event_id", req.EventId, locale),
		ValidateRequired("user_id", req.UserId, locale),
	)
	if len(validationErrors) > 0 {
		return nil, returnValidationError(validationErrors)
	}

	err := h.userEventService.CloseUserEventReport(ctx, req.EventId, req.UserId)
	if err != nil {
		return nil, MapServiceError(err)
	}

	return &pbCommon.Empty{}, nil
}

// Helper functions to convert models to proto
func convertUserEventToProto(event *models.UserEvent) *pb.UserEventResponse {
	if event == nil {
		return nil
	}
	return &pb.UserEventResponse{
		Id:          event.ID,
		UserId:      event.UserID,
		Title:       event.Event,
		Description: "",
		EventDate:   utils.FormatJalaliDate(event.CreatedAt),
		CreatedAt:   utils.FormatJalaliDateTime(event.CreatedAt),
		Ip:          event.IP,
		Device:      event.Device,
		StatusOk:    event.Status,
	}
}

func convertUserEventWithReportToProto(event *models.UserEventWithReport) *pb.UserEventResponse {
	if event == nil {
		return nil
	}
	resp := convertUserEventToProto(&event.UserEvent)
	if event.Report != nil {
		r := event.Report
		detail := &pb.UserEventReportDetail{
			Id:               r.ID,
			EventDescription: r.EventDescription,
			Status:           r.Status,
			Closed:           r.Closed,
			Date:             utils.FormatJalaliDate(r.CreatedAt),
			Time:             utils.FormatJalaliTime(r.CreatedAt),
		}
		if r.SuspeciousCitizen != nil {
			detail.SuspiciousCitizen = *r.SuspeciousCitizen
		}
		for _, rr := range event.Responses {
			detail.Responses = append(detail.Responses, &pb.UserEventReportResponseItem{
				Id:            rr.ID,
				ResponserName: rr.ResponserName,
				Response:      rr.Response,
				Date:          utils.FormatJalaliDate(rr.CreatedAt),
				Time:          utils.FormatJalaliTime(rr.CreatedAt),
			})
		}
		resp.Report = detail
	}
	return resp
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
