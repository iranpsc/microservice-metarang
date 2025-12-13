package handler

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbCommon "metargb/shared/pb/common"
	pb "metargb/shared/pb/notifications"

	"metargb/notifications-service/internal/errs"
	"metargb/notifications-service/internal/models"
	"metargb/notifications-service/internal/service"
)

// NotificationHandler implements the gRPC NotificationService.
type NotificationHandler struct {
	pb.UnimplementedNotificationServiceServer
	service service.NotificationService
}

// RegisterNotificationHandler registers the notification handler with the gRPC server.
func RegisterNotificationHandler(grpcServer *grpc.Server, svc service.NotificationService) {
	handler := &NotificationHandler{service: svc}
	pb.RegisterNotificationServiceServer(grpcServer, handler)
}

func (h *NotificationHandler) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.NotificationResponse, error) {
	// #region agent log
	logEntry, _ := json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_handler.go:32", "message": "SendNotification handler entry",
		"data": map[string]interface{}{"userId": req.UserId, "type": req.Type, "title": req.Title},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Message == "" {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	input := service.SendNotificationInput{
		UserID:    req.UserId,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Data:      req.Data,
		SendSMS:   req.SendSms,
		SendEmail: req.SendEmail,
	}

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_handler.go:56", "message": "Before service.SendNotification call",
		"data": map[string]interface{}{"userID": input.UserID, "type": input.Type, "title": input.Title},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	result, err := h.service.SendNotification(ctx, input)

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "A",
		"location": "notification_handler.go:65", "message": "After service.SendNotification call",
		"data": map[string]interface{}{"result": result, "error": err != nil, "errorMsg": func() string { if err != nil { return err.Error() } else { return "" } }()},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	if err != nil {
		return nil, handleServiceError(err)
	}

	return &pb.NotificationResponse{
		Id:   result.ID,
		Sent: result.Sent,
	}, nil
}

func (h *NotificationHandler) GetNotifications(ctx context.Context, req *pb.GetNotificationsRequest) (*pb.NotificationsResponse, error) {
	// #region agent log
	logEntry, _ := json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "B",
		"location": "notification_handler.go:67", "message": "GetNotifications handler entry",
		"data": map[string]interface{}{"userId": req.UserId},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	filter := models.NotificationFilter{
		Page:    1,
		PerPage: 10,
	}

	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			filter.Page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			filter.PerPage = req.Pagination.PerPage
		}
	}

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "B",
		"location": "notification_handler.go:86", "message": "Before service.GetNotifications call",
		"data": map[string]interface{}{"filter": filter},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	notifications, total, err := h.service.GetNotifications(ctx, req.UserId, filter)

	// #region agent log
	logEntry, _ = json.Marshal(map[string]interface{}{
		"sessionId": "debug-session", "runId": "run1", "hypothesisId": "B",
		"location": "notification_handler.go:95", "message": "After service.GetNotifications call",
		"data": map[string]interface{}{"count": len(notifications), "total": total, "error": err != nil, "errorMsg": func() string { if err != nil { return err.Error() } else { return "" } }()},
		"timestamp": time.Now().UnixMilli(),
	})
	if f, err := os.OpenFile("d:\\microservice-metarang\\.cursor\\debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.Write(append(logEntry, '\n'))
		f.Close()
	}
	// #endregion

	if err != nil {
		return nil, handleServiceError(err)
	}

	response := &pb.NotificationsResponse{
		Notifications: make([]*pb.Notification, 0, len(notifications)),
		Pagination: &pbCommon.PaginationMeta{
			CurrentPage: filter.Page,
			PerPage:     filter.PerPage,
			Total:       int32(total),
		},
	}

	for _, notification := range notifications {
		response.Notifications = append(response.Notifications, convertNotification(notification))
	}

	return response, nil
}

func (h *NotificationHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pbCommon.Empty, error) {
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := h.service.MarkAsRead(ctx, req.NotificationId, req.UserId); err != nil {
		return nil, handleServiceError(err)
	}

	return &pbCommon.Empty{}, nil
}

func (h *NotificationHandler) MarkAllAsRead(ctx context.Context, req *pb.MarkAllAsReadRequest) (*pbCommon.Empty, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if err := h.service.MarkAllAsRead(ctx, req.UserId); err != nil {
		return nil, handleServiceError(err)
	}

	return &pbCommon.Empty{}, nil
}

func convertNotification(notification models.Notification) *pb.Notification {
	protoNotification := &pb.Notification{
		Id:      notification.ID,
		Type:    notification.Type,
		Title:   notification.Title,
		Message: notification.Message,
		Data:    notification.Data,
	}

	if !notification.CreatedAt.IsZero() {
		protoNotification.CreatedAt = notification.CreatedAt.Format(time.RFC3339)
	}

	if notification.ReadAt != nil {
		protoNotification.ReadAt = notification.ReadAt.Format(time.RFC3339)
	}

	return protoNotification
}

func handleServiceError(err error) error {
	if errors.Is(err, errs.ErrNotImplemented) {
		return status.Error(codes.Unimplemented, err.Error())
	}
	return status.Errorf(codes.Internal, "service error: %v", err)
}
