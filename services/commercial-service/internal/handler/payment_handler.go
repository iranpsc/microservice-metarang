package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/commercial-service/internal/service"
	pb "metargb/shared/pb/commercial"
)

type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	paymentService service.PaymentService
}

func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func RegisterPaymentHandler(grpcServer *grpc.Server, paymentService service.PaymentService) {
	handler := NewPaymentHandler(paymentService)
	pb.RegisterPaymentServiceServer(grpcServer, handler)
}

func (h *PaymentHandler) InitiatePayment(ctx context.Context, req *pb.InitiatePaymentRequest) (*pb.InitiatePaymentResponse, error) {
	paymentURL, orderID, transactionID, err := h.paymentService.InitiatePayment(ctx, req.UserId, req.Asset, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initiate payment: %v", err)
	}

	return &pb.InitiatePaymentResponse{
		PaymentUrl:    paymentURL,
		OrderId:       orderID,
		TransactionId: transactionID,
	}, nil
}

func (h *PaymentHandler) HandleCallback(ctx context.Context, req *pb.HandleCallbackRequest) (*pb.HandleCallbackResponse, error) {
	success, redirectURL, message, err := h.paymentService.HandleCallback(ctx, req.OrderId, req.Status, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to handle callback: %v", err)
	}

	return &pb.HandleCallbackResponse{
		Success:     success,
		RedirectUrl: redirectURL,
		Message:     message,
	}, nil
}

func (h *PaymentHandler) VerifyPayment(ctx context.Context, req *pb.VerifyPaymentRequest) (*pb.VerifyPaymentResponse, error) {
	success, statusCode, referenceID, cardHash, message, err := h.paymentService.VerifyPayment(ctx, req.Token, req.MerchantId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify payment: %v", err)
	}

	return &pb.VerifyPaymentResponse{
		Success:     success,
		Status:      statusCode,
		ReferenceId: referenceID,
		CardHash:    cardHash,
		Message:     message,
	}, nil
}
