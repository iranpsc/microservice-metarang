package handler_test

import (
	"context"
	"errors"
	"testing"

	"metarang/commercial-service/internal/handler"
	pb "metarang/shared/pb/commercial"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubWalletService struct {
	wallet map[string]string
	err    error
}

func (s *stubWalletService) GetWallet(ctx context.Context, userID uint64) (map[string]string, error) {
	return s.wallet, s.err
}

func (s *stubWalletService) CreateWallet(ctx context.Context, userID uint64) (map[string]string, error) {
	return s.wallet, s.err
}

func (s *stubWalletService) DeductBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error) {
	return s.wallet, s.err
}

func (s *stubWalletService) AddBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error) {
	return s.wallet, s.err
}

func (s *stubWalletService) LockBalance(ctx context.Context, userID uint64, asset string, amount float64, reason string) error {
	return s.err
}

func (s *stubWalletService) UnlockBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	return s.err
}

func sampleWallet() map[string]string {
	return map[string]string{
		"psc": "10", "irr": "0", "red": "0", "blue": "0", "yellow": "0",
		"satisfaction": "0.10", "effect": "1.5",
	}
}

func TestWalletHandler_GetWallet(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{wallet: sampleWallet()})
	resp, err := h.GetWallet(context.Background(), &pb.GetWalletRequest{UserId: 1})
	require.NoError(t, err)
	assert.Equal(t, "10", resp.Psc)
	assert.Equal(t, 1.5, resp.Effect)
}

func TestWalletHandler_GetWallet_ServiceError(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{err: errors.New("fail")})
	_, err := h.GetWallet(context.Background(), &pb.GetWalletRequest{UserId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWalletHandler_CreateWallet_Validation(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{})
	_, err := h.CreateWallet(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = h.CreateWallet(context.Background(), &pb.CreateWalletRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestWalletHandler_CreateWallet_Success(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{wallet: sampleWallet()})
	resp, err := h.CreateWallet(context.Background(), &pb.CreateWalletRequest{UserId: 1})
	require.NoError(t, err)
	assert.Equal(t, "10", resp.Psc)
}

func TestWalletHandler_CreateWallet_ServiceError(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{err: errors.New("fail")})
	_, err := h.CreateWallet(context.Background(), &pb.CreateWalletRequest{UserId: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestWalletHandler_DeductBalance(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{wallet: sampleWallet()})
	resp, err := h.DeductBalance(context.Background(), &pb.DeductBalanceRequest{UserId: 1, Asset: "psc", Amount: 1})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	h = handler.NewWalletHandler(&stubWalletService{err: errors.New("insufficient")})
	resp, err = h.DeductBalance(context.Background(), &pb.DeductBalanceRequest{UserId: 1, Asset: "psc", Amount: 99})
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestWalletHandler_AddBalance(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{wallet: sampleWallet()})
	resp, err := h.AddBalance(context.Background(), &pb.AddBalanceRequest{UserId: 1, Asset: "psc", Amount: 5})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	h = handler.NewWalletHandler(&stubWalletService{err: errors.New("fail")})
	resp, err = h.AddBalance(context.Background(), &pb.AddBalanceRequest{UserId: 1, Asset: "psc", Amount: 5})
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestWalletHandler_LockUnlockBalance(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{})
	_, err := h.LockBalance(context.Background(), &pb.LockBalanceRequest{UserId: 1, Asset: "psc", Amount: 1, Reason: "test"})
	require.NoError(t, err)

	h = handler.NewWalletHandler(&stubWalletService{err: errors.New("fail")})
	_, err = h.LockBalance(context.Background(), &pb.LockBalanceRequest{UserId: 1, Asset: "psc", Amount: 1})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))

	h = handler.NewWalletHandler(&stubWalletService{})
	_, err = h.UnlockBalance(context.Background(), &pb.UnlockBalanceRequest{UserId: 1, Asset: "psc", Amount: 1})
	require.NoError(t, err)

	h = handler.NewWalletHandler(&stubWalletService{err: errors.New("fail")})
	_, err = h.UnlockBalance(context.Background(), &pb.UnlockBalanceRequest{UserId: 1, Asset: "psc", Amount: 1})
	require.Error(t, err)
}

func TestWalletHandler_EffectParseFallback(t *testing.T) {
	h := handler.NewWalletHandler(&stubWalletService{wallet: map[string]string{"effect": "bad"}})
	resp, err := h.GetWallet(context.Background(), &pb.GetWalletRequest{UserId: 1})
	require.NoError(t, err)
	assert.Equal(t, 0.0, resp.Effect)
}
