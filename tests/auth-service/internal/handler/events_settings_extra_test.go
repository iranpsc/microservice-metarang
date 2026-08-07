package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

func TestUserEventsHandler_GetWithReport(t *testing.T) {
	ctx := authenticatedContext(1)
	m := &mockUserEventsService{}
	m.getUserEventFunc = func(context.Context, uint64, uint64) (*models.UserEvent, *models.UserEventReport, []*models.UserEventReportResponse, error) {
		return &models.UserEvent{ID: 1, UserID: 1, Event: "e", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			&models.UserEventReport{
				ID: 2, UserEventID: 1, EventDescription: "d", Status: 1,
				SuspeciousCitizen: sql.NullString{String: "hm-x", Valid: true},
				CreatedAt:         time.Now(), UpdatedAt: time.Now(),
			},
			[]*models.UserEventReportResponse{{
				ID: 3, UserEventReportID: 2, Response: "r", ResponserName: "n",
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}}, nil
	}
	h := handler.NewUserEventsHandler(m, &mockUserRepo{
		findByIDFunc: func(context.Context, uint64) (*models.User, error) { return &models.User{ID: 1, Name: "n"}, nil },
	})
	resp, err := h.GetUserEvent(ctx, &pb.GetUserEventRequest{EventId: 1})
	if err != nil || resp.Data == nil || resp.Data.Report == nil || len(resp.Data.Report.Responses) != 1 {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
	if resp.Data.Report.SuspeciousCitizen != "hm-x" {
		t.Fatalf("citizen=%q", resp.Data.Report.SuspeciousCitizen)
	}
}

func TestSettingsHandler_UpdateGeneralErrorPaths(t *testing.T) {
	ctx := authenticatedContext(1)
	m := &settingsSvcMock{}
	m.updateGeneralSettingsFunc = func(context.Context, uint64, uint64, map[string]bool) (map[string]bool, error) {
		return nil, service.ErrSettingsNotFound
	}
	h := handler.RegisterSettingsHandler(grpc.NewServer(), m)
	_, err := h.UpdateGeneralSettings(ctx, &pb.UpdateGeneralSettingsRequest{
		SettingId: 1,
		Notifications: &pb.NotificationSettingsData{AnnouncementsSms: true},
	})
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Fatalf("code=%v", st.Code())
	}

	m.updateGeneralSettingsFunc = func(context.Context, uint64, uint64, map[string]bool) (map[string]bool, error) {
		return nil, errors.New("settings do not belong to user")
	}
	_, err = h.UpdateGeneralSettings(ctx, &pb.UpdateGeneralSettingsRequest{
		SettingId: 1, Notifications: &pb.NotificationSettingsData{},
	})
	st, _ = status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("code=%v", st.Code())
	}

	m.updateGeneralSettingsFunc = func(context.Context, uint64, uint64, map[string]bool) (map[string]bool, error) {
		return nil, errors.New("missing required notification channel")
	}
	_, err = h.UpdateGeneralSettings(ctx, &pb.UpdateGeneralSettingsRequest{
		SettingId: 1, Notifications: &pb.NotificationSettingsData{},
	})
	st, _ = status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code=%v", st.Code())
	}

	m.updateGeneralSettingsFunc = func(context.Context, uint64, uint64, map[string]bool) (map[string]bool, error) {
		return map[string]bool{
			"announcements_sms": true, "announcements_email": true,
			"reports_sms": true, "reports_email": true,
			"login_verification_sms": true, "login_verification_email": true,
			"transactions_sms": true, "transactions_email": true,
			"trades_sms": true, "trades_email": true,
		}, nil
	}
	resp, err := h.UpdateGeneralSettings(ctx, &pb.UpdateGeneralSettingsRequest{
		SettingId: 1,
		Notifications: &pb.NotificationSettingsData{
			AnnouncementsSms: true, AnnouncementsEmail: true,
			ReportsSms: true, ReportsEmail: true,
			LoginVerificationSms: true, LoginVerificationEmail: true,
			TransactionsSms: true, TransactionsEmail: true,
			TradesSms: true, TradesEmail: true,
		},
	})
	if err != nil || resp.Data == nil || !resp.Data.AnnouncementsSms {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
}
