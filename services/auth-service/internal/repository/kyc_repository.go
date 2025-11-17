package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type KYCRepository interface {
	Create(ctx context.Context, kyc *models.KYC) error
	FindByUserID(ctx context.Context, userID uint64) (*models.KYC, error)
	Update(ctx context.Context, kyc *models.KYC) error
	CreateBankAccount(ctx context.Context, bankAccount *models.BankAccount) error
	FindBankAccountsByUserID(ctx context.Context, userID uint64) ([]*models.BankAccount, error)
}

type kycRepository struct {
	db *sql.DB
}

func NewKYCRepository(db *sql.DB) KYCRepository {
	return &kycRepository{db: db}
}

func (r *kycRepository) Create(ctx context.Context, kyc *models.KYC) error {
	query := `
		INSERT INTO kycs (user_id, fname, lname, national_code, status, birthdate, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		kyc.UserID, kyc.Fname, kyc.Lname, kyc.NationalCode, kyc.Status,
		kyc.Birthdate, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create kyc: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	kyc.ID = uint64(id)

	return nil
}

func (r *kycRepository) FindByUserID(ctx context.Context, userID uint64) (*models.KYC, error) {
	query := `
		SELECT id, user_id, fname, lname, national_code, status, birthdate, created_at, updated_at
		FROM kycs
		WHERE user_id = ?
	`
	kyc := &models.KYC{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&kyc.ID, &kyc.UserID, &kyc.Fname, &kyc.Lname, &kyc.NationalCode,
		&kyc.Status, &kyc.Birthdate, &kyc.CreatedAt, &kyc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find kyc: %w", err)
	}
	return kyc, nil
}

func (r *kycRepository) Update(ctx context.Context, kyc *models.KYC) error {
	query := `
		UPDATE kycs
		SET fname = ?, lname = ?, national_code = ?, status = ?, birthdate = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		kyc.Fname, kyc.Lname, kyc.NationalCode, kyc.Status, kyc.Birthdate, time.Now(), kyc.ID)
	if err != nil {
		return fmt.Errorf("failed to update kyc: %w", err)
	}
	return nil
}

func (r *kycRepository) CreateBankAccount(ctx context.Context, bankAccount *models.BankAccount) error {
	query := `
		INSERT INTO bank_accounts (bankable_type, bankable_id, bank_name, shaba_num, card_num, status, errors, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		bankAccount.BankableType, bankAccount.BankableID, bankAccount.BankName,
		bankAccount.ShabaNum, bankAccount.CardNum, bankAccount.Status,
		bankAccount.Errors, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create bank account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	bankAccount.ID = uint64(id)

	return nil
}

func (r *kycRepository) FindBankAccountsByUserID(ctx context.Context, userID uint64) ([]*models.BankAccount, error) {
	query := `
		SELECT id, bankable_type, bankable_id, bank_name, shaba_num, card_num, status, errors, created_at, updated_at
		FROM bank_accounts
		WHERE bankable_type = 'App\\Models\\User' AND bankable_id = ?
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find bank accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*models.BankAccount
	for rows.Next() {
		account := &models.BankAccount{}
		err := rows.Scan(
			&account.ID, &account.BankableType, &account.BankableID,
			&account.BankName, &account.ShabaNum, &account.CardNum,
			&account.Status, &account.Errors, &account.CreatedAt, &account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bank account: %w", err)
		}
		accounts = append(accounts, account)
	}

	return accounts, nil
}

