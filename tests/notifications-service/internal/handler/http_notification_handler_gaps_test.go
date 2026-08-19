package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/notifications-service/internal/handler"
	pb "metarang/shared/pb/notifications"
)

func passThroughAuth(next http.Handler) http.Handler { return next }

func TestHTTP_GetNotifications_PaginationDefaults(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantPage    int32
		wantPerPage int32
	}{
		{name: "omitted", query: "", wantPage: 1, wantPerPage: 100},
		{name: "invalid page", query: "?page=abc&per_page=20", wantPage: 1, wantPerPage: 20},
		{name: "zero page", query: "?page=0&per_page=20", wantPage: 1, wantPerPage: 20},
		{name: "negative page", query: "?page=-2&per_page=20", wantPage: 1, wantPerPage: 20},
		{name: "invalid per_page", query: "?page=3&per_page=nope", wantPage: 3, wantPerPage: 100},
		{name: "zero per_page", query: "?page=3&per_page=0", wantPage: 3, wantPerPage: 100},
		{name: "negative per_page", query: "?page=3&per_page=-5", wantPage: 3, wantPerPage: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := muxWithAPI(&mockNotificationAPI{
				GetNotificationsFunc: func(_ context.Context, req *pb.GetNotificationsRequest) (*pb.NotificationsResponse, error) {
					require.NotNil(t, req.Pagination)
					assert.Equal(t, tt.wantPage, req.Pagination.Page)
					assert.Equal(t, tt.wantPerPage, req.Pagination.PerPage)
					return &pb.NotificationsResponse{}, nil
				},
			}, 42)

			rr := serveRequest(mux, http.MethodGet, "/api/notifications"+tt.query, 42)
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestHTTP_UnauthorizedWithoutUser(t *testing.T) {
	httpH := newHTTPMuxPassThrough(&mockNotificationAPI{})

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodGet, "/api/notifications/n1"},
		{http.MethodPost, "/api/notifications/read/n1"},
		{http.MethodPost, "/api/notifications/read/all"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rr := serveRequest(httpH, tc.method, tc.path, 0)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
	}
}

func TestHTTP_MarkAsRead_MethodNotAllowed(t *testing.T) {
	mux := muxWithAPI(&mockNotificationAPI{}, 42)
	rr := serveRequest(mux, http.MethodGet, "/api/notifications/read/n1", 42)
	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestHTTP_TransformNotification(t *testing.T) {
	tests := []struct {
		name  string
		notif *pb.Notification
		check func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "single created_at token defaults time",
			notif: &pb.Notification{
				Id: "n1", Type: "system", Message: "hello", CreatedAt: "1403/01/15",
			},
			check: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "1403/01/15", body["date"])
				assert.Equal(t, "00:00:00", body["time"])
			},
		},
		{
			name: "empty created_at",
			notif: &pb.Notification{
				Id: "n2", Type: "system", Message: "hello",
			},
			check: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "", body["date"])
				assert.Equal(t, "", body["time"])
				assert.Nil(t, body["read_at"])
			},
		},
		{
			name: "read_at null string",
			notif: &pb.Notification{
				Id: "n3", Type: "system", Message: "hello", ReadAt: "null",
			},
			check: func(t *testing.T, body map[string]interface{}) {
				assert.Nil(t, body["read_at"])
			},
		},
		{
			name: "read_at set",
			notif: &pb.Notification{
				Id: "n4", Type: "system", Message: "hello", ReadAt: "2024-03-10T14:30:00Z",
			},
			check: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "2024-03-10T14:30:00Z", body["read_at"])
			},
		},
		{
			name: "existing data keys are preserved",
			notif: &pb.Notification{
				Id:      "n5",
				Type:    "trade",
				Message: "fallback-message",
				Data: map[string]string{
					"related-to":   "custom-related",
					"sender-name":  "Alice",
					"sender-image": "alice.png",
					"message":      "keep-me",
				},
			},
			check: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].(map[string]interface{})
				assert.Equal(t, "custom-related", data["related-to"])
				assert.Equal(t, "Alice", data["sender-name"])
				assert.Equal(t, "alice.png", data["sender-image"])
				assert.Equal(t, "keep-me", data["message"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := muxWithAPI(&mockNotificationAPI{
				GetNotificationFunc: func(context.Context, *pb.GetNotificationRequest) (*pb.Notification, error) {
					return tt.notif, nil
				},
			}, 42)

			rr := serveRequest(mux, http.MethodGet, "/api/notifications/"+tt.notif.Id, 42)
			require.Equal(t, http.StatusOK, rr.Code)

			var wrapped map[string]map[string]interface{}
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&wrapped))
			tt.check(t, wrapped["data"])
		})
	}
}

func newHTTPMuxPassThrough(api *mockNotificationAPI) *http.ServeMux {
	mux := http.NewServeMux()
	handler.NewHTTPNotificationHandler(api).RegisterHTTPRoutes(mux, passThroughAuth)
	return mux
}
