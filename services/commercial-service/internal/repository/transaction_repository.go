package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/commercial-service/internal/models"
)

type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) error
	Update(ctx context.Context, transaction *models.Transaction) error
	FindByID(ctx context.Context, id string) (*models.Transaction, error)
	FindLatestByUserID(ctx context.Context, userID uint64) (*models.Transaction, error)
	FindByUserID(ctx context.Context, userID uint64, filters map[string]interface{}) ([]*models.Transaction, error)
}

type transactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, user_id, asset, amount, action, status, token, ref_id, payable_type, payable_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		transaction.ID, transaction.UserID, transaction.Asset, transaction.Amount,
		transaction.Action, transaction.Status, transaction.Token, transaction.RefID,
		transaction.PayableType, transaction.PayableID, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) Update(ctx context.Context, transaction *models.Transaction) error {
	query := `
		UPDATE transactions
		SET user_id = ?, asset = ?, amount = ?, action = ?, status = ?, token = ?, ref_id = ?, payable_type = ?, payable_id = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		transaction.UserID,
		transaction.Asset,
		transaction.Amount,
		transaction.Action,
		transaction.Status,
		transaction.Token,
		transaction.RefID,
		transaction.PayableType,
		transaction.PayableID,
		time.Now(),
		transaction.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) FindByID(ctx context.Context, id string) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, asset, amount, action, status, token, ref_id, payable_type, payable_id, created_at, updated_at
		FROM transactions
		WHERE id = ?
	`
	transaction := &models.Transaction{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID, &transaction.UserID, &transaction.Asset, &transaction.Amount,
		&transaction.Action, &transaction.Status, &transaction.Token, &transaction.RefID,
		&transaction.PayableType, &transaction.PayableID,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}
	return transaction, nil
}

func (r *transactionRepository) FindLatestByUserID(ctx context.Context, userID uint64) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, asset, amount, action, status, token, ref_id, payable_type, payable_id, created_at, updated_at
		FROM transactions
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	transaction := &models.Transaction{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&transaction.ID, &transaction.UserID, &transaction.Asset, &transaction.Amount,
		&transaction.Action, &transaction.Status, &transaction.Token, &transaction.RefID,
		&transaction.PayableType, &transaction.PayableID,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find latest transaction: %w", err)
	}
	return transaction, nil
}

func (r *transactionRepository) FindByUserID(ctx context.Context, userID uint64, filters map[string]interface{}) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, asset, amount, action, status, token, ref_id, payable_type, payable_id, created_at, updated_at
		FROM transactions
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	// Add filters (simplified for now)
	if asset, ok := filters["asset"].(string); ok && asset != "" {
		query += " AND asset = ?"
		args = append(args, asset)
	}
	if action, ok := filters["action"].(string); ok && action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		transaction := &models.Transaction{}
		err := rows.Scan(
			&transaction.ID, &transaction.UserID, &transaction.Asset, &transaction.Amount,
			&transaction.Action, &transaction.Status, &transaction.Token, &transaction.RefID,
			&transaction.PayableType, &transaction.PayableID,
			&transaction.CreatedAt, &transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
