package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

type KYCService interface {
	SubmitKYC(ctx context.Context, userID uint64, fname, lname, nationalCode, birthdate string) (*models.KYC, error)
	GetKYCStatus(ctx context.Context, userID uint64) (*models.KYC, error)
	VerifyBankAccount(ctx context.Context, userID uint64, bankName, shabaNum, cardNum string) (*models.BankAccount, error)
}

type kycService struct {
	kycRepo repository.KYCRepository
}

func NewKYCService(kycRepo repository.KYCRepository) KYCService {
	return &kycService{
		kycRepo: kycRepo,
	}
}

func (s *kycService) SubmitKYC(ctx context.Context, userID uint64, fname, lname, nationalCode, birthdate string) (*models.KYC, error) {
	// Check if KYC already exists
	existing, err := s.kycRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing kyc: %w", err)
	}

	// Parse birthdate
	parsedDate, err := time.Parse("2006-01-02", birthdate)
	if err != nil {
		return nil, fmt.Errorf("invalid birthdate format: %w", err)
	}

	if existing != nil {
		// Update existing KYC
		existing.Fname = fname
		existing.Lname = lname
		existing.NationalCode = nationalCode
		existing.Birthdate = sql.NullTime{Time: parsedDate, Valid: true}
		existing.Status = 0 // Pending verification

		if err := s.kycRepo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("failed to update kyc: %w", err)
		}
		return existing, nil
	}

	// Create new KYC
	kyc := &models.KYC{
		UserID:       userID,
		Fname:        fname,
		Lname:        lname,
		NationalCode: nationalCode,
		Status:       0, // Pending
		Birthdate:    sql.NullTime{Time: parsedDate, Valid: true},
	}

	if err := s.kycRepo.Create(ctx, kyc); err != nil {
		return nil, fmt.Errorf("failed to create kyc: %w", err)
	}

	return kyc, nil
}

func (s *kycService) GetKYCStatus(ctx context.Context, userID uint64) (*models.KYC, error) {
	kyc, err := s.kycRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get kyc status: %w", err)
	}
	if kyc == nil {
		return nil, fmt.Errorf("kyc not found")
	}
	return kyc, nil
}

func (s *kycService) VerifyBankAccount(ctx context.Context, userID uint64, bankName, shabaNum, cardNum string) (*models.BankAccount, error) {
	// TODO: Implement actual bank verification logic (e.g., call to bank API)
	
	bankAccount := &models.BankAccount{
		BankableType: "App\\Models\\User",
		BankableID:   userID,
		BankName:     bankName,
		ShabaNum:     shabaNum,
		CardNum:      cardNum,
		Status:       0, // Pending verification
		Errors:       sql.NullString{},
	}

	if err := s.kycRepo.CreateBankAccount(ctx, bankAccount); err != nil {
		return nil, fmt.Errorf("failed to create bank account: %w", err)
	}

	return bankAccount, nil
}

