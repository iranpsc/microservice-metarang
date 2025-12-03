package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/repository"
	notificationspb "metargb/shared/pb/notifications"
)

func TestRequestAccountSecurityCreatesAndDispatchesOTP(t *testing.T) {
	ctx := context.Background()

	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "",
			PhoneVerifiedAt: sql.NullTime{Valid: false},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{}

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	if err := svc.RequestAccountSecurity(ctx, 1, 15, " 09123456789 "); err != nil {
		t.Fatalf("RequestAccountSecurity returned error: %v", err)
	}

	security := accountRepo.records[1]
	if security == nil {
		t.Fatalf("expected account security record to be created")
	}
	if security.Unlocked {
		t.Errorf("expected security to remain locked")
	}
	if security.Length != 15*60 {
		t.Errorf("expected length 900, got %d", security.Length)
	}
	if security.Until.Valid {
		t.Errorf("expected until to be cleared")
	}

	otp := accountRepo.otps[security.ID]
	if otp == nil {
		t.Fatalf("expected otp to be stored")
	}

	if smsClient.lastRequest == nil {
		t.Fatalf("expected SMS client to receive request")
	}
	if smsClient.lastRequest.Phone != "09123456789" {
		t.Errorf("expected trimmed phone, got %q", smsClient.lastRequest.Phone)
	}
	if smsClient.lastRequest.Reason != "verify" {
		t.Errorf("expected reason 'verify', got %q", smsClient.lastRequest.Reason)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.Code), []byte(smsClient.lastRequest.Code)); err != nil {
		t.Errorf("stored otp hash does not match dispatched code: %v", err)
	}

	updatedUser := users[1]
	if updatedUser.Phone != "09123456789" {
		t.Errorf("expected user phone updated, got %q", updatedUser.Phone)
	}
	if updatedUser.PhoneVerifiedAt.Valid {
		t.Errorf("phone should remain unverified until verification step")
	}

	if accountRepo.createCount != 1 {
		t.Errorf("expected create count 1, got %d", accountRepo.createCount)
	}
	if accountRepo.updateCount != 0 {
		t.Errorf("expected update count 0 for new record, got %d", accountRepo.updateCount)
	}
}

func TestRequestAccountSecurityUpdatesExistingRecord(t *testing.T) {
	ctx := context.Background()

	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "09101234567",
			PhoneVerifiedAt: sql.NullTime{Valid: true},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{}

	existing := &models.AccountSecurity{
		ID:       42,
		UserID:   1,
		Unlocked: true,
		Until:    sql.NullInt64{Int64: time.Now().Unix() + 300, Valid: true},
		Length:   300,
	}
	accountRepo.records[1] = existing

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	if err := svc.RequestAccountSecurity(ctx, 1, 20, ""); err != nil {
		t.Fatalf("RequestAccountSecurity returned error: %v", err)
	}

	security := accountRepo.records[1]
	if security.Unlocked {
		t.Errorf("expected security to be reset to locked")
	}
	if security.Until.Valid {
		t.Errorf("expected until to be cleared")
	}
	if security.Length != 20*60 {
		t.Errorf("expected updated length, got %d", security.Length)
	}

	if accountRepo.createCount != 0 {
		t.Errorf("expected no new create, got %d", accountRepo.createCount)
	}
	if accountRepo.updateCount != 1 {
		t.Errorf("expected single update, got %d", accountRepo.updateCount)
	}
}

func TestRequestAccountSecurityValidations(t *testing.T) {
	ctx := context.Background()
	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "",
			PhoneVerifiedAt: sql.NullTime{Valid: false},
		},
		2: {
			ID:              2,
			Phone:           "09123456789",
			PhoneVerifiedAt: sql.NullTime{Valid: true},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{}

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	t.Run("invalid duration", func(t *testing.T) {
		err := svc.RequestAccountSecurity(ctx, 1, 3, "09111111111")
		if !errors.Is(err, ErrInvalidUnlockDuration) {
			t.Fatalf("expected ErrInvalidUnlockDuration, got %v", err)
		}
	})

	t.Run("missing phone", func(t *testing.T) {
		err := svc.RequestAccountSecurity(ctx, 1, 10, "")
		if !errors.Is(err, ErrPhoneRequired) {
			t.Fatalf("expected ErrPhoneRequired, got %v", err)
		}
	})

	t.Run("duplicate phone", func(t *testing.T) {
		err := svc.RequestAccountSecurity(ctx, 1, 10, "09123456789")
		if !errors.Is(err, ErrPhoneAlreadyTaken) {
			t.Fatalf("expected ErrPhoneAlreadyTaken, got %v", err)
		}
	})
}

