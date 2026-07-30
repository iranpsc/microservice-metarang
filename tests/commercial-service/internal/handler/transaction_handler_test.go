package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metarang/commercial-service/internal/handler"
	"metarang/commercial-service/internal/models"
	pb "metarang/shared/pb/commercial"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubTransactionService struct {
	list      []*models.TransactionDTO
	latest    *models.Transaction
	listErr   error
	getErr    error
	createErr error
}

func (s *stubTransactionService) ListTransactions(ctx context.Context, userID uint64, filters map[string]interface{}) ([]*models.TransactionDTO, error) {
	return s.list, s.listErr
}

func (s *stubTransactionService) GetLatestTransaction(ctx context.Context, userID uint64) (*models.Transaction, error) {
	return s.latest, s.getErr
}

func (s *stubTransactionService) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	if s.createErr != nil {
		return s.createErr
	}
	transaction.ID = "TR-test"
	transaction.CreatedAt = time.Now()
	transaction.UpdatedAt = time.Now()
	return nil
}

func TestTransactionHandler_ListTransactions(t *testing.T) {
	h := handler.NewTransactionHandler(&stubTransactionService{
		list: []*models.TransactionDTO{
			{ID: "TR-1", Asset: "psc", Amount: "10", Action: "deposit", Status: 1, Date: "1404/01/01", Time: "12:00:00", Type: "order"},
			{ID: "TR-2", Asset: "psc", Amount: "5", Action: "deposit", Status: 1, Date: "1404/01/02", Time: "12:00:00", Type: "trade"},
		},
	})
	resp, err := h.ListTransactions(context.Background(), &pb.ListTransactionsRequest{
		UserId: 1,
		Search: "TR-1",
		Status: []int32{1},
		Action: "deposit",
		Asset:  "psc",
		Type:   "order",
	})
	require.NoError(t, err)
	assert.Len(t, resp.Transactions, 2)
	assert.False(t, resp.HasMorePages)

	h = handler.NewTransactionHandler(&stubTransactionService{listErr: errors.New("fail")})
	_, err = h.ListTransactions(context.Background(), &pb.ListTransactionsRequest{UserId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestTransactionHandler_ListTransactions_Pagination(t *testing.T) {
	items := make([]*models.TransactionDTO, 16)
	for i := range items {
		items[i] = &models.TransactionDTO{ID: "TR-x", Amount: "1"}
	}
	h := handler.NewTransactionHandler(&stubTransactionService{list: items})
	resp, err := h.ListTransactions(context.Background(), &pb.ListTransactionsRequest{UserId: 1, PerPage: 15, Page: 1})
	require.NoError(t, err)
	assert.Len(t, resp.Transactions, 15)
	assert.True(t, resp.HasMorePages)
}

func TestTransactionHandler_GetLatestTransaction(t *testing.T) {
	token := int64(99)
	refID := int64(88)
	payableType := `App\Models\Order`
	payableID := uint64(7)
	now := time.Now()

	h := handler.NewTransactionHandler(&stubTransactionService{
		latest: &models.Transaction{
			ID: "TR-1", UserID: 1, Asset: "psc", Amount: 10, Action: "deposit", Status: 1,
			Token: &token, RefID: &refID, PayableType: &payableType, PayableID: &payableID,
			CreatedAt: now, UpdatedAt: now,
		},
	})
	resp, err := h.GetLatestTransaction(context.Background(), &pb.GetLatestTransactionRequest{UserId: 1})
	require.NoError(t, err)
	require.NotNil(t, resp.LatestTransaction)
	assert.Equal(t, "TR-1", resp.LatestTransaction.Id)
	assert.Equal(t, int64(99), resp.LatestTransaction.Token)

	h = handler.NewTransactionHandler(&stubTransactionService{})
	resp, err = h.GetLatestTransaction(context.Background(), &pb.GetLatestTransactionRequest{UserId: 1})
	require.NoError(t, err)
	assert.Nil(t, resp.LatestTransaction)

	h = handler.NewTransactionHandler(&stubTransactionService{getErr: errors.New("fail")})
	_, err = h.GetLatestTransaction(context.Background(), &pb.GetLatestTransactionRequest{UserId: 1})
	require.Error(t, err)
}

func TestTransactionHandler_CreateTransaction(t *testing.T) {
	h := handler.NewTransactionHandler(&stubTransactionService{})
	resp, err := h.CreateTransaction(context.Background(), &pb.CreateTransactionRequest{
		UserId: 1, Asset: "psc", Amount: 10, Action: "deposit", Status: 1,
		PayableType: `App\Models\Order`, PayableId: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, "TR-test", resp.Id)

	h = handler.NewTransactionHandler(&stubTransactionService{createErr: errors.New("fail")})
	_, err = h.CreateTransaction(context.Background(), &pb.CreateTransactionRequest{UserId: 1, Asset: "psc", Amount: 1, Action: "deposit", Status: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
