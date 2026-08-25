package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

type mockWalletConnectionService struct {
	getLinkNonceFunc            func(context.Context, uint64, string) (string, error)
	linkWalletFunc              func(context.Context, uint64, string, string, string) (string, error)
	getSecurityNonceFunc        func(context.Context, uint64, string) (string, error)
	verifySecuritySignatureFunc func(context.Context, uint64, string, string, int32, string, string) (int64, error)
	checkRegisteredFunc         func(context.Context, string) (bool, string, error)
}

func (m *mockWalletConnectionService) GetLinkNonce(ctx context.Context, userID uint64, address string) (string, error) {
	if m.getLinkNonceFunc != nil {
		return m.getLinkNonceFunc(ctx, userID, address)
	}
	return "nonce", nil
}
func (m *mockWalletConnectionService) LinkWallet(ctx context.Context, userID uint64, address, signature, ip string) (string, error) {
	if m.linkWalletFunc != nil {
		return m.linkWalletFunc(ctx, userID, address, signature, ip)
	}
	return address, nil
}
func (m *mockWalletConnectionService) GetSecurityNonce(ctx context.Context, userID uint64, address string) (string, error) {
	if m.getSecurityNonceFunc != nil {
		return m.getSecurityNonceFunc(ctx, userID, address)
	}
	return "nonce", nil
}
func (m *mockWalletConnectionService) VerifySecuritySignature(ctx context.Context, userID uint64, address, signature string, duration int32, ip, ua string) (int64, error) {
	if m.verifySecuritySignatureFunc != nil {
		return m.verifySecuritySignatureFunc(ctx, userID, address, signature, duration, ip, ua)
	}
	return time.Now().Add(time.Hour).Unix(), nil
}

func (m *mockWalletConnectionService) CheckRegistered(ctx context.Context, walletAddress string) (bool, string, error) {
	if m.checkRegisteredFunc != nil {
		return m.checkRegisteredFunc(ctx, walletAddress)
	}
	return false, "", nil
}

var _ service.WalletConnectionService = (*mockWalletConnectionService)(nil)

func newWalletHandler(m *mockWalletConnectionService) pb.WalletConnectionServiceServer {
	return handler.RegisterWalletConnectionHandler(grpc.NewServer(), m, "en")
}

func TestWalletConnectionHandler(t *testing.T) {
	ctx := authenticatedContext(1)
	addr := "0x1111111111111111111111111111111111111111"

	t.Run("unauthenticated", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		_, err := h.GetLinkNonce(context.Background(), &pb.GetWalletLinkNonceRequest{Address: addr})
		st, _ := status.FromError(err)
		if st.Code() != codes.Unauthenticated {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("address required", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		_, err := h.GetLinkNonce(ctx, &pb.GetWalletLinkNonceRequest{})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("get link nonce success", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		resp, err := h.GetLinkNonce(ctx, &pb.GetWalletLinkNonceRequest{Address: addr})
		if err != nil || resp.Nonce != "nonce" {
			t.Fatalf("resp=%v err=%v", resp, err)
		}
	})

	t.Run("link wallet validation", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		_, err := h.LinkWallet(ctx, &pb.LinkWalletRequest{Address: addr})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("link wallet success and error map", func(t *testing.T) {
		m := &mockWalletConnectionService{}
		h := newWalletHandler(m)
		resp, err := h.LinkWallet(ctx, &pb.LinkWalletRequest{Address: addr, Signature: "sig"})
		if err != nil || resp.WalletAddress != addr {
			t.Fatalf("resp=%v err=%v", resp, err)
		}

		errorCases := []struct {
			err  error
			code codes.Code
		}{
			{service.ErrInvalidWalletAddress, codes.InvalidArgument},
			{service.ErrInvalidWalletSignature, codes.InvalidArgument},
			{service.ErrInvalidWalletSecurityDuration, codes.InvalidArgument},
			{service.ErrWalletAlreadyConnected, codes.FailedPrecondition},
			{service.ErrWalletAlreadyLinked, codes.FailedPrecondition},
			{service.ErrWalletNonceExpired, codes.FailedPrecondition},
			{service.ErrWalletSignatureFailed, codes.Unauthenticated},
			{service.ErrWalletNotConnectedToAccount, codes.PermissionDenied},
			{service.ErrUserNotFound, codes.NotFound},
			{errors.New("other"), codes.Internal},
		}
		for _, tc := range errorCases {
			m.linkWalletFunc = func(context.Context, uint64, string, string, string) (string, error) {
				return "", tc.err
			}
			_, err := h.LinkWallet(ctx, &pb.LinkWalletRequest{Address: addr, Signature: "sig"})
			st, _ := status.FromError(err)
			if st.Code() != tc.code {
				t.Fatalf("err=%v code=%v want=%v", tc.err, st.Code(), tc.code)
			}
		}
	})

	t.Run("security nonce and verify", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		resp, err := h.GetSecurityNonce(ctx, &pb.GetWalletSecurityNonceRequest{Address: addr})
		if err != nil || resp.Nonce == "" {
			t.Fatalf("resp=%v err=%v", resp, err)
		}

		_, err = h.VerifySecuritySignature(ctx, &pb.VerifyWalletSecuritySignatureRequest{
			Address: addr, Signature: "sig", Duration: 3,
		})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}

		ok, err := h.VerifySecuritySignature(ctx, &pb.VerifyWalletSecuritySignatureRequest{
			Address: addr, Signature: "sig", Duration: 15,
		})
		if err != nil || ok.Until == 0 {
			t.Fatalf("resp=%v err=%v", ok, err)
		}
	})

	t.Run("check registered validation", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		_, err := h.CheckRegistered(context.Background(), &pb.CheckWalletRegisteredRequest{})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("check registered found", func(t *testing.T) {
		m := &mockWalletConnectionService{
			checkRegisteredFunc: func(context.Context, string) (bool, string, error) {
				return true, "hm-123", nil
			},
		}
		h := newWalletHandler(m)
		resp, err := h.CheckRegistered(context.Background(), &pb.CheckWalletRegisteredRequest{WalletAddress: "0xtf"})
		if err != nil {
			t.Fatal(err)
		}
		if !resp.AlreadyRegistered || resp.GetUserCode() != "hm-123" {
			t.Fatalf("resp=%v", resp)
		}
	})

	t.Run("check registered not found", func(t *testing.T) {
		h := newWalletHandler(&mockWalletConnectionService{})
		resp, err := h.CheckRegistered(context.Background(), &pb.CheckWalletRegisteredRequest{WalletAddress: "0xtf"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.AlreadyRegistered || resp.UserCode != nil {
			t.Fatalf("resp=%v", resp)
		}
	})
}
