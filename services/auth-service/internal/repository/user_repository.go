package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metargb/auth-service/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint64) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateLastSeen(ctx context.Context, userID uint64) error
	FindByCode(ctx context.Context, code string) (*models.User, error)
	GetSettings(ctx context.Context, userID uint64) (*models.Settings, error)
	CreateSettings(ctx context.Context, settings *models.Settings) error
	GetKYC(ctx context.Context, userID uint64) (*models.KYC, error)
	GetUnreadNotificationsCount(ctx context.Context, userID uint64) (int32, error)
	MarkEmailAsVerified(ctx context.Context, userID uint64) error
	UpdatePhone(ctx context.Context, userID uint64, phone string) error
	MarkPhoneAsVerified(ctx context.Context, userID uint64) error
	IsPhoneTaken(ctx context.Context, phone string, excludeUserID uint64) (bool, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (name, email, phone, password, code, ip, referrer_id, 
			access_token, refresh_token, token_type, expires_in, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		user.Name, user.Email, user.Phone, user.Password, user.Code, user.IP,
		user.ReferrerID, user.AccessToken, user.RefreshToken, user.TokenType,
		user.ExpiresIn, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	user.ID = uint64(id)

	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, email, phone, password, code, referrer_id, score, ip, 
			last_seen, email_verified_at, phone_verified_at, access_token, 
			refresh_token, token_type, expires_in, created_at, updated_at
		FROM users
		WHERE email = ?
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone, &user.Password,
		&user.Code, &user.ReferrerID, &user.Score, &user.IP, &user.LastSeen,
		&user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.AccessToken,
		&user.RefreshToken, &user.TokenType, &user.ExpiresIn,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uint64) (*models.User, error) {
	query := `
		SELECT id, name, email, phone, password, code, referrer_id, score, ip, 
			last_seen, email_verified_at, phone_verified_at, access_token, 
			refresh_token, token_type, expires_in, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone, &user.Password,
		&user.Code, &user.ReferrerID, &user.Score, &user.IP, &user.LastSeen,
		&user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.AccessToken,
		&user.RefreshToken, &user.TokenType, &user.ExpiresIn,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET name = ?, email = ?, phone = ?, access_token = ?, refresh_token = ?,
			token_type = ?, expires_in = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		user.Name, user.Email, user.Phone, user.AccessToken, user.RefreshToken,
		user.TokenType, user.ExpiresIn, time.Now(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateLastSeen(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET last_seen = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}
	return nil
}

func (r *userRepository) FindByCode(ctx context.Context, code string) (*models.User, error) {
	query := `SELECT id FROM users WHERE code = ?`
	var id uint64
	err := r.db.QueryRowContext(ctx, query, code).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by code: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *userRepository) GetSettings(ctx context.Context, userID uint64) (*models.Settings, error) {
	query := `SELECT id, user_id, automatic_logout FROM settings WHERE user_id = ?`
	settings := &models.Settings{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&settings.ID, &settings.UserID, &settings.AutomaticLogout,
	)
	if err == sql.ErrNoRows {
		// Return default settings
		return &models.Settings{
			UserID:          userID,
			AutomaticLogout: 55,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}
	return settings, nil
}

func (r *userRepository) GetKYC(ctx context.Context, userID uint64) (*models.KYC, error) {
	query := `
		SELECT id, user_id, fname, lname, national_code, status, birthdate, created_at, updated_at
		FROM kycs WHERE user_id = ?
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
		return nil, fmt.Errorf("failed to get kyc: %w", err)
	}
	return kyc, nil
}

func (r *userRepository) GetUnreadNotificationsCount(ctx context.Context, userID uint64) (int32, error) {
	query := `
		SELECT COUNT(*) FROM notifications 
		WHERE notifiable_type = 'App\\Models\\User' 
		AND notifiable_id = ? 
		AND read_at IS NULL
	`
	var count int32
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notifications count: %w", err)
	}
	return count, nil
}

func (r *userRepository) CreateSettings(ctx context.Context, settings *models.Settings) error {
	query := `INSERT INTO settings (user_id, automatic_logout) VALUES (?, ?)`
	result, err := r.db.ExecContext(ctx, query, settings.UserID, settings.AutomaticLogout)
	if err != nil {
		return fmt.Errorf("failed to create settings: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	settings.ID = uint64(id)

	return nil
}

func (r *userRepository) MarkEmailAsVerified(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET email_verified_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to mark email as verified: %w", err)
	}
	return nil
}

func (r *userRepository) UpdatePhone(ctx context.Context, userID uint64, phone string) error {
	query := `UPDATE users SET phone = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, phone, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user phone: %w", err)
	}
	return nil
}

func (r *userRepository) MarkPhoneAsVerified(ctx context.Context, userID uint64) error {
	query := `UPDATE users SET phone_verified_at = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to mark phone as verified: %w", err)
	}
	return nil
}

func (r *userRepository) IsPhoneTaken(ctx context.Context, phone string, excludeUserID uint64) (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE phone = ? AND id != ?`
	var count int
	if err := r.db.QueryRowContext(ctx, query, phone, excludeUserID).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check phone uniqueness: %w", err)
	}
	return count > 0, nil
}
