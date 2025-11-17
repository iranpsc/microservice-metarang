package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
	notificationspb "metargb/shared/pb/notifications"
)

type AuthService interface {
	Register(ctx context.Context, backURL, referral string) (string, error)
	Redirect(ctx context.Context, redirectTo, backURL string) (string, string, error) // returns url and state
	Callback(ctx context.Context, state, code, cachedState string) (*CallbackResult, error)
	GetMe(ctx context.Context, token string) (*UserDetails, error)
	Logout(ctx context.Context, userID uint64, ip, userAgent string) error
	ValidateToken(ctx context.Context, token string) (*models.User, error)
	RequestAccountSecurity(ctx context.Context, userID uint64, minutes int32, phone string) error
	VerifyAccountSecurity(ctx context.Context, userID uint64, code, ip, userAgent string) error
}

type authService struct {
	userRepo            repository.UserRepository
	tokenRepo           repository.TokenRepository
	accountSecurityRepo repository.AccountSecurityRepository
	activityRepo        repository.ActivityRepository
	observerService     ObserverService
	helperService       HelperService
	notificationsClient notificationspb.SMSServiceClient
	oauthServerURL      string
	oauthClientID       string
	oauthClientSecret   string
	httpClient          *http.Client
}

type CallbackResult struct {
	Token       string
	ExpiresAt   int32
	RedirectURL string
}

type UserDetails struct {
	ID                         uint64
	Name                       string
	Token                      string
	AccessToken                string
	AutomaticLogout            int32
	Code                       string
	Level                      *LevelInfo
	Image                      string
	Notifications              int32
	ScorePercentageToNextLevel float64
	UnansweredQuestionsCount   int32
	HourlyProfitTimePercentage float64
	VerifiedKYC                bool
	Birthdate                  string
}

type LevelInfo struct {
	ID          uint64
	Title       string
	Description string
	Score       int32
}

var (
	ErrAccountSecurityNotFound        = errors.New("account security not found")
	ErrAccountSecurityAlreadyUnlocked = errors.New("account security already unlocked")
	ErrInvalidOTPCode                 = errors.New("invalid verification code")
	ErrPhoneRequired                  = errors.New("phone number is required")
	ErrInvalidPhoneFormat             = errors.New("invalid phone format")
	ErrPhoneAlreadyTaken              = errors.New("phone already in use")
	ErrUserNotFound                   = errors.New("user not found")
	ErrInvalidUnlockDuration          = errors.New("invalid unlock duration")
)

