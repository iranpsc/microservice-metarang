package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/notifications-service/internal/errs"
	"metarang/notifications-service/internal/handler"
	"metarang/notifications-service/internal/models"
	"metarang/notifications-service/internal/service"
	"metarang/notifications-service/tests/internal/testutil"
	pbCommon "metarang/shared/pb/common"
	pb "metarang/shared/pb/notifications"
)

var errHandlerTest = errors.New("handler test error")

func newNotificationHandler(t *testing.T, svc service.NotificationService) *handler.NotificationHandler {
	t.Helper()
	s := testutil.NewBufGRPCTestServer()
	h := handler.RegisterNotificationHandler(s.Server, svc)
	s.Start(t)
	return h
}

func TestNotificationHandler_SendNotification(t *testing.T) {
	ctx := context.Background()
	h := newNotificationHandler(t, &testutil.MockNotificationService{
		SendNotificationFunc: func(_ context.Context, input service.SendNotificationInput) (*models.NotificationResult, error) {
			assert.Equal(t, uint64(42), input.UserID)
			return &models.NotificationResult{ID: 99, Sent: true}, nil
		},
	})

	resp, err := h.SendNotification(ctx, &pb.SendNotificationRequest{
		UserId: 42, Type: "system", Title: "Title", Message: "Message",
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(99), resp.Id)
	assert.True(t, resp.Sent)

	for _, tc := range []struct {
		name string
		req  *pb.SendNotificationRequest
	}{
		{"missing user_id", &pb.SendNotificationRequest{Type: "t", Title: "t", Message: "m"}},
		{"missing type", &pb.SendNotificationRequest{UserId: 1, Title: "t", Message: "m"}},
		{"missing title", &pb.SendNotificationRequest{UserId: 1, Type: "t", Message: "m"}},
		{"missing message", &pb.SendNotificationRequest{UserId: 1, Type: "t", Title: "t"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.SendNotification(ctx, tc.req)
			require.Error(t, err)
			st, _ := status.FromError(err)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}

	t.Run("service error", func(t *testing.T) {
		hErr := newNotificationHandler(t, &testutil.MockNotificationService{
			SendNotificationFunc: func(context.Context, service.SendNotificationInput) (*models.NotificationResult, error) {
				return nil, errHandlerTest
			},
		})
		_, err := hErr.SendNotification(ctx, &pb.SendNotificationRequest{UserId: 1, Type: "t", Title: "t", Message: "m"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestNotificationHandler_GetNotifications(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	h := newNotificationHandler(t, &testutil.MockNotificationService{
		GetNotificationsFunc: func(_ context.Context, userID uint64, filter models.NotificationFilter) ([]models.Notification, int64, error) {
			assert.Equal(t, uint64(7), userID)
			assert.Equal(t, int32(2), filter.Page)
			return []models.Notification{{ID: "n1", Type: "system", Title: "T", Message: "M", CreatedAt: now}}, 1, nil
		},
	})

	resp, err := h.GetNotifications(ctx, &pb.GetNotificationsRequest{
		UserId:     7,
		Pagination: &pbCommon.PaginationRequest{Page: 2, PerPage: 5},
	})
	require.NoError(t, err)
	require.Len(t, resp.Notifications, 1)
	assert.Equal(t, "n1", resp.Notifications[0].Id)
	assert.Equal(t, int32(1), resp.Pagination.Total)

	_, err = h.GetNotifications(ctx, &pb.GetNotificationsRequest{})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestNotificationHandler_GetNotification(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := newNotificationHandler(t, &testutil.MockNotificationService{
			GetNotificationByIDFunc: func(_ context.Context, id string, userID uint64) (*models.Notification, error) {
				return &models.Notification{ID: id, UserID: userID, Type: "system", Title: "T", Message: "M", CreatedAt: now}, nil
			},
		})
		resp, err := h.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: "abc", UserId: 3})
		require.NoError(t, err)
		assert.Equal(t, "abc", resp.Id)
	})

	t.Run("not found via service error", func(t *testing.T) {
		h := newNotificationHandler(t, &testutil.MockNotificationService{
			GetNotificationByIDFunc: func(context.Context, string, uint64) (*models.Notification, error) {
				return nil, errs.ErrNotificationNotFound
			},
		})
		_, err := h.GetNotification(ctx, &pb.GetNotificationRequest{NotificationId: "x", UserId: 3})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("validation", func(t *testing.T) {
		h := newNotificationHandler(t, &testutil.MockNotificationService{})
		_, err := h.GetNotification(ctx, &pb.GetNotificationRequest{UserId: 1})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	ctx := context.Background()
	called := false
	h := newNotificationHandler(t, &testutil.MockNotificationService{
		MarkAsReadFunc: func(_ context.Context, id string, userID uint64) error {
			called = true
			assert.Equal(t, "n1", id)
			assert.Equal(t, uint64(5), userID)
			return nil
		},
	})

	_, err := h.MarkAsRead(ctx, &pb.MarkAsReadRequest{NotificationId: "n1", UserId: 5})
	require.NoError(t, err)
	assert.True(t, called)

	_, err = h.MarkAsRead(ctx, &pb.MarkAsReadRequest{UserId: 5})
	require.Error(t, err)
}

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	ctx := context.Background()
	h := newNotificationHandler(t, &testutil.MockNotificationService{
		MarkAllAsReadFunc: func(_ context.Context, userID uint64) error {
			assert.Equal(t, uint64(9), userID)
			return nil
		},
	})

	_, err := h.MarkAllAsRead(ctx, &pb.MarkAllAsReadRequest{UserId: 9})
	require.NoError(t, err)

	_, err = h.MarkAllAsRead(ctx, &pb.MarkAllAsReadRequest{})
	require.Error(t, err)
}

func TestNotificationHandler_ConvertNotificationReadAt(t *testing.T) {
	readAt := time.Now().UTC()
	h := newNotificationHandler(t, &testutil.MockNotificationService{
		GetNotificationByIDFunc: func(context.Context, string, uint64) (*models.Notification, error) {
			return &models.Notification{
				ID: "n1", Type: "alert", Title: "T", Message: "M",
				CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				ReadAt:    &readAt,
			}, nil
		},
	})
	resp, err := h.GetNotification(context.Background(), &pb.GetNotificationRequest{NotificationId: "n1", UserId: 1})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.Equal(t, readAt.Format(time.RFC3339), resp.ReadAt)
}
