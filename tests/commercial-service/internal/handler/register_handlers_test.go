package handler_test

import (
	"testing"

	"metarang/commercial-service/internal/handler"
	"metarang/commercial-service/internal/service"

	"google.golang.org/grpc"
)

func TestRegisterHandlers_Smoke(t *testing.T) {
	s := grpc.NewServer()
	t.Cleanup(s.Stop)

	handler.RegisterWalletHandler(s, &stubWalletService{})
	handler.RegisterTransactionHandler(s, &stubTransactionService{})
	handler.RegisterReferralHandler(s, &stubReferralService{})
	handler.RegisterUserVariableHandler(s, &stubUserVariableService{})
	handler.RegisterWalletHistoryHandler(s, service.NewWalletHistoryService(nil, nil, nil))

	info := s.GetServiceInfo()
	for _, name := range []string{
		"commercial.WalletService",
		"commercial.TransactionService",
		"commercial.ReferralService",
		"commercial.UserVariableService",
		"commercial.WalletHistoryService",
	} {
		if _, ok := info[name]; !ok {
			t.Fatalf("expected service %s to be registered, got %#v", name, info)
		}
	}
}
