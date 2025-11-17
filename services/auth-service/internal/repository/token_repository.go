package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"metargb/auth-service/internal/models"
)

type TokenRepository interface {
	Create(ctx context.Context, userID uint64, name string, expiresAt time.Time) (string, error)
	ValidateToken(ctx context.Context, token string) (*models.User, error)
	DeleteUserTokens(ctx context.Context, userID uint64) error
	FindTokenByHash(ctx context.Context, tokenHash string) (*models.PersonalAccessToken, error)
}

type tokenRepository struct {
	db *sql.DB
}

func NewTokenRepository(db *sql.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(ctx context.Context, userID uint64, name string, expiresAt time.Time) (string, error) {
	// Generate a random token (Sanctum-like format)
	plainToken := generatePlainToken()
	tokenHash := hashToken(plainToken)

	query := `
		INSERT INTO personal_access_tokens (tokenable_type, tokenable_id, name, token, abilities, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		"App\\Models\\User",
		userID,
		name,
		tokenHash,
		"[\"*\"]",
		expiresAt,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	tokenID, err := result.LastInsertId()
	if err != nil {
		return "", fmt.Errorf("failed to get token id: %w", err)
	}

	// Return token in Laravel Sanctum format: {id}|{plainToken}
	fullToken := fmt.Sprintf("%d|%s", tokenID, plainToken)
	return fullToken, nil
}

func (r *tokenRepository) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	tokenHash := hashToken(token)

	query := `
		SELECT pat.id, pat.tokenable_id, pat.expires_at, pat.last_used_at,
			   u.id, u.name, u.email, u.phone, u.password, u.code, u.referrer_id, u.score, u.ip,
			   u.last_seen, u.email_verified_at, u.phone_verified_at, u.access_token,
			   u.refresh_token, u.token_type, u.expires_in, u.created_at, u.updated_at
		FROM personal_access_tokens pat
		INNER JOIN users u ON pat.tokenable_id = u.id AND pat.tokenable_type = 'App\\Models\\User'
		WHERE pat.token = ?
	`

	var patID uint64
	var tokenableID uint64
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime
	user := &models.User{}

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&patID, &tokenableID, &expiresAt, &lastUsedAt,
		&user.ID, &user.Name, &user.Email, &user.Phone, &user.Password,
		&user.Code, &user.ReferrerID, &user.Score, &user.IP, &user.LastSeen,
		&user.EmailVerifiedAt, &user.PhoneVerifiedAt, &user.AccessToken,
		&user.RefreshToken, &user.TokenType, &user.ExpiresIn,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	// Check if token is expired
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// Update last_used_at
	go r.updateLastUsedAt(patID)

	return user, nil
}

func (r *tokenRepository) DeleteUserTokens(ctx context.Context, userID uint64) error {
	query := `DELETE FROM personal_access_tokens WHERE tokenable_id = ? AND tokenable_type = 'App\\Models\\User'`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user tokens: %w", err)
	}
	return nil
}

func (r *tokenRepository) FindTokenByHash(ctx context.Context, tokenHash string) (*models.PersonalAccessToken, error) {
	query := `
		SELECT id, tokenable_type, tokenable_id, name, token, abilities, last_used_at, expires_at, created_at, updated_at
		FROM personal_access_tokens
		WHERE token = ?
	`
	token := &models.PersonalAccessToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.TokenableType, &token.TokenableID, &token.Name,
		&token.Token, &token.Abilities, &token.LastUsedAt, &token.ExpiresAt,
		&token.CreatedAt, &token.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find token: %w", err)
	}
	return token, nil
}

func (r *tokenRepository) updateLastUsedAt(tokenID uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `UPDATE personal_access_tokens SET last_used_at = ? WHERE id = ?`
	_, _ = r.db.ExecContext(ctx, query, time.Now(), tokenID)
}

// generatePlainToken generates a random 40-character token
func generatePlainToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 40
	
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// hashToken creates SHA-256 hash of the token (Sanctum format)
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

