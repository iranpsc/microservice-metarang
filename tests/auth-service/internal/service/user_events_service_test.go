package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
)

func TestUserEventsService(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository(map[uint64]*models.User{
		1: {ID: 1, Code: "hm-1"},
		2: {ID: 2, Code: "hm-2"},
	})

	t.Run("list pagination", func(t *testing.T) {
		act := newFakeActivityRepository()
		act.getUserEventsByUserIDFunc = func(_ context.Context, _ uint64, page int32) ([]*models.UserEvent, error) {
			if page != 1 {
				t.Fatalf("page=%d", page)
			}
			out := make([]*models.UserEvent, 11)
			for i := range out {
				out[i] = &models.UserEvent{ID: uint64(i + 1), UserID: 1}
			}
			return out, nil
		}
		svc := service.NewUserEventsService(act, users)
		events, next, prev, err := svc.ListUserEvents(ctx, 1, 0)
		if err != nil || len(events) != 10 || next == "" || prev != "" {
			t.Fatalf("events=%d next=%q prev=%q err=%v", len(events), next, prev, err)
		}

		act.getUserEventsByUserIDFunc = func(context.Context, uint64, int32) ([]*models.UserEvent, error) {
			return []*models.UserEvent{{ID: 1}}, nil
		}
		events, next, prev, err = svc.ListUserEvents(ctx, 1, 2)
		if err != nil || len(events) != 1 || next != "" || prev == "" {
			t.Fatalf("events=%d next=%q prev=%q err=%v", len(events), next, prev, err)
		}

		act.getUserEventsByUserIDFunc = func(context.Context, uint64, int32) ([]*models.UserEvent, error) {
			return nil, errors.New("db")
		}
		_, _, _, err = svc.ListUserEvents(ctx, 1, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("get event", func(t *testing.T) {
		act := newFakeActivityRepository()
		svc := service.NewUserEventsService(act, users)
		_, _, _, err := svc.GetUserEvent(ctx, 1, 9)
		if !errors.Is(err, service.ErrUserEventNotFound) {
			t.Fatalf("err=%v", err)
		}

		act.events = []*models.UserEvent{{ID: 9, UserID: 1}}
		act.reports[9] = &models.UserEventReport{ID: 3, UserEventID: 9}
		act.responses[3] = []*models.UserEventReportResponse{{ID: 1, Response: "ok"}}
		ev, rep, resps, err := svc.GetUserEvent(ctx, 1, 9)
		if err != nil || ev == nil || rep == nil || len(resps) != 1 {
			t.Fatalf("ev=%v rep=%v resps=%v err=%v", ev, rep, resps, err)
		}
	})

	t.Run("report event", func(t *testing.T) {
		act := newFakeActivityRepository()
		act.events = []*models.UserEvent{{ID: 9, UserID: 1}}
		svc := service.NewUserEventsService(act, users)

		_, err := svc.ReportUserEvent(ctx, 1, 9, nil, "")
		if err == nil {
			t.Fatal("expected required description")
		}
		_, err = svc.ReportUserEvent(ctx, 1, 9, nil, strings.Repeat("a", 501))
		if !errors.Is(err, service.ErrEventDescriptionTooLong) {
			t.Fatalf("err=%v", err)
		}

		citizen := "hm-2"
		bad := "missing"
		_, err = svc.ReportUserEvent(ctx, 1, 9, &bad, "desc")
		if !errors.Is(err, service.ErrInvalidCitizenCode) {
			t.Fatalf("err=%v", err)
		}

		rep, err := svc.ReportUserEvent(ctx, 1, 9, &citizen, "desc")
		if err != nil || rep == nil {
			t.Fatalf("rep=%v err=%v", rep, err)
		}

		_, err = svc.ReportUserEvent(ctx, 1, 9, nil, "desc")
		if !errors.Is(err, service.ErrUserEventReportExists) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("response and close", func(t *testing.T) {
		act := newFakeActivityRepository()
		act.events = []*models.UserEvent{{ID: 9, UserID: 1}}
		act.reports[9] = &models.UserEventReport{ID: 3, UserEventID: 9}
		svc := service.NewUserEventsService(act, users)

		_, err := svc.SendReportResponse(ctx, 1, 9, "admin", "")
		if err == nil {
			t.Fatal("expected empty response error")
		}
		_, err = svc.SendReportResponse(ctx, 1, 9, "admin", strings.Repeat("a", 301))
		if !errors.Is(err, service.ErrResponseTooLong) {
			t.Fatalf("err=%v", err)
		}
		resp, err := svc.SendReportResponse(ctx, 1, 9, "admin", "ok")
		if err != nil || resp == nil {
			t.Fatalf("resp=%v err=%v", resp, err)
		}

		if err := svc.CloseEventReport(ctx, 1, 9); err != nil {
			t.Fatal(err)
		}

		delete(act.reports, 9)
		err = svc.CloseEventReport(ctx, 1, 9)
		if !errors.Is(err, service.ErrUserEventReportNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
