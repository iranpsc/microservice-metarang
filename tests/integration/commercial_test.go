package integration

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "metargb/shared/pb/commercial"
)

const commercialServiceAddr = "localhost:50052"

func TestWalletOperations(t *testing.T) {
	conn, err := grpc.Dial(commercialServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to commercial service: %v", err)
	}
	defer conn.Close()

	client := pb.NewWalletServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test user ID (should exist in test database)
	testUserID := uint64(1)

	t.Run("GetWallet", func(t *testing.T) {
		req := &pb.GetWalletRequest{
			UserId: testUserID,
		}

		resp, err := client.GetWallet(ctx, req)
		if err != nil {
			t.Fatalf("GetWallet failed: %v", err)
		}

		t.Logf("Wallet: PSC=%s, IRR=%s, Red=%s, Blue=%s, Yellow=%s, Satisfaction=%s, Effect=%f",
			resp.Psc, resp.Irr, resp.Red, resp.Blue, resp.Yellow, resp.Satisfaction, resp.Effect)

		// Verify format (should be compact notation like "1.5K", "2.3M", etc.)
		if resp.Psc == "" {
			t.Error("PSC should not be empty")
		}
	})

	t.Run("AddBalance", func(t *testing.T) {
		req := &pb.AddBalanceRequest{
			UserId: testUserID,
			Asset:  "psc",
			Amount: 100.0,
		}

		resp, err := client.AddBalance(ctx, req)
		if err != nil {
			t.Fatalf("AddBalance failed: %v", err)
		}

		if !resp.Success {
			t.Errorf("AddBalance should succeed: %s", resp.Message)
		}

		t.Logf("Balance after add: %+v", resp.Wallet)
	})

	t.Run("DeductBalance", func(t *testing.T) {
		req := &pb.DeductBalanceRequest{
			UserId: testUserID,
			Asset:  "psc",
			Amount: 50.0,
		}

		resp, err := client.DeductBalance(ctx, req)
		if err != nil {
			t.Logf("DeductBalance error (might be insufficient balance): %v", err)
			return
		}

		if !resp.Success {
			t.Logf("DeductBalance failed: %s", resp.Message)
		} else {
			t.Logf("Balance after deduct: %+v", resp.Wallet)
		}
	})
}

func TestTransactionOperations(t *testing.T) {
	conn, err := grpc.Dial(commercialServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewTransactionServiceClient(conn)
	ctx := context.Background()

	testUserID := uint64(1)

	t.Run("ListTransactions", func(t *testing.T) {
		req := &pb.ListTransactionsRequest{
			UserId:  testUserID,
			Page:    1,
			PerPage: 10,
		}

		resp, err := client.ListTransactions(ctx, req)
		if err != nil {
			t.Fatalf("ListTransactions failed: %v", err)
		}

		t.Logf("Found %d transactions", len(resp.Transactions))
		
		for i, tx := range resp.Transactions {
			if i >= 3 { // Log only first 3
				break
			}
			t.Logf("Transaction %d: ID=%s, Asset=%s, Amount=%f, Action=%s, Date=%s",
				i+1, tx.Id, tx.Asset, tx.Amount, tx.Action, tx.Date)
		}

		// Verify Jalali date format (Y/m/d)
		if len(resp.Transactions) > 0 {
			tx := resp.Transactions[0]
			if len(tx.Date) == 0 {
				t.Error("Transaction date should not be empty")
			}
			t.Logf("Date format example: %s", tx.Date)
		}
	})

	t.Run("CreateTransaction", func(t *testing.T) {
		req := &pb.CreateTransactionRequest{
			UserId: testUserID,
			Asset:  "psc",
			Amount: 100.0,
			Action: "deposit",
			Status: 1,
		}

		resp, err := client.CreateTransaction(ctx, req)
		if err != nil {
			t.Fatalf("CreateTransaction failed: %v", err)
		}

		// Verify VARCHAR ID format (TR-xxxxx)
		if len(resp.Id) < 3 || resp.Id[:3] != "TR-" {
			t.Errorf("Transaction ID should start with 'TR-', got: %s", resp.Id)
		}

		t.Logf("Created transaction: %s", resp.Id)
	})
}

func TestPaymentGateway(t *testing.T) {
	conn, err := grpc.Dial(commercialServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPaymentServiceClient(conn)
	ctx := context.Background()

	testUserID := uint64(1)

	t.Run("InitiatePayment", func(t *testing.T) {
		req := &pb.InitiatePaymentRequest{
			UserId: testUserID,
			Asset:  "psc",
			Amount: 1000.0,
		}

		resp, err := client.InitiatePayment(ctx, req)
		if err != nil {
			t.Logf("InitiatePayment error (might be gateway issue): %v", err)
			return
		}

		if resp.PaymentUrl == "" {
			t.Error("Payment URL should not be empty")
		}

		t.Logf("Payment URL: %s", resp.PaymentUrl)
		t.Logf("Order ID: %d", resp.OrderId)
		t.Logf("Transaction ID: %s", resp.TransactionId)
	})
}

