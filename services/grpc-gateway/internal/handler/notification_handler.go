package handler

import (
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"metargb/grpc-gateway/internal/middleware"
	pb "metargb/shared/pb/auth"
	commonpb "metargb/shared/pb/common"
	notificationpb "metargb/shared/pb/notifications"
)

type NotificationHandler struct {
	notificationClient notificationpb.NotificationServiceClient
	authClient         pb.AuthServiceClient
}

func NewNotificationHandler(notificationConn *grpc.ClientConn, authConn *grpc.ClientConn) *NotificationHandler {
	return &NotificationHandler{
		notificationClient: notificationpb.NewNotificationServiceClient(notificationConn),
		authClient:         pb.NewAuthServiceClient(authConn),
	}
}

// GetNotifications handles GET /api/notifications
// Returns unread notifications for the authenticated user
func (h *NotificationHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from token
	userID, err := h.extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse pagination (optional)
	var page, perPage int32 = 1, 100 // Default to 100 per page for notifications
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 32); err == nil && parsed > 0 {
			page = int32(parsed)
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if parsed, err := strconv.ParseInt(pp, 10, 32); err == nil && parsed > 0 {
			perPage = int32(parsed)
		}
	}

	// Build gRPC request - unread_only defaults to true per API docs
	grpcReq := &notificationpb.GetNotificationsRequest{
		UserId:     userID,
		UnreadOnly: true, // API docs say GET /api/notifications returns unread only
		Pagination: &commonpb.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		},
	}

	// Call gRPC service
	resp, err := h.notificationClient.GetNotifications(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Transform response to match API docs format
	notifications := make([]map[string]interface{}, 0, len(resp.Notifications))
	for _, notif := range resp.Notifications {
		notifMap := h.transformNotification(notif)
		notifications = append(notifications, notifMap)
	}

	// Return array directly (not wrapped in object) per API docs
	writeJSON(w, http.StatusOK, notifications)
}

// GetNotification handles GET /api/notifications/{notification}
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract notification ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/notifications/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusBadRequest, "notification ID is required")
		return
	}

	// Extract user ID from token
	userID, err := h.extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Build gRPC request
	grpcReq := &notificationpb.GetNotificationRequest{
		NotificationId: path,
		UserId:         userID,
	}

	// Call gRPC service
	notif, err := h.notificationClient.GetNotification(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Transform response to match API docs format
	notifMap := h.transformNotification(notif)
	writeJSON(w, http.StatusOK, notifMap)
}

// MarkAsRead handles POST /api/notifications/read/{notification}
func (h *NotificationHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract notification ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/notifications/read/")
	if path == "" || path == r.URL.Path {
		writeError(w, http.StatusBadRequest, "notification ID is required")
		return
	}

	// Extract user ID from token
	userID, err := h.extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Build gRPC request
	grpcReq := &notificationpb.MarkAsReadRequest{
		NotificationId: path,
		UserId:         userID,
	}

	// Call gRPC service
	_, err = h.notificationClient.MarkAsRead(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return 204 No Content per API docs
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllAsRead handles POST /api/notifications/read/all
func (h *NotificationHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from token
	userID, err := h.extractUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Build gRPC request
	grpcReq := &notificationpb.MarkAllAsReadRequest{
		UserId: userID,
	}

	// Call gRPC service
	_, err = h.notificationClient.MarkAllAsRead(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Return 204 No Content per API docs
	w.WriteHeader(http.StatusNoContent)
}

// transformNotification transforms proto notification to API docs format
func (h *NotificationHandler) transformNotification(notif *notificationpb.Notification) map[string]interface{} {
	// Parse created_at Jalali datetime string and split into date and time
	var dateStr, timeStr string
	var readAt *string

	// Parse created_at (Jalali format: "Y/m/d H:m:s" or just "Y/m/d")
	if notif.CreatedAt != "" {
		parts := strings.Fields(notif.CreatedAt) // Split by whitespace
		if len(parts) >= 2 {
			dateStr = parts[0] // Y/m/d
			timeStr = parts[1] // H:m:s
		} else if len(parts) == 1 {
			// Only date provided, use default time
			dateStr = parts[0]
			timeStr = "00:00:00"
		}
	}

	// Handle read_at - proto returns empty string for unread, RFC3339 for read
	if notif.ReadAt != "" && notif.ReadAt != "null" {
		readAt = &notif.ReadAt
	} else {
		readAt = nil // null for unread notifications
	}

	// Build data object according to API docs format
	// The data field should contain: related-to, sender-name, sender-image, message
	dataObj := make(map[string]interface{})

	// Copy existing data fields if present
	for k, v := range notif.Data {
		dataObj[k] = v
	}

	// Ensure required fields exist in data (with defaults if not present)
	if _, ok := dataObj["related-to"]; !ok {
		dataObj["related-to"] = notif.Type // Use type as fallback
	}
	if _, ok := dataObj["sender-name"]; !ok {
		dataObj["sender-name"] = "متارنگ" // Default sender name
	}
	if _, ok := dataObj["sender-image"]; !ok {
		dataObj["sender-image"] = "" // Default empty image
	}
	if _, ok := dataObj["message"]; !ok {
		dataObj["message"] = notif.Message // Use message as fallback
	}

	result := map[string]interface{}{
		"id":      notif.Id,
		"data":    dataObj,
		"read_at": readAt,
		"date":    dateStr,
		"time":    timeStr,
	}

	return result
}

// Helper methods from other handlers

func (h *NotificationHandler) extractUserID(r *http.Request) (uint64, error) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		return 0, err
	}
	return userCtx.UserID, nil
}