func TestRequestAccountSecurityNotificationError(t *testing.T) {
	ctx := context.Background()
	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "",
			PhoneVerifiedAt: sql.NullTime{Valid: false},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{err: errors.New("dispatch failure")}

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	err := svc.RequestAccountSecurity(ctx, 1, 15, "09122223333")
	if err == nil || err.Error() == "" {
		t.Fatalf("expected wrapped notification error, got %v", err)
	}
}

func TestVerifyAccountSecuritySuccess(t *testing.T) {
	ctx := context.Background()

	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "09100000000",
			PhoneVerifiedAt: sql.NullTime{Valid: false},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{}

	security := &models.AccountSecurity{
		ID:       10,
		UserID:   1,
		Unlocked: false,
		Length:   600,
	}
	accountRepo.records[1] = security

	plainCode := "654321"
	hashed, err := bcrypt.GenerateFromPassword([]byte(plainCode), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash otp: %v", err)
	}
	accountRepo.otps[security.ID] = &models.Otp{
		ID:           99,
		UserID:       1,
		VerifiableID: security.ID,
		Code:         string(hashed),
	}

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	err = svc.VerifyAccountSecurity(ctx, 1, plainCode, " 127.0.0.1 ", " Mozilla/5.0 ")
	if err != nil {
		t.Fatalf("VerifyAccountSecurity returned error: %v", err)
	}

	updatedSecurity := accountRepo.records[1]
	if !updatedSecurity.Unlocked {
		t.Fatalf("expected account security unlocked")
	}
	if !updatedSecurity.Until.Valid {
		t.Fatalf("expected unlock window to be set")
	}
	if updatedSecurity.Until.Int64 < time.Now().Unix() {
		t.Fatalf("expected unlock window in the future, got %d", updatedSecurity.Until.Int64)
	}

	if _, found := accountRepo.otps[security.ID]; found {
		t.Fatalf("expected otp to be deleted after verification")
	}

	updatedUser := users[1]
	if !updatedUser.PhoneVerifiedAt.Valid {
		t.Fatalf("expected phone to be marked verified")
	}

	if len(activityRepo.events) != 1 {
		t.Fatalf("expected one user event, got %d", len(activityRepo.events))
	}
	event := activityRepo.events[0]
	if event.Event != "غیر فعال سازی امنیت حساب کاربری" {
		t.Fatalf("unexpected event message: %q", event.Event)
	}
	if event.IP != "127.0.0.1" {
		t.Fatalf("expected trimmed IP, got %q", event.IP)
	}
	if event.Device != "Mozilla/5.0" {
		t.Fatalf("expected trimmed user agent, got %q", event.Device)
	}
}