var (
	iranMobileRegex = regexp.MustCompile(`^09\d{9}$`)
	otpCodeRegex    = regexp.MustCompile(`^\d{6}$`)
)

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	accountSecurityRepo repository.AccountSecurityRepository,
	activityRepo repository.ActivityRepository,
	observerService ObserverService,
	helperService HelperService,
	notificationsClient notificationspb.SMSServiceClient,
	oauthServerURL, oauthClientID, oauthClientSecret string,
) AuthService {
	return &authService{
		userRepo:            userRepo,
		tokenRepo:           tokenRepo,
		accountSecurityRepo: accountSecurityRepo,
		activityRepo:        activityRepo,
		observerService:     observerService,
		helperService:       helperService,
		notificationsClient: notificationsClient,
		oauthServerURL:      oauthServerURL,
		oauthClientID:       oauthClientID,
		oauthClientSecret:   oauthClientSecret,
		httpClient:          &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *authService) Register(ctx context.Context, backURL, referral string) (string, error) {
	params := url.Values{}
	params.Set("client_id", s.oauthClientID)
	params.Set("redirect_uri", s.oauthServerURL+"/auth/redirect") // This should be your callback URL
	params.Set("referral", referral)
	params.Set("back_url", backURL)

	redirectURL := fmt.Sprintf("%s/register?%s", s.oauthServerURL, params.Encode())
	return redirectURL, nil
}

func (s *authService) Redirect(ctx context.Context, redirectTo, backURL string) (string, string, error) {
	// Generate state token
	state, err := generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	params := url.Values{}
	params.Set("client_id", s.oauthClientID)
	params.Set("redirect_uri", s.oauthServerURL+"/auth/callback")
	params.Set("response_type", "code")
	params.Set("scope", "")
	params.Set("state", state)

	authURL := fmt.Sprintf("%s/oauth/authorize?%s", s.oauthServerURL, params.Encode())
	return authURL, state, nil
}

func (s *authService) Callback(ctx context.Context, state, code, cachedState string) (*CallbackResult, error) {
	// Validate state
	if state != cachedState {
		return nil, fmt.Errorf("invalid state value")
	}

	// Exchange code for token
	tokenData, err := s.exchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user data from OAuth server
	userData, err := s.getUserFromOAuth(ctx, tokenData.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	// Create or update user
	user, err := s.userRepo.FindByEmail(ctx, userData.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Get referrer ID if provided
	var referrerID sql.NullInt64
	if userData.Referral != "" {
		referrer, err := s.userRepo.FindByCode(ctx, userData.Referral)
		if err == nil && referrer != nil {
			referrerID = sql.NullInt64{Int64: int64(referrer.ID), Valid: true}
		}
	}

	isNewUser := user == nil

	if isNewUser {
		// Create new user
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(generateRandomString(10)), bcrypt.DefaultCost)
		user = &models.User{
			Name:         userData.Name,
			Email:        userData.Email,
			Phone:        userData.Mobile,
			Password:     string(hashedPassword),
			Code:         userData.Code,
			IP:           "", // Should be set from request context
			ReferrerID:   referrerID,
			AccessToken:  sql.NullString{String: tokenData.AccessToken, Valid: true},
			RefreshToken: sql.NullString{String: tokenData.RefreshToken, Valid: true},
			TokenType:    sql.NullString{String: tokenData.TokenType, Valid: true},
			ExpiresIn:    sql.NullInt64{Int64: tokenData.ExpiresIn, Valid: true},
		}
		err = s.userRepo.Create(ctx, user)

		// Trigger user created observer (creates settings, log, initial activity)
		if err == nil && s.observerService != nil {
			if obsErr := s.observerService.OnUserCreated(ctx, user); obsErr != nil {
				// Log error but don't fail the registration
				fmt.Printf("observer error on user creation: %v\n", obsErr)
			}
		}

		// TODO: Call Commercial service to create wallet and user_variables
		// This should be done via gRPC:
		// - commercialClient.CreateWallet(ctx, &pb.CreateWalletRequest{UserId: user.ID})
		// - commercialClient.CreateUserVariables(ctx, &pb.CreateUserVariablesRequest{UserId: user.ID})
	} else {
		// Update existing user
		user.Name = userData.Name
		user.Email = userData.Email
		user.Phone = userData.Mobile
		user.AccessToken = sql.NullString{String: tokenData.AccessToken, Valid: true}
		user.RefreshToken = sql.NullString{String: tokenData.RefreshToken, Valid: true}
		user.TokenType = sql.NullString{String: tokenData.TokenType, Valid: true}
		user.ExpiresIn = sql.NullInt64{Int64: tokenData.ExpiresIn, Valid: true}
		err = s.userRepo.Update(ctx, user)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Get user settings
	settings, err := s.userRepo.GetSettings(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Create Sanctum token
	automaticLogout := settings.AutomaticLogout
	if automaticLogout == 0 {
		automaticLogout = 55
	}
	expiresAt := time.Now().Add(time.Duration(automaticLogout) * time.Minute)

	token, err := s.tokenRepo.Create(ctx, user.ID, fmt.Sprintf("token_%d", user.ID), expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	// Extract just the token part (after the |)
	tokenParts := splitToken(token)
	plainToken := tokenParts[1]

	// Trigger login observer (activity tracking, events, notifications, WebSocket)
	// Note: IP and UserAgent should be extracted from gRPC metadata
	if s.observerService != nil {
		if err := s.observerService.OnUserLogin(ctx, user, user.IP, ""); err != nil {
			// Log error but don't fail the login
			fmt.Printf("observer error on login: %v\n", err)
		}
	}

	result := &CallbackResult{
		Token:       plainToken,
		ExpiresAt:   int32(time.Until(expiresAt).Minutes()),
		RedirectURL: "", // Should be populated from cache
	}

	return result, nil
}

func (s *authService) GetMe(ctx context.Context, token string) (*UserDetails, error) {
	user, err := s.tokenRepo.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Update last seen
	_ = s.userRepo.UpdateLastSeen(ctx, user.ID)

	// Get settings
	settings, err := s.userRepo.GetSettings(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Get KYC
	kyc, err := s.userRepo.GetKYC(ctx, user.ID)

	// Get unread notifications count
	notificationsCount, _ := s.userRepo.GetUnreadNotificationsCount(ctx, user.ID)

	// Prepare user details
	details := &UserDetails{
		ID:              user.ID,
		Name:            user.Name,
		Token:           token,
		Code:            user.Code,
		AutomaticLogout: settings.AutomaticLogout,
		Notifications:   notificationsCount,
		VerifiedKYC:     kyc != nil && kyc.Status == 1,
	}

	if user.AccessToken.Valid {
		details.AccessToken = user.AccessToken.String
	}

	if kyc != nil && kyc.Status == 1 {
		details.Name = kyc.FullName()
		if kyc.Birthdate.Valid {
			// Format as Jalali date Y/m/d
			// Import shared helpers for Jalali formatting
			// For now, using simple format - TODO: integrate shared/pkg/helpers/jalali.go
			details.Birthdate = kyc.Birthdate.Time.Format("2006/01/02")
		}
	}

	// Get level, score percentage, unanswered questions, hourly profit percentage
	// These require integration with Levels and Features services
	if s.helperService != nil {
		// Get user level
		level, err := s.helperService.GetUserLevel(ctx, user.ID)
		if err == nil && level != nil {
			details.Level = level
		}

		// Get score percentage to next level
		scorePercentage, err := s.helperService.GetScorePercentageToNextLevel(ctx, user.ID, user.Score)
		if err == nil {
			details.ScorePercentageToNextLevel = scorePercentage
		}

		// Get unanswered questions count
		unansweredCount, err := s.helperService.GetUnansweredQuestionsCount(ctx, user.ID)
		if err == nil {
			details.UnansweredQuestionsCount = unansweredCount
		}

		// Get hourly profit time percentage
		profitPercentage, err := s.helperService.GetHourlyProfitTimePercentage(ctx, user.ID)
		if err == nil {
			details.HourlyProfitTimePercentage = profitPercentage
		}
	}

	// TODO: Get profile image (from images table with polymorphic relation)
	// This would require either a separate repository or a join query

	return details, nil
}

func (s *authService) Logout(ctx context.Context, userID uint64, ip, userAgent string) error {
	// Get user first
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Trigger logout observer (activity tracking, score calculation, WebSocket)
	if s.observerService != nil {
		if err := s.observerService.OnUserLogout(ctx, user, ip, userAgent); err != nil {
			// Log error but don't fail the logout
			fmt.Printf("observer error on logout: %v\n", err)
		}
	}

	// Delete tokens
	return s.tokenRepo.DeleteUserTokens(ctx, userID)
}

func (s *authService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	return s.tokenRepo.ValidateToken(ctx, token)
}

func (s *authService) RequestAccountSecurity(ctx context.Context, userID uint64, minutes int32, phone string) error {
	if minutes < 5 || minutes > 60 {
		return ErrInvalidUnlockDuration
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	lengthSeconds := int64(minutes) * 60

	security, err := s.accountSecurityRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load account security: %w", err)
	}

	if security == nil {
		security = &models.AccountSecurity{
			UserID:   userID,
			Unlocked: false,
			Until:    sql.NullInt64{},
			Length:   lengthSeconds,
		}
		if err := s.accountSecurityRepo.Create(ctx, security); err != nil {
			return fmt.Errorf("failed to create account security: %w", err)
		}
	} else {
		security.Unlocked = false
		security.Until = sql.NullInt64{}
		security.Length = lengthSeconds
		if err := s.accountSecurityRepo.Update(ctx, security); err != nil {
			return fmt.Errorf("failed to update account security: %w", err)
		}
	}

	if !user.PhoneVerifiedAt.Valid {
		sanitizedPhone := strings.TrimSpace(phone)
		if sanitizedPhone == "" {
			return ErrPhoneRequired
		}
		if !iranMobileRegex.MatchString(sanitizedPhone) {
			return ErrInvalidPhoneFormat
		}

		taken, err := s.userRepo.IsPhoneTaken(ctx, sanitizedPhone, user.ID)
		if err != nil {
			return fmt.Errorf("failed to validate phone uniqueness: %w", err)
		}
		if taken {
			return ErrPhoneAlreadyTaken
		}

		if err := s.userRepo.UpdatePhone(ctx, user.ID, sanitizedPhone); err != nil {
			return fmt.Errorf("failed to update phone: %w", err)
		}
		user.Phone = sanitizedPhone
	}

	user.Phone = strings.TrimSpace(user.Phone)

	code, err := generateOtpCode()
	if err != nil {
		return fmt.Errorf("failed to generate otp: %w", err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash otp: %w", err)
	}

	otp := &models.Otp{
		UserID:       user.ID,
		VerifiableID: security.ID,
		Code:         string(hashed),
	}

	if err := s.accountSecurityRepo.UpsertOtp(ctx, otp); err != nil {
		return fmt.Errorf("failed to persist otp: %w", err)
	}

	if err := s.dispatchAccountSecurityOTP(ctx, user.Phone, code); err != nil {
		return err
	}

	return nil
}

func (s *authService) VerifyAccountSecurity(ctx context.Context, userID uint64, code, ip, userAgent string) error {
	sanitizedCode := strings.TrimSpace(code)
	if !otpCodeRegex.MatchString(sanitizedCode) {
		return ErrInvalidOTPCode
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	security, err := s.accountSecurityRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load account security: %w", err)
	}
	if security == nil {
		return ErrAccountSecurityNotFound
	}
	if security.Unlocked {
		return ErrAccountSecurityAlreadyUnlocked
	}

	otp, err := s.accountSecurityRepo.GetOtpByAccountSecurity(ctx, security.ID)
	if err != nil {
		return fmt.Errorf("failed to load otp: %w", err)
	}
	if otp == nil {
		return ErrAccountSecurityNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.Code), []byte(sanitizedCode)); err != nil {
		return ErrInvalidOTPCode
	}

	if !user.PhoneVerifiedAt.Valid {
		if err := s.userRepo.MarkPhoneAsVerified(ctx, user.ID); err != nil {
			return fmt.Errorf("failed to mark phone as verified: %w", err)
		}
		user.PhoneVerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	expiresAt := time.Now().Unix() + security.Length
	security.Unlocked = true
	security.Until = sql.NullInt64{Int64: expiresAt, Valid: true}
	if err := s.accountSecurityRepo.Update(ctx, security); err != nil {
		return fmt.Errorf("failed to update account security: %w", err)
	}

	if err := s.accountSecurityRepo.DeleteOtp(ctx, otp.ID); err != nil {
		return fmt.Errorf("failed to delete otp: %w", err)
	}

	event := &models.UserEvent{
		UserID: user.ID,
		Event:  "غیر فعال سازی امنیت حساب کاربری",
		IP:     strings.TrimSpace(ip),
		Device: strings.TrimSpace(userAgent),
		Status: 1,
	}
	if err := s.activityRepo.CreateUserEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record account security event: %w", err)
	}

	return nil
}

func (s *authService) dispatchAccountSecurityOTP(ctx context.Context, phone, code string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ErrPhoneRequired
	}

	if s.notificationsClient == nil {
		return fmt.Errorf("notification service client is not configured")
	}

	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.notificationsClient.SendOTP(sendCtx, &notificationspb.SendOTPRequest{
		Phone:  phone,
		Code:   code,
		Reason: "verify",
	})
	if err != nil {
		return fmt.Errorf("failed to dispatch account security otp: %w", err)
	}

	return nil
}

// OAuth helper methods

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type OAuthUserData struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Code     string `json:"code"`
	Referral string `json:"referral"`
}

func (s *authService) exchangeCodeForToken(ctx context.Context, code string) (*OAuthTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", s.oauthClientID)
	data.Set("client_secret", s.oauthClientSecret)
	data.Set("redirect_uri", s.oauthServerURL+"/auth/callback")
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", s.oauthServerURL+"/oauth/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth token request failed: %s", string(body))
	}

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (s *authService) getUserFromOAuth(ctx context.Context, accessToken string) (*OAuthUserData, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.oauthServerURL+"/api/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth user request failed: %s", string(body))
	}

	var userData OAuthUserData
	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		return nil, err
	}

	return &userData, nil
}

// Utility functions

func generateOtpCode() (string, error) {
	max := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	code := n.Int64() + 100000
	return fmt.Sprintf("%06d", code), nil
}

func generateState() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func splitToken(token string) [2]string {
	for i := 0; i < len(token); i++ {
		if token[i] == '|' {
			return [2]string{token[:i], token[i+1:]}
		}
	}
	return [2]string{"", token}
}
