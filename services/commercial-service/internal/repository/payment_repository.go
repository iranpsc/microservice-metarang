package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/commercial-service/internal/models"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	FindLatestByUserID(ctx context.Context, userID uint64) (*models.Payment, error)
}

type paymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (user_id, ref_id, card_pan, gateway, amount, product, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		payment.UserID, payment.RefID, payment.CardPan, payment.Gateway,
		payment.Amount, payment.Product, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	payment.ID = uint64(id)

	return nil
}

func (r *paymentRepository) FindLatestByUserID(ctx context.Context, userID uint64) (*models.Payment, error) {
	query := `
		SELECT id, user_id, ref_id, card_pan, gateway, amount, product, created_at, updated_at
		FROM payments
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	payment := &models.Payment{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&payment.ID, &payment.UserID, &payment.RefID, &payment.CardPan,
		&payment.Gateway, &payment.Amount, &payment.Product,
		&payment.CreatedAt, &payment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find latest payment: %w", err)
	}
	return payment, nil
}