func TestVerifyAccountSecurityFailures(t *testing.T) {
	ctx := context.Background()

	users := map[uint64]*models.User{
		1: {
			ID:              1,
			Phone:           "09100000000",
			PhoneVerifiedAt: sql.NullTime{Valid: true},
		},
	}

	userRepo := newFakeUserRepository(users)
	accountRepo := newFakeAccountSecurityRepository()
	activityRepo := newFakeActivityRepository()
	smsClient := &fakeSMSServiceClient{}

	security := &models.AccountSecurity{
		ID:       5,
		UserID:   1,
		Unlocked: false,
		Length:   300,
	}
	accountRepo.records[1] = security

	hashed, err := bcrypt.GenerateFromPassword([]byte("111111"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash otp: %v", err)
	}
	accountRepo.otps[security.ID] = &models.Otp{ID: 7, UserID: 1, VerifiableID: security.ID, Code: string(hashed)}

	svc := NewAuthService(userRepo, nil, accountRepo, activityRepo, nil, nil, smsClient, "", "", "", "", "")

	t.Run("invalid code", func(t *testing.T) {
		err := svc.VerifyAccountSecurity(ctx, 1, "000000", "", "")
		if !errors.Is(err, ErrInvalidOTPCode) {
			t.Fatalf("expected ErrInvalidOTPCode, got %v", err)
		}
	})

	accountRepo.records = map[uint64]*models.AccountSecurity{}

	t.Run("missing security record", func(t *testing.T) {
		err := svc.VerifyAccountSecurity(ctx, 1, "111111", "", "")
		if !errors.Is(err, ErrAccountSecurityNotFound) {
			t.Fatalf("expected ErrAccountSecurityNotFound, got %v", err)
		}
	})
}

// --- test fakes ---

type fakeUserRepository struct {
	users map[uint64]*models.User
}

func newFakeUserRepository(users map[uint64]*models.User) *fakeUserRepository {
	return &fakeUserRepository{users: users}
}

func (f *fakeUserRepository) Create(context.Context, *models.User) error {
	panic("unexpected call to Create")
}

func (f *fakeUserRepository) FindByEmail(context.Context, string) (*models.User, error) {
	panic("unexpected call to FindByEmail")
}

func (f *fakeUserRepository) FindByID(_ context.Context, id uint64) (*models.User, error) {
	if user, ok := f.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (f *fakeUserRepository) Update(context.Context, *models.User) error {
	panic("unexpected call to Update")
}

func (f *fakeUserRepository) UpdateLastSeen(context.Context, uint64) error {
	panic("unexpected call to UpdateLastSeen")
}

func (f *fakeUserRepository) FindByCode(context.Context, string) (*models.User, error) {
	panic("unexpected call to FindByCode")
}

func (f *fakeUserRepository) GetSettings(context.Context, uint64) (*models.Settings, error) {
	panic("unexpected call to GetSettings")
}

func (f *fakeUserRepository) CreateSettings(context.Context, *models.Settings) error {
	panic("unexpected call to CreateSettings")
}

func (f *fakeUserRepository) GetKYC(context.Context, uint64) (*models.KYC, error) {
	panic("unexpected call to GetKYC")
}

func (f *fakeUserRepository) GetUnreadNotificationsCount(context.Context, uint64) (int32, error) {
	panic("unexpected call to GetUnreadNotificationsCount")
}

func (f *fakeUserRepository) MarkEmailAsVerified(context.Context, uint64) error {
	panic("unexpected call to MarkEmailAsVerified")
}

func (f *fakeUserRepository) UpdatePhone(_ context.Context, userID uint64, phone string) error {
	if user, ok := f.users[userID]; ok {
		user.Phone = phone
		return nil
	}
	return fmt.Errorf("user %d not found", userID)
}

func (f *fakeUserRepository) MarkPhoneAsVerified(_ context.Context, userID uint64) error {
	if user, ok := f.users[userID]; ok {
		user.PhoneVerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
		return nil
	}
	return fmt.Errorf("user %d not found", userID)
}

func (f *fakeUserRepository) IsPhoneTaken(_ context.Context, phone string, excludeUserID uint64) (bool, error) {
	for id, user := range f.users {
		if id == excludeUserID {
			continue
		}
		if user.Phone == phone {
			return true, nil
		}
	}
	return false, nil
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)

type fakeAccountSecurityRepository struct {
	nextID      uint64
	nextOtpID   uint64
	records     map[uint64]*models.AccountSecurity
	otps        map[uint64]*models.Otp
	createCount int
	updateCount int
}

func newFakeAccountSecurityRepository() *fakeAccountSecurityRepository {
	return &fakeAccountSecurityRepository{
		nextID:    100,
		nextOtpID: 200,
		records:   make(map[uint64]*models.AccountSecurity),
		otps:      make(map[uint64]*models.Otp),
	}
}

func (f *fakeAccountSecurityRepository) GetByUserID(_ context.Context, userID uint64) (*models.AccountSecurity, error) {
	if security, ok := f.records[userID]; ok {
		return security, nil
	}
	return nil, nil
}

func (f *fakeAccountSecurityRepository) Create(_ context.Context, security *models.AccountSecurity) error {
	f.createCount++
	if security.ID == 0 {
		security.ID = f.nextID
		f.nextID++
	}
	now := time.Now()
	security.CreatedAt = now
	security.UpdatedAt = now
	f.records[security.UserID] = security
	return nil
}

func (f *fakeAccountSecurityRepository) Update(_ context.Context, security *models.AccountSecurity) error {
	f.updateCount++
	security.UpdatedAt = time.Now()
	f.records[security.UserID] = security
	return nil
}

func (f *fakeAccountSecurityRepository) GetOtpByAccountSecurity(_ context.Context, accountSecurityID uint64) (*models.Otp, error) {
	if otp, ok := f.otps[accountSecurityID]; ok {
		return otp, nil
	}
	return nil, nil
}

func (f *fakeAccountSecurityRepository) UpsertOtp(_ context.Context, otp *models.Otp) error {
	if otp.ID == 0 {
		otp.ID = f.nextOtpID
		f.nextOtpID++
	}
	now := time.Now()
	otp.CreatedAt = now
	otp.UpdatedAt = now
	otp.VerifiableType = "App\\Models\\AccountSecurity"
	f.otps[otp.VerifiableID] = otp
	return nil
}

func (f *fakeAccountSecurityRepository) DeleteOtp(_ context.Context, otpID uint64) error {
	for key, otp := range f.otps {
		if otp.ID == otpID {
			delete(f.otps, key)
			return nil
		}
	}
	return nil
}

var _ repository.AccountSecurityRepository = (*fakeAccountSecurityRepository)(nil)

type fakeActivityRepository struct {
	events []*models.UserEvent
}

func newFakeActivityRepository() *fakeActivityRepository {
	return &fakeActivityRepository{}
}

func (f *fakeActivityRepository) CreateUserEvent(_ context.Context, event *models.UserEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeActivityRepository) CreateActivity(context.Context, *models.UserActivity) error {
	panic("unexpected call to CreateActivity")
}

func (f *fakeActivityRepository) GetLatestActivity(context.Context, uint64) (*models.UserActivity, error) {
	panic("unexpected call to GetLatestActivity")
}

func (f *fakeActivityRepository) UpdateActivity(context.Context, *models.UserActivity) error {
	panic("unexpected call to UpdateActivity")
}

func (f *fakeActivityRepository) GetTotalActivityMinutes(context.Context, uint64) (int32, error) {
	panic("unexpected call to GetTotalActivityMinutes")
}

func (f *fakeActivityRepository) GetUserLog(context.Context, uint64) (*models.UserLog, error) {
	panic("unexpected call to GetUserLog")
}

func (f *fakeActivityRepository) CreateUserLog(context.Context, *models.UserLog) error {
	panic("unexpected call to CreateUserLog")
}

func (f *fakeActivityRepository) UpdateUserLog(context.Context, *models.UserLog) error {
	panic("unexpected call to UpdateUserLog")
}

func (f *fakeActivityRepository) IncrementLogField(context.Context, uint64, string, float64) error {
	panic("unexpected call to IncrementLogField")
}

var _ repository.ActivityRepository = (*fakeActivityRepository)(nil)

type fakeSMSServiceClient struct {
	lastRequest *notificationspb.SendOTPRequest
	err         error
}

func (f *fakeSMSServiceClient) SendSMS(context.Context, *notificationspb.SendSMSRequest, ...grpc.CallOption) (*notificationspb.SMSResponse, error) {
	panic("unexpected call to SendSMS")
}

func (f *fakeSMSServiceClient) SendOTP(_ context.Context, req *notificationspb.SendOTPRequest, _ ...grpc.CallOption) (*notificationspb.SMSResponse, error) {
	f.lastRequest = req
	if f.err != nil {
		return nil, f.err
	}
	return &notificationspb.SMSResponse{Sent: true}, nil
}

var _ notificationspb.SMSServiceClient = (*fakeSMSServiceClient)(nil)
