package handler

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"metargb/commercial-service/internal/models"
	"metargb/commercial-service/internal/service"
	pb "metargb/shared/pb/commercial"
)

type TransactionHandler struct {
	pb.UnimplementedTransactionServiceServer
	transactionService service.TransactionService
	orderRepo          interface{} // Simplified
	paymentRepo        interface{} // Simplified
}

func NewTransactionHandler(transactionService service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

func RegisterTransactionHandler(grpcServer *grpc.Server, transactionService service.TransactionService) {
	handler := NewTransactionHandler(transactionService)
	pb.RegisterTransactionServiceServer(grpcServer, handler)
}

func (h *TransactionHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
	filters := make(map[string]interface{})
	if req.Asset != "" {
		filters["asset"] = req.Asset
	}
	if req.Action != "" {
		filters["action"] = req.Action
	}
	if req.PerPage > 0 {
		filters["limit"] = int(req.PerPage)
	}

	transactions, err := h.transactionService.ListTransactions(ctx, req.UserId, filters)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list transactions: %v", err)
	}

	var resources []*pb.TransactionResource
	for _, t := range transactions {
		// Parse amount string to float64
		var amount float64
		fmt.Sscanf(t.Amount, "%f", &amount)
		
		resources = append(resources, &pb.TransactionResource{
			Id:     t.ID,
			Type:   t.Type,
			Asset:  t.Asset,
			Amount: amount,
			Action: t.Action,
			Status: t.Status,
			Date:   t.Date, // Already in Jalali format
			Time:   t.Time, // Already in Jalali format
		})
	}

	return &pb.ListTransactionsResponse{
		Transactions:  resources,
		CurrentPage:   req.Page,
		HasMorePages:  len(resources) >= int(req.PerPage),
	}, nil
}

func (h *TransactionHandler) GetLatestTransaction(ctx context.Context, req *pb.GetLatestTransactionRequest) (*pb.LatestTransactionResponse, error) {
	transaction, err := h.transactionService.GetLatestTransaction(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get latest transaction: %v", err)
	}

	response := &pb.LatestTransactionResponse{}
	
	if transaction != nil {
		response.LatestTransaction = &pb.Transaction{
			Id:      transaction.ID,
			UserId:  transaction.UserID,
			Asset:   transaction.Asset,
			Amount:  transaction.Amount,
			Action:  transaction.Action,
			Status:  transaction.Status,
			CreatedAt: timestamppb.New(transaction.CreatedAt),
			UpdatedAt: timestamppb.New(transaction.UpdatedAt),
		}
		
		if transaction.Token != nil {
			response.LatestTransaction.Token = *transaction.Token
		}
		if transaction.RefID != nil {
			response.LatestTransaction.RefId = *transaction.RefID
		}
		if transaction.PayableType != nil {
			response.LatestTransaction.PayableType = *transaction.PayableType
		}
		if transaction.PayableID != nil {
			response.LatestTransaction.PayableId = *transaction.PayableID
		}
	}

	return response, nil
}

func (h *TransactionHandler) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.Transaction, error) {
	transaction := &models.Transaction{
		UserID: req.UserId,
		Asset:  req.Asset,
		Amount: req.Amount,
		Action: req.Action,
		Status: req.Status,
	}

	if req.PayableType != "" {
		transaction.PayableType = &req.PayableType
	}
	if req.PayableId > 0 {
		transaction.PayableID = &req.PayableId
	}

	err := h.transactionService.CreateTransaction(ctx, transaction)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	return &pb.Transaction{
		Id:        transaction.ID,
		UserId:    transaction.UserID,
		Asset:     transaction.Asset,
		Amount:    transaction.Amount,
		Action:    transaction.Action,
		Status:    transaction.Status,
		CreatedAt: timestamppb.New(transaction.CreatedAt),
		UpdatedAt: timestamppb.New(transaction.UpdatedAt),
	}, nil
}

