package handler_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbCommon "metarang/shared/pb/common"
	pb "metarang/shared/pb/support"

	"metarang/support-service/internal/handler"
	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/tests/internal/testutil"
)

func userEventClient(t *testing.T, repo *testutil.MockUserEventRepo) (pb.UserEventReportServiceClient, func()) {
	t.Helper()
	svc := service.NewUserEventService(repo)
	conn, cleanup := testutil.DialBufConn(func(s *grpc.Server) {
		handler.RegisterUserEventHandler(s, svc)
	})
	return pb.NewUserEventReportServiceClient(conn), cleanup
}

func TestUserEventHandler_CreateAndList_InternalAndDefaultPagination(t *testing.T) {
	repo := &testutil.MockUserEventRepo{
		CreateFunc: func(ctx context.Context, event *models.UserEvent) (*models.UserEvent, error) {
			return nil, errString("db")
		},
	}
	client, cleanup := userEventClient(t, repo)
	defer cleanup()
	_, err := client.CreateUserEvent(context.Background(), &pb.CreateUserEventRequest{UserId: 1, Title: "t"})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}

	var page, per int32
	repo = &testutil.MockUserEventRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64, p, pp int32) ([]*models.UserEvent, int, error) {
			page, per = p, pp
			return []*models.UserEvent{{ID: 2, UserID: userID, Event: "login"}}, 1, nil
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	resp, err := client.GetUserEvents(context.Background(), &pb.GetUserEventsRequest{UserId: 4})
	if err != nil || len(resp.Events) != 1 || page != 1 || per != 10 {
		t.Fatalf("err=%v page=%d per=%d", err, page, per)
	}

	repo = &testutil.MockUserEventRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64, p, pp int32) ([]*models.UserEvent, int, error) {
			return nil, 0, errString("list")
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	_, err = client.GetUserEvents(context.Background(), &pb.GetUserEventsRequest{
		UserId: 4, Pagination: &pbCommon.PaginationRequest{Page: 2, PerPage: 3},
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestUserEventHandler_ReportUserEvent_ValidationAndInternal(t *testing.T) {
	client, cleanup := userEventClient(t, &testutil.MockUserEventRepo{})
	defer cleanup()
	_, err := client.ReportUserEvent(context.Background(), &pb.ReportUserEventRequest{EventId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}

	repo := &testutil.MockUserEventRepo{
		CreateReportFunc: func(ctx context.Context, report *models.UserEventReport) (*models.UserEventReport, error) {
			return nil, errString("db")
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	_, err = client.ReportUserEvent(context.Background(), &pb.ReportUserEventRequest{
		EventId: 3, EventDescription: "bad",
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestUserEventHandler_SendEventReportResponse_ReportNotFoundDefaultNameAndValidation(t *testing.T) {
	client, cleanup := userEventClient(t, &testutil.MockUserEventRepo{})
	defer cleanup()
	_, err := client.SendEventReportResponse(context.Background(), &pb.SendEventReportResponseRequest{EventId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}

	repo := &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, nil
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	_, err = client.SendEventReportResponse(context.Background(), &pb.SendEventReportResponseRequest{
		EventId: 8, Response: "ok",
	})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}

	var name string
	repo = &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return &models.UserEventReport{ID: 20, UserEventID: eventID}, nil
		},
		CreateReportResponseFunc: func(ctx context.Context, response *models.UserEventReportResponse) (*models.UserEventReportResponse, error) {
			name = response.ResponserName
			response.ID = 1
			return response, nil
		},
		UpdateReportStatusFunc: func(ctx context.Context, reportID uint64, status int32) error {
			return nil
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	resp, err := client.SendEventReportResponse(context.Background(), &pb.SendEventReportResponseRequest{
		EventId: 8, Response: "ok", ResponderName: "  ",
	})
	if err != nil || name != "Admin" || resp.ResponserName != "Admin" {
		t.Fatalf("err=%v name=%q resp=%+v", err, name, resp)
	}
}

func TestUserEventHandler_CloseUserEventReport_UnauthorizedEventNotFound(t *testing.T) {
	repo := &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return nil, nil
		},
	}
	client, cleanup := userEventClient(t, repo)
	defer cleanup()
	_, err := client.CloseUserEventReport(context.Background(), &pb.CloseUserEventReportRequest{EventId: 1, UserId: 7})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("event not found got %v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 7}}, nil
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	_, err = client.CloseUserEventReport(context.Background(), &pb.CloseUserEventReportRequest{EventId: 1, UserId: 99})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 7}}, nil
		},
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, nil
		},
	}
	client, cleanup = userEventClient(t, repo)
	defer cleanup()
	_, err = client.CloseUserEventReport(context.Background(), &pb.CloseUserEventReportRequest{EventId: 1, UserId: 7})
	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("report not found got %v", err)
	}
}
