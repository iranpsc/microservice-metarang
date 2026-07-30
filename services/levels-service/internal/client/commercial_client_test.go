package client

import (
	"context"
	"errors"
	"net"
	"testing"

	pb "metarang/shared/pb/commercial"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type mockWalletServer struct {
	pb.UnimplementedWalletServiceServer
	resp *pb.AddBalanceResponse
	err  error
}

func (m *mockWalletServer) AddBalance(_ context.Context, _ *pb.AddBalanceRequest) (*pb.AddBalanceResponse, error) {
	return m.resp, m.err
}

func newBufconnClient(t *testing.T, srv pb.WalletServiceServer) CommercialClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterWalletServiceServer(s, srv)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(func() { s.Stop() })

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &commercialClient{
		walletClient: pb.NewWalletServiceClient(conn),
		conn:         conn,
	}
}

func TestNewCommercialClient_ReturnsClient(t *testing.T) {
	c, err := NewCommercialClient("127.0.0.1:65530")
	if err != nil {
		t.Fatalf("NewCommercialClient: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	requireClose(t, c)
}

func TestCommercialClient_AddBalance_Success(t *testing.T) {
	c := newBufconnClient(t, &mockWalletServer{
		resp: &pb.AddBalanceResponse{Success: true},
	})

	if err := c.AddBalance(context.Background(), 42, "psc", 100.5); err != nil {
		t.Fatalf("AddBalance: %v", err)
	}
}

func TestCommercialClient_AddBalance_RPCError(t *testing.T) {
	c := newBufconnClient(t, &mockWalletServer{
		err: errors.New("rpc failed"),
	})

	err := c.AddBalance(context.Background(), 1, "blue", 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommercialClient_AddBalance_NotSuccess(t *testing.T) {
	c := newBufconnClient(t, &mockWalletServer{
		resp: &pb.AddBalanceResponse{Success: false, Message: "insufficient funds"},
	})

	err := c.AddBalance(context.Background(), 1, "red", 5)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommercialClient_Close(t *testing.T) {
	t.Run("NilConn", func(t *testing.T) {
		c := &commercialClient{}
		if err := c.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	t.Run("ActiveConn", func(t *testing.T) {
		c := newBufconnClient(t, &mockWalletServer{})
		if err := c.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
}

func requireClose(t *testing.T, c CommercialClient) {
	t.Helper()
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
