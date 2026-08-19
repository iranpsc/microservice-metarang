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

func TestUserEventService_CreateUserEvent_DiscardsDescriptionAndUsesDefaults(t *testing.T) {
	var stored *models.UserEvent
	repo := &testutil.MockUserEventRepo{
		CreateFunc: func(ctx context.Context, event *models.UserEvent) (*models.UserEvent, error) {
			cp := *event
			stored = &cp
			event.ID = 9
			return event, nil
		},
	}
	svc := service.NewUserEventService(repo)
	got, err := svc.CreateUserEvent(context.Background(), 4, "login", "should-be-discarded", "2024-01-01")
	if err != nil || got.ID != 9 {
		t.Fatalf("err=%v got=%+v", err, got)
	}
	if stored == nil || stored.Event != "login" || stored.IP != "0.0.0.0" || stored.Device != "unknown" || !stored.Status {
		t.Fatalf("expected discarded description/date and default ip/device/status: %+v", stored)
	}
}

func TestUserEventService_GetUserEvents_PaginationDefaults(t *testing.T) {
	var gotPage, gotPer int32
	repo := &testutil.MockUserEventRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64, page, perPage int32) ([]*models.UserEvent, int, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, nil
		},
	}
	svc := service.NewUserEventService(repo)
	_, _, err := svc.GetUserEvents(context.Background(), 1, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != 1 || gotPer != 10 {
		t.Fatalf("page=%d per=%d", gotPage, gotPer)
	}
}

func TestUserEventService_GetUserEvent_ErrorNilUnauthorized(t *testing.T) {
	repo := &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewUserEventService(repo)
	_, err := svc.GetUserEvent(context.Background(), 1, 1)
	if err == nil || err.Error() != "db" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return nil, nil
		},
	}
	svc = service.NewUserEventService(repo)
	got, err := svc.GetUserEvent(context.Background(), 1, 1)
	if err != nil || got != nil {
		t.Fatalf("expected nil event got=%v err=%v", got, err)
	}
}

func TestUserEventService_ReportUserEvent_EmptySuspiciousCitizenIsNilPointer(t *testing.T) {
	var stored *models.UserEventReport
	repo := &testutil.MockUserEventRepo{
		CreateReportFunc: func(ctx context.Context, report *models.UserEventReport) (*models.UserEventReport, error) {
			cp := *report
			stored = &cp
			report.ID = 3
			return report, nil
		},
	}
	svc := service.NewUserEventService(repo)
	_, err := svc.ReportUserEvent(context.Background(), 10, "", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.SuspeciousCitizen != nil {
		t.Fatalf("empty suspiciousCitizen must stay nil: %+v", stored)
	}

	_, err = svc.ReportUserEvent(context.Background(), 10, "citizen", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if stored.SuspeciousCitizen == nil || *stored.SuspeciousCitizen != "citizen" {
		t.Fatalf("non-empty must set pointer: %+v", stored)
	}
}

func TestUserEventService_SendEventReportResponse_ErrorsAndSuccessStatus(t *testing.T) {
	repo := &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, errors.New("lookup")
		},
	}
	svc := service.NewUserEventService(repo)
	_, err := svc.SendEventReportResponse(context.Background(), 1, "Admin", "ok")
	if err == nil || !strings.Contains(err.Error(), "failed to get report") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, nil
		},
	}
	svc = service.NewUserEventService(repo)
	_, err = svc.SendEventReportResponse(context.Background(), 1, "Admin", "ok")
	if err == nil || err.Error() != "report not found" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return &models.UserEventReport{ID: 7, UserEventID: eventID}, nil
		},
		CreateReportResponseFunc: func(ctx context.Context, response *models.UserEventReportResponse) (*models.UserEventReportResponse, error) {
			return nil, errors.New("insert")
		},
	}
	svc = service.NewUserEventService(repo)
	_, err = svc.SendEventReportResponse(context.Background(), 1, "Admin", "ok")
	if err == nil || !strings.Contains(err.Error(), "failed to create response") {
		t.Fatalf("err=%v", err)
	}

	var status int32
	repo = &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return &models.UserEventReport{ID: 7, UserEventID: eventID}, nil
		},
		CreateReportResponseFunc: func(ctx context.Context, response *models.UserEventReportResponse) (*models.UserEventReportResponse, error) {
			response.ID = 1
			return response, nil
		},
		UpdateReportStatusFunc: func(ctx context.Context, reportID uint64, st int32) error {
			return errors.New("status")
		},
	}
	svc = service.NewUserEventService(repo)
	_, err = svc.SendEventReportResponse(context.Background(), 1, "Admin", "ok")
	if err == nil || !strings.Contains(err.Error(), "failed to update report status") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return &models.UserEventReport{ID: 7, UserEventID: eventID}, nil
		},
		CreateReportResponseFunc: func(ctx context.Context, response *models.UserEventReportResponse) (*models.UserEventReportResponse, error) {
			response.ID = 11
			return response, nil
		},
		UpdateReportStatusFunc: func(ctx context.Context, reportID uint64, st int32) error {
			status = st
			return nil
		},
	}
	svc = service.NewUserEventService(repo)
	got, err := svc.SendEventReportResponse(context.Background(), 1, "Admin", "ok")
	if err != nil || got.ID != 11 || status != 1 {
		t.Fatalf("err=%v got=%+v status=%d", err, got, status)
	}
}

func TestUserEventService_CloseUserEventReport_ErrorPaths(t *testing.T) {
	repo := &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewUserEventService(repo)
	err := svc.CloseUserEventReport(context.Background(), 1, 5)
	if err == nil || err.Error() != "db" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return nil, nil
		},
	}
	svc = service.NewUserEventService(repo)
	err = svc.CloseUserEventReport(context.Background(), 1, 5)
	if err == nil || err.Error() != "user event not found" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 5}}, nil
		},
	}
	svc = service.NewUserEventService(repo)
	err = svc.CloseUserEventReport(context.Background(), 1, 9)
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 5}}, nil
		},
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, errors.New("report db")
		},
	}
	svc = service.NewUserEventService(repo)
	err = svc.CloseUserEventReport(context.Background(), 1, 5)
	if err == nil || err.Error() != "report db" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 5}}, nil
		},
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return nil, nil
		},
	}
	svc = service.NewUserEventService(repo)
	err = svc.CloseUserEventReport(context.Background(), 1, 5)
	if err == nil || err.Error() != "report not found" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockUserEventRepo{
		GetByIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventWithReport, error) {
			return &models.UserEventWithReport{UserEvent: models.UserEvent{ID: 1, UserID: 5}}, nil
		},
		GetReportByEventIDFunc: func(ctx context.Context, eventID uint64) (*models.UserEventReport, error) {
			return &models.UserEventReport{ID: 9}, nil
		},
		CloseReportFunc: func(ctx context.Context, reportID uint64) error {
			return errors.New("close")
		},
	}
	svc = service.NewUserEventService(repo)
	err = svc.CloseUserEventReport(context.Background(), 1, 5)
	if err == nil || err.Error() != "close" {
		t.Fatalf("err=%v", err)
	}
}
