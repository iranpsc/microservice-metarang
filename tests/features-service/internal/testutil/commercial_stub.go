package testutil

import (
	"context"
	"sync"

	pb "metarang/shared/pb/commercial"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BalanceOp records a wallet add/deduct call.
type BalanceOp struct {
	UserID uint64
	Asset  string
	Amount float64
}

// CreateTxOp records a CreateTransaction call.
type CreateTxOp struct {
	UserID      uint64
	Asset       string
	Amount      float64
	Action      string
	Status      int32
	PayableType string
	PayableID   uint64
}

// CommercialStub is a configurable WalletServiceClient + TransactionServiceClient for tests.
type CommercialStub struct {
	pb.WalletServiceClient
	pb.TransactionServiceClient

	mu sync.Mutex

	Psc    string
	Irr    string
	Yellow string
	Red    string
	Blue   string

	GetWalletErr   error
	FailNthDeduct  int // 1-based; 0 = never fail
	FailNthAdd     int // 1-based; 0 = never fail
	FailAddGRPC    error
	FailDeductGRPC error

	GetWalletCalls []uint64
	DeductCalls    []BalanceOp
	AddCalls       []BalanceOp
	CreateTxCalls  []CreateTxOp

	deductN int
	addN    int
}

func NewCommercialStub() *CommercialStub {
	return &CommercialStub{
		Psc:    "1000000",
		Irr:    "1000000",
		Yellow: "1000000",
		Red:    "1000000",
		Blue:   "1000000",
	}
}

func (s *CommercialStub) GetWallet(_ context.Context, in *pb.GetWalletRequest, _ ...grpc.CallOption) (*pb.WalletResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GetWalletCalls = append(s.GetWalletCalls, in.GetUserId())
	if s.GetWalletErr != nil {
		return nil, s.GetWalletErr
	}
	return &pb.WalletResponse{
		Psc:    s.Psc,
		Irr:    s.Irr,
		Yellow: s.Yellow,
		Red:    s.Red,
		Blue:   s.Blue,
	}, nil
}

func (s *CommercialStub) CreateWallet(context.Context, *pb.CreateWalletRequest, ...grpc.CallOption) (*pb.WalletResponse, error) {
	return &pb.WalletResponse{}, nil
}

func (s *CommercialStub) DeductBalance(_ context.Context, in *pb.DeductBalanceRequest, _ ...grpc.CallOption) (*pb.DeductBalanceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deductN++
	s.DeductCalls = append(s.DeductCalls, BalanceOp{UserID: in.GetUserId(), Asset: in.GetAsset(), Amount: in.GetAmount()})
	if s.FailDeductGRPC != nil {
		return nil, s.FailDeductGRPC
	}
	if s.FailNthDeduct > 0 && s.deductN == s.FailNthDeduct {
		return &pb.DeductBalanceResponse{Success: false, Message: "deduct failed"}, nil
	}
	return &pb.DeductBalanceResponse{Success: true}, nil
}

func (s *CommercialStub) AddBalance(_ context.Context, in *pb.AddBalanceRequest, _ ...grpc.CallOption) (*pb.AddBalanceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addN++
	s.AddCalls = append(s.AddCalls, BalanceOp{UserID: in.GetUserId(), Asset: in.GetAsset(), Amount: in.GetAmount()})
	if s.FailAddGRPC != nil {
		return nil, s.FailAddGRPC
	}
	if s.FailNthAdd > 0 && s.addN == s.FailNthAdd {
		return &pb.AddBalanceResponse{Success: false, Message: "add failed"}, nil
	}
	return &pb.AddBalanceResponse{Success: true}, nil
}

func (s *CommercialStub) LockBalance(context.Context, *pb.LockBalanceRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *CommercialStub) UnlockBalance(context.Context, *pb.UnlockBalanceRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *CommercialStub) ListTransactions(context.Context, *pb.ListTransactionsRequest, ...grpc.CallOption) (*pb.ListTransactionsResponse, error) {
	return &pb.ListTransactionsResponse{}, nil
}

func (s *CommercialStub) GetLatestTransaction(context.Context, *pb.GetLatestTransactionRequest, ...grpc.CallOption) (*pb.LatestTransactionResponse, error) {
	return &pb.LatestTransactionResponse{}, nil
}

func (s *CommercialStub) CreateTransaction(_ context.Context, in *pb.CreateTransactionRequest, _ ...grpc.CallOption) (*pb.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CreateTxCalls = append(s.CreateTxCalls, CreateTxOp{
		UserID:      in.GetUserId(),
		Asset:       in.GetAsset(),
		Amount:      in.GetAmount(),
		Action:      in.GetAction(),
		Status:      in.GetStatus(),
		PayableType: in.GetPayableType(),
		PayableID:   in.GetPayableId(),
	})
	return &pb.Transaction{Id: "1"}, nil
}
