package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
)

// Valid test data for Iranian bank accounts
// These are valid format but not real accounts
const (
	validShebaNum  = "6201600000000000080068121" // 24 digits (sample format)
	validCardNum   = "6037997551234567"           // Valid Luhn-algorithm card number
	validBankName  = "Tejarat"
	validBankName2 = "Melli"
)

func TestListBankAccounts(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty list when user has no accounts", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		accounts, err := service.ListBankAccounts(ctx, 1)
		if err != nil {
			t.Fatalf("ListBankAccounts returned error: %v", err)
		}
		if len(accounts) != 0 {
			t.Errorf("expected empty list, got %d accounts", len(accounts))
		}
	})

	t.Run("returns user's bank accounts", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		// Create test accounts
		account1 := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			BankName:     validBankName,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       0,
		}
		kycRepo.bankAccounts[1] = account1

		account2 := &models.BankAccount{
			ID:           2,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			BankName:     validBankName2,
			ShabaNum:     "820540102680020817909003",
			CardNum:      "6037997551234568",
			Status:       1,
		}
		kycRepo.bankAccounts[2] = account2

		// Create account for different user (should not appear)
		account3 := &models.BankAccount{
			ID:           3,
			BankableType: "App\\Models\\User",
			BankableID:   2,
			BankName:     validBankName,
			ShabaNum:     "820540102680020817909004",
			CardNum:      "6037997551234569",
			Status:       0,
		}
		kycRepo.bankAccounts[3] = account3

		accounts, err := service.ListBankAccounts(ctx, 1)
		if err != nil {
			t.Fatalf("ListBankAccounts returned error: %v", err)
		}
		if len(accounts) != 2 {
			t.Errorf("expected 2 accounts, got %d", len(accounts))
		}
	})
}

func TestCreateBankAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		account, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, validCardNum)
		if err != nil {
			t.Fatalf("CreateBankAccount returned error: %v", err)
		}
		if account == nil {
			t.Fatal("expected bank account to be created")
		}
		if account.BankName != validBankName {
			t.Errorf("expected BankName %q, got %q", validBankName, account.BankName)
		}
		if account.ShabaNum != validShebaNum {
			t.Errorf("expected ShabaNum %q, got %q", validShebaNum, account.ShabaNum)
		}
		if account.CardNum != validCardNum {
			t.Errorf("expected CardNum %q, got %q", validCardNum, account.CardNum)
		}
		if account.Status != 0 {
			t.Errorf("expected Status 0 (pending), got %d", account.Status)
		}
		if account.BankableType != "App\\Models\\User" {
			t.Errorf("expected BankableType 'App\\Models\\User', got %q", account.BankableType)
		}
		if account.BankableID != 1 {
			t.Errorf("expected BankableID 1, got %d", account.BankableID)
		}
		if account.ID == 0 {
			t.Error("expected account ID to be set")
		}
	})

	t.Run("requires verified user", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		unverifiedUser := &models.User{
			ID: 1,
			// No email or phone verified
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: unverifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for unverified user")
		}
		if !errors.Is(err, ErrUserNotVerified) {
			t.Errorf("expected ErrUserNotVerified, got %v", err)
		}
	})

	t.Run("allows phone verified user", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		phoneVerifiedUser := &models.User{
			ID:              1,
			PhoneVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: phoneVerifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		account, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, validCardNum)
		if err != nil {
			t.Fatalf("CreateBankAccount returned error: %v", err)
		}
		if account == nil {
			t.Fatal("expected bank account to be created")
		}
	})

	t.Run("validates bank name minimum length", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.CreateBankAccount(ctx, 1, "A", validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for bank name too short")
		}
		if !errors.Is(err, ErrInvalidBankName) {
			t.Errorf("expected ErrInvalidBankName, got %v", err)
		}
	})

	t.Run("validates bank name maximum length", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		longName := make([]byte, 256)
		for i := range longName {
			longName[i] = 'A'
		}

		_, err := service.CreateBankAccount(ctx, 1, string(longName), validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for bank name too long")
		}
		if !errors.Is(err, ErrInvalidBankName) {
			t.Errorf("expected ErrInvalidBankName, got %v", err)
		}
	})

	t.Run("validates sheba number format", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.CreateBankAccount(ctx, 1, validBankName, "INVALID_SHEBA", validCardNum)
		if err == nil {
			t.Fatal("expected error for invalid sheba number")
		}
		if !errors.Is(err, ErrInvalidShabaNum) {
			t.Errorf("expected ErrInvalidShabaNum, got %v", err)
		}
	})

	t.Run("validates card number format", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, "1234567890123456")
		if err == nil {
			t.Fatal("expected error for invalid card number (fails Luhn check)")
		}
		if !errors.Is(err, ErrInvalidCardNum) {
			t.Errorf("expected ErrInvalidCardNum, got %v", err)
		}
	})

	t.Run("validates sheba uniqueness", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		// Create existing account with same sheba
		existingAccount := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   2,
			ShabaNum:     validShebaNum,
			CardNum:      "6037997551234568",
		}
		kycRepo.bankAccounts[1] = existingAccount

		_, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for duplicate sheba number")
		}
		if !errors.Is(err, ErrShabaNumNotUnique) {
			t.Errorf("expected ErrShabaNumNotUnique, got %v", err)
		}
	})

	t.Run("validates card number uniqueness", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		// Create existing account with same card number
		existingAccount := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   2,
			ShabaNum:     "820540102680020817909003",
			CardNum:      validCardNum,
		}
		kycRepo.bankAccounts[1] = existingAccount

		_, err := service.CreateBankAccount(ctx, 1, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for duplicate card number")
		}
		if !errors.Is(err, ErrCardNumNotUnique) {
			t.Errorf("expected ErrCardNumNotUnique, got %v", err)
		}
	})

	t.Run("trims whitespace from inputs", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		account, err := service.CreateBankAccount(ctx, 1, "  "+validBankName+"  ", "  "+validShebaNum+"  ", "  "+validCardNum+"  ")
		if err != nil {
			t.Fatalf("CreateBankAccount returned error: %v", err)
		}
		if account.BankName != validBankName {
			t.Errorf("expected trimmed BankName %q, got %q", validBankName, account.BankName)
		}
		if account.ShabaNum != validShebaNum {
			t.Errorf("expected trimmed ShabaNum %q, got %q", validShebaNum, account.ShabaNum)
		}
		if account.CardNum != validCardNum {
			t.Errorf("expected trimmed CardNum %q, got %q", validCardNum, account.CardNum)
		}
	})

	t.Run("uppercases sheba number", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		verifiedUser := &models.User{
			ID:              1,
			EmailVerifiedAt: sql.NullTime{Time: time.Now(), Valid: true},
		}
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: verifiedUser})
		service := NewKYCService(kycRepo, userRepo)

		lowercaseSheba := "ir820540102680020817909002"
		account, err := service.CreateBankAccount(ctx, 1, validBankName, lowercaseSheba, validCardNum)
		if err != nil {
			t.Fatalf("CreateBankAccount returned error: %v", err)
		}
		if account.ShabaNum != validShebaNum {
			t.Errorf("expected uppercased ShabaNum %q, got %q", validShebaNum, account.ShabaNum)
		}
	})
}

func TestGetBankAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("returns account when found and owned", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			BankName:     validBankName,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       0,
		}
		kycRepo.bankAccounts[1] = account

		result, err := service.GetBankAccount(ctx, 1, 1)
		if err != nil {
			t.Fatalf("GetBankAccount returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected account to be found")
		}
		if result.ID != 1 {
			t.Errorf("expected ID 1, got %d", result.ID)
		}
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.GetBankAccount(ctx, 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent account")
		}
		if !errors.Is(err, ErrBankAccountNotFound) {
			t.Errorf("expected ErrBankAccountNotFound, got %v", err)
		}
	})

	t.Run("returns error when account not owned by user", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   2, // Different user
			BankName:     validBankName,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       0,
		}
		kycRepo.bankAccounts[1] = account

		_, err := service.GetBankAccount(ctx, 1, 1)
		if err == nil {
			t.Fatal("expected error for account not owned by user")
		}
		if !errors.Is(err, ErrBankAccountNotOwned) {
			t.Errorf("expected ErrBankAccountNotOwned, got %v", err)
		}
	})
}

func TestUpdateBankAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful update of rejected account", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		existingAccount := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			BankName:     validBankName,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       -1, // Rejected
			Errors:       sql.NullString{String: "Some error message", Valid: true},
		}
		kycRepo.bankAccounts[1] = existingAccount

		newSheba := "820540102680020817909003"
		newCard := "6037997551234568"
		updatedAccount, err := service.UpdateBankAccount(ctx, 1, 1, validBankName2, newSheba, newCard)
		if err != nil {
			t.Fatalf("UpdateBankAccount returned error: %v", err)
		}
		if updatedAccount.BankName != validBankName2 {
			t.Errorf("expected BankName %q, got %q", validBankName2, updatedAccount.BankName)
		}
		if updatedAccount.ShabaNum != newSheba {
			t.Errorf("expected ShabaNum %q, got %q", newSheba, updatedAccount.ShabaNum)
		}
		if updatedAccount.CardNum != newCard {
			t.Errorf("expected CardNum %q, got %q", newCard, updatedAccount.CardNum)
		}
		if updatedAccount.Status != 0 {
			t.Errorf("expected Status 0 (pending), got %d", updatedAccount.Status)
		}
		if updatedAccount.Errors.Valid {
			t.Error("expected errors to be cleared")
		}
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		_, err := service.UpdateBankAccount(ctx, 1, 999, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for non-existent account")
		}
		if !errors.Is(err, ErrBankAccountNotFound) {
			t.Errorf("expected ErrBankAccountNotFound, got %v", err)
		}
	})

	t.Run("returns error when account not owned", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   2, // Different user
			Status:       -1,
		}
		kycRepo.bankAccounts[1] = account

		_, err := service.UpdateBankAccount(ctx, 1, 1, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for account not owned")
		}
		if !errors.Is(err, ErrBankAccountNotOwned) {
			t.Errorf("expected ErrBankAccountNotOwned, got %v", err)
		}
	})

	t.Run("returns error when account is not rejected", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		testCases := []struct {
			name   string
			status int32
		}{
			{"pending", 0},
			{"verified", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				account := &models.BankAccount{
					ID:           1,
					BankableType: "App\\Models\\User",
					BankableID:   1,
					Status:       tc.status,
				}
				kycRepo.bankAccounts[1] = account

				_, err := service.UpdateBankAccount(ctx, 1, 1, validBankName, validShebaNum, validCardNum)
				if err == nil {
					t.Fatal("expected error for account not rejected")
				}
				if !errors.Is(err, ErrBankAccountNotRejected) {
					t.Errorf("expected ErrBankAccountNotRejected, got %v", err)
				}
			})
		}
	})

	t.Run("validates input fields", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			Status:       -1,
		}
		kycRepo.bankAccounts[1] = account

		testCases := []struct {
			name      string
			bankName  string
			shebaNum  string
			cardNum   string
			expectErr error
		}{
			{"invalid bank name short", "A", validShebaNum, validCardNum, ErrInvalidBankName},
			{"invalid sheba", validBankName, "INVALID", validCardNum, ErrInvalidShabaNum},
			{"invalid card", validBankName, validShebaNum, "1234567890123456", ErrInvalidCardNum},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := service.UpdateBankAccount(ctx, 1, 1, tc.bankName, tc.shebaNum, tc.cardNum)
				if err == nil {
					t.Fatal("expected validation error")
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("expected %v, got %v", tc.expectErr, err)
				}
			})
		}
	})

	t.Run("validates uniqueness excluding current account", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		existingAccount := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       -1,
		}
		kycRepo.bankAccounts[1] = existingAccount

		// Create another account with different numbers
		otherAccount := &models.BankAccount{
			ID:           2,
			BankableType: "App\\Models\\User",
			BankableID:   2,
			ShabaNum:     "820540102680020817909003",
			CardNum:      "6037997551234568",
			Status:       0,
		}
		kycRepo.bankAccounts[2] = otherAccount

		// Should allow updating to same values (uniqueness check excludes current ID)
		_, err := service.UpdateBankAccount(ctx, 1, 1, validBankName, validShebaNum, validCardNum)
		if err != nil {
			t.Errorf("expected to allow same values, got error: %v", err)
		}
	})

	t.Run("rejects duplicate sheba from other account", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		existingAccount := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			ShabaNum:     "820540102680020817909003",
			CardNum:      validCardNum,
			Status:       -1,
		}
		kycRepo.bankAccounts[1] = existingAccount

		otherAccount := &models.BankAccount{
			ID:           2,
			BankableType: "App\\Models\\User",
			BankableID:   2,
			ShabaNum:     validShebaNum,
			CardNum:      "6037997551234568",
			Status:       0,
		}
		kycRepo.bankAccounts[2] = otherAccount

		_, err := service.UpdateBankAccount(ctx, 1, 1, validBankName, validShebaNum, validCardNum)
		if err == nil {
			t.Fatal("expected error for duplicate sheba")
		}
		if !errors.Is(err, ErrShabaNumNotUnique) {
			t.Errorf("expected ErrShabaNumNotUnique, got %v", err)
		}
	})
}

func TestDeleteBankAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   1,
			BankName:     validBankName,
			ShabaNum:     validShebaNum,
			CardNum:      validCardNum,
			Status:       1, // Verified - can still delete
		}
		kycRepo.bankAccounts[1] = account

		err := service.DeleteBankAccount(ctx, 1, 1)
		if err != nil {
			t.Fatalf("DeleteBankAccount returned error: %v", err)
		}

		// Verify account is deleted
		deletedAccount, _ := kycRepo.FindBankAccountByID(ctx, 1)
		if deletedAccount != nil {
			t.Error("expected account to be deleted")
		}
	})

	t.Run("returns error when account not found", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		err := service.DeleteBankAccount(ctx, 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent account")
		}
		if !errors.Is(err, ErrBankAccountNotFound) {
			t.Errorf("expected ErrBankAccountNotFound, got %v", err)
		}
	})

	t.Run("returns error when account not owned", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		account := &models.BankAccount{
			ID:           1,
			BankableType: "App\\Models\\User",
			BankableID:   2, // Different user
			Status:       0,
		}
		kycRepo.bankAccounts[1] = account

		err := service.DeleteBankAccount(ctx, 1, 1)
		if err == nil {
			t.Fatal("expected error for account not owned")
		}
		if !errors.Is(err, ErrBankAccountNotOwned) {
			t.Errorf("expected ErrBankAccountNotOwned, got %v", err)
		}
	})

	t.Run("allows deletion regardless of status", func(t *testing.T) {
		kycRepo := newFakeKYCRepository()
		userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
		service := NewKYCService(kycRepo, userRepo)

		testCases := []struct {
			name   string
			status int32
		}{
			{"rejected", -1},
			{"pending", 0},
			{"verified", 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				accountID := uint64(tc.status + 10) // Ensure unique IDs
				account := &models.BankAccount{
					ID:           accountID,
					BankableType: "App\\Models\\User",
					BankableID:   1,
					Status:       tc.status,
				}
				kycRepo.bankAccounts[accountID] = account

				err := service.DeleteBankAccount(ctx, 1, accountID)
				if err != nil {
					t.Errorf("expected deletion to succeed for status %d, got error: %v", tc.status, err)
				}
			})
		}
	})
}

func TestBankAccountHelperMethods(t *testing.T) {
	t.Run("Pending method", func(t *testing.T) {
		account := &models.BankAccount{Status: 0}
		if !account.Pending() {
			t.Error("expected Pending() to return true for status 0")
		}
		if account.Verified() {
			t.Error("expected Verified() to return false for status 0")
		}
		if account.Rejected() {
			t.Error("expected Rejected() to return false for status 0")
		}
	})

	t.Run("Verified method", func(t *testing.T) {
		account := &models.BankAccount{Status: 1}
		if account.Pending() {
			t.Error("expected Pending() to return false for status 1")
		}
		if !account.Verified() {
			t.Error("expected Verified() to return true for status 1")
		}
		if account.Rejected() {
			t.Error("expected Rejected() to return false for status 1")
		}
	})

	t.Run("Rejected method", func(t *testing.T) {
		account := &models.BankAccount{Status: -1}
		if account.Pending() {
			t.Error("expected Pending() to return false for status -1")
		}
		if account.Verified() {
			t.Error("expected Verified() to return false for status -1")
		}
		if !account.Rejected() {
			t.Error("expected Rejected() to return true for status -1")
		}
	})
}

