package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"metargb/commercial-service/internal/repository"
	"metargb/shared/pkg/helpers"
)

type WalletService interface {
	GetWallet(ctx context.Context, userID uint64) (map[string]string, error)
	DeductBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error)
	AddBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error)
	LockBalance(ctx context.Context, userID uint64, asset string, amount float64, reason string) error
	UnlockBalance(ctx context.Context, userID uint64, asset string, amount float64) error
}

type walletService struct {
	walletRepo repository.WalletRepository
}

func NewWalletService(walletRepo repository.WalletRepository) WalletService {
	return &walletService{
		walletRepo: walletRepo,
	}
}

func (s *walletService) GetWallet(ctx context.Context, userID uint64) (map[string]string, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	// Convert decimals to float64 for formatting
	pscFloat, _ := wallet.PSC.Float64()
	irrFloat, _ := wallet.IRR.Float64()
	redFloat, _ := wallet.Red.Float64()
	blueFloat, _ := wallet.Blue.Float64()
	yellowFloat, _ := wallet.Yellow.Float64()
	satisfactionFloat, _ := wallet.Satisfaction.Float64()

	// Format exactly as Laravel WalletResource does
	// Laravel: 'psc' => formatCompactNumber($this->psc)
	return map[string]string{
		"psc":          helpers.FormatCompactNumber(pscFloat),
		"irr":          helpers.FormatCompactNumber(irrFloat),
		"red":          helpers.FormatCompactNumber(redFloat),
		"blue":         helpers.FormatCompactNumber(blueFloat),
		"yellow":       helpers.FormatCompactNumber(yellowFloat),
		"satisfaction": helpers.NumberFormat(satisfactionFloat, 1), // Laravel: number_format($this->satisfaction, 1)
		"effect":       wallet.Effect.String(),                     // effect is returned as-is
	}, nil
}

func (s *walletService) DeductBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error) {
	amountDec := decimal.NewFromFloat(amount)
	
	err := s.walletRepo.DeductBalance(ctx, userID, asset, amountDec)
	if err != nil {
		return nil, fmt.Errorf("failed to deduct balance: %w", err)
	}

	return s.GetWallet(ctx, userID)
}

func (s *walletService) AddBalance(ctx context.Context, userID uint64, asset string, amount float64) (map[string]string, error) {
	amountDec := decimal.NewFromFloat(amount)
	
	err := s.walletRepo.AddBalance(ctx, userID, asset, amountDec)
	if err != nil {
		return nil, fmt.Errorf("failed to add balance: %w", err)
	}

	return s.GetWallet(ctx, userID)
}

func (s *walletService) LockBalance(ctx context.Context, userID uint64, asset string, amount float64, reason string) error {
	amountDec := decimal.NewFromFloat(amount)
	
	err := s.walletRepo.LockBalance(ctx, userID, asset, amountDec, reason)
	if err != nil {
		return fmt.Errorf("failed to lock balance: %w", err)
	}

	return nil
}

func (s *walletService) UnlockBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	amountDec := decimal.NewFromFloat(amount)
	
	err := s.walletRepo.UnlockBalance(ctx, userID, asset, amountDec)
	if err != nil {
		return fmt.Errorf("failed to unlock balance: %w", err)
	}

	return nil
}

