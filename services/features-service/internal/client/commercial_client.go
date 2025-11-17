package client

import (
	"context"
	"fmt"
	"time"

	pb "metargb/shared/pb/commercial"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CommercialClient wraps gRPC clients for Commercial Service
type CommercialClient struct {
	walletClient      pb.WalletServiceClient
	transactionClient pb.TransactionServiceClient
	conn              *grpc.ClientConn
}

// NewCommercialClient creates a new Commercial Service client
func NewCommercialClient(address string) (*CommercialClient, error) {
	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to commercial service at %s: %w", address, err)
	}

	return &CommercialClient{
		walletClient:      pb.NewWalletServiceClient(conn),
		transactionClient: pb.NewTransactionServiceClient(conn),
		conn:              conn,
	}, nil
}

// Close closes the gRPC connection
func (c *CommercialClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// UpdateWallet updates a user's wallet balance (add or deduct)
// Positive amount = add, negative amount = deduct
func (c *CommercialClient) UpdateWallet(ctx context.Context, userID uint64, asset string, amount float64) error {
	if amount > 0 {
		return c.AddBalance(ctx, userID, asset, amount)
	} else if amount < 0 {
		return c.DeductBalance(ctx, userID, asset, -amount) // Make positive for deduct
	}
	return nil // Zero amount, no-op
}

// AddBalance adds balance to a user's wallet
func (c *CommercialClient) AddBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	req := &pb.AddBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount,
	}

	resp, err := c.walletClient.AddBalance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add balance: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("add balance failed: %s", resp.Message)
	}

	return nil
}

// DeductBalance deducts balance from a user's wallet
func (c *CommercialClient) DeductBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	req := &pb.DeductBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount,
	}

	resp, err := c.walletClient.DeductBalance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("deduct balance failed: %s", resp.Message)
	}

	return nil
}

// GetWallet retrieves a user's wallet information
func (c *CommercialClient) GetWallet(ctx context.Context, userID uint64) (*pb.WalletResponse, error) {
	req := &pb.GetWalletRequest{
		UserId: userID,
	}

	resp, err := c.walletClient.GetWallet(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return resp, nil
}

// CreateTransaction creates a transaction record
func (c *CommercialClient) CreateTransaction(ctx context.Context, userID uint64, asset string, amount float64, action string, status int32, payableType string, payableID uint64) (*pb.Transaction, error) {
	req := &pb.CreateTransactionRequest{
		UserId:      userID,
		Asset:       asset,
		Amount:      amount,
		Action:      action,
		Status:      status,
		PayableType: payableType,
		PayableId:   payableID,
	}

	resp, err := c.transactionClient.CreateTransaction(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return resp, nil
}

// LockBalance locks balance for a pending transaction
func (c *CommercialClient) LockBalance(ctx context.Context, userID uint64, asset string, amount float64, reason string) error {
	req := &pb.LockBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount,
		Reason: reason,
	}

	_, err := c.walletClient.LockBalance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to lock balance: %w", err)
	}

	return nil
}

// UnlockBalance unlocks previously locked balance
func (c *CommercialClient) UnlockBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	req := &pb.UnlockBalanceRequest{
		UserId: userID,
		Asset:  asset,
		Amount: amount,
	}

	_, err := c.walletClient.UnlockBalance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to unlock balance: %w", err)
	}

	return nil
}

// CheckBalance verifies if user has sufficient balance
// Returns true if balance >= required amount
func (c *CommercialClient) CheckBalance(ctx context.Context, userID uint64, asset string, requiredAmount float64) (bool, error) {
	wallet, err := c.GetWallet(ctx, userID)
	if err != nil {
		return false, err
	}

	var balance float64
	switch asset {
	case "psc":
		balance = parseWalletString(wallet.Psc)
	case "irr":
		balance = parseWalletString(wallet.Irr)
	case "red":
		balance = parseWalletString(wallet.Red)
	case "blue":
		balance = parseWalletString(wallet.Blue)
	case "yellow":
		balance = parseWalletString(wallet.Yellow)
	default:
		return false, fmt.Errorf("unknown asset: %s", asset)
	}

	return balance >= requiredAmount, nil
}

// parseWalletString converts formatted wallet string to float
// Handles compact notation like "1.5K", "2.3M"
func parseWalletString(s string) float64 {
	// TODO: Implement proper parsing of compact notation
	// For now, this is a placeholder
	// In production, this should parse strings like "1.5K" -> 1500.0
	return 0
}

