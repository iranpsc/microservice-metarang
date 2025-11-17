package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/notifications"

	"metargb/notifications-service/internal/errs"
	"metargb/notifications-service/internal/models"
	"metargb/notifications-service/internal/service"
)

// SMSHandler implements the gRPC SMSService.
type SMSHandler struct {
	pb.UnimplementedSMSServiceServer
	service service.SMSService
}

// RegisterSMSHandler registers the SMS handler with the gRPC server.
func RegisterSMSHandler(grpcServer *grpc.Server, svc service.SMSService) {
	handler := &SMSHandler{service: svc}
	pb.RegisterSMSServiceServer(grpcServer, handler)
}

func (h *SMSHandler) SendSMS(ctx context.Context, req *pb.SendSMSRequest) (*pb.SMSResponse, error) {
	if req.Phone == "" {
		return nil, status.Error(codes.InvalidArgument, "phone is required")
	}
	if req.Message == "" && req.Template == "" {
		return nil, status.Error(codes.InvalidArgument, "either message or template is required")
	}

	payload := models.SMSPayload{
		Phone:    req.Phone,
		Message:  req.Message,
		Template: req.Template,
		Tokens:   req.Tokens,
	}

	messageID, err := h.service.SendSMS(ctx, payload)
	if err != nil {
		return nil, handleSMSError(err)
	}

	return &pb.SMSResponse{
		Sent:      true,
		MessageId: messageID,
		Status:    "queued",
	}, nil
}

func (h *SMSHandler) SendOTP(ctx context.Context, req *pb.SendOTPRequest) (*pb.SMSResponse, error) {
	if req.Phone == "" {
		return nil, status.Error(codes.InvalidArgument, "phone is required")
	}
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	payload := models.OTPPayload{
		Phone:  req.Phone,
		Code:   req.Code,
		Reason: req.Reason,
	}

	messageID, err := h.service.SendOTP(ctx, payload)
	if err != nil {
		return nil, handleSMSError(err)
	}

	return &pb.SMSResponse{
		Sent:      true,
		MessageId: messageID,
		Status:    "queued",
	}, nil
}

func handleSMSError(err error) error {
	if errors.Is(err, errs.ErrNotImplemented) {
		return status.Error(codes.Unimplemented, err.Error())
	}
	return status.Errorf(codes.Internal, "sms service error: %v", err)
}
