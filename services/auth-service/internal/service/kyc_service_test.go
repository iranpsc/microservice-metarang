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

// fakeKYCRepository is a mock implementation of KYCRepository for testing
type fakeKYCRepository struct {
	kycs           map[uint64]*models.KYC
	bankAccounts   map[uint64]*models.BankAccount
	createCount    int
	updateCount    int
	findByUserID   func(ctx context.Context, userID uint64) (*models.KYC, error)
	checkMelliCode func(ctx context.Context, melliCode string, excludeUserID uint64) (bool, error)
}

func newFakeKYCRepository() *fakeKYCRepository {
	return &fakeKYCRepository{
		kycs:         make(map[uint64]*models.KYC),
		bankAccounts: make(map[uint64]*models.BankAccount),
	}
}

func (r *fakeKYCRepository) Create(ctx context.Context, kyc *models.KYC) error {
	r.createCount++
	if kyc.ID == 0 {
		kyc.ID = uint64(len(r.kycs) + 1)
	}
	r.kycs[kyc.UserID] = kyc
	return nil
}

func (r *fakeKYCRepository) FindByUserID(ctx context.Context, userID uint64) (*models.KYC, error) {
	if r.findByUserID != nil {
		return r.findByUserID(ctx, userID)
	}
	return r.kycs[userID], nil
}

func (r *fakeKYCRepository) Update(ctx context.Context, kyc *models.KYC) error {
	r.updateCount++
	r.kycs[kyc.UserID] = kyc
	return nil
}

func (r *fakeKYCRepository) CheckUniqueMelliCode(ctx context.Context, melliCode string, excludeUserID uint64) (bool, error) {
	if r.checkMelliCode != nil {
		return r.checkMelliCode(ctx, melliCode, excludeUserID)
	}
	for _, kyc := range r.kycs {
		if kyc.MelliCode == melliCode && kyc.UserID != excludeUserID {
			return false, nil
		}
	}
	return true, nil
}

func (r *fakeKYCRepository) CreateBankAccount(ctx context.Context, bankAccount *models.BankAccount) error {
	if bankAccount.ID == 0 {
		bankAccount.ID = uint64(len(r.bankAccounts) + 1)
	}
	r.bankAccounts[bankAccount.ID] = bankAccount
	return nil
}

func (r *fakeKYCRepository) FindBankAccountsByUserID(ctx context.Context, userID uint64) ([]*models.BankAccount, error) {
	var accounts []*models.BankAccount
	for _, account := range r.bankAccounts {
		if account.BankableID == userID {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

func (r *fakeKYCRepository) FindBankAccountByID(ctx context.Context, bankAccountID uint64) (*models.BankAccount, error) {
	return r.bankAccounts[bankAccountID], nil
}

func (r *fakeKYCRepository) UpdateBankAccount(ctx context.Context, bankAccount *models.BankAccount) error {
	r.bankAccounts[bankAccount.ID] = bankAccount
	return nil
}

func (r *fakeKYCRepository) DeleteBankAccount(ctx context.Context, bankAccountID uint64) error {
	delete(r.bankAccounts, bankAccountID)
	return nil
}

func (r *fakeKYCRepository) CheckUniqueShaba(ctx context.Context, shabaNum string, excludeID uint64) (bool, error) {
	for _, account := range r.bankAccounts {
		if account.ShabaNum == shabaNum && account.ID != excludeID {
			return false, nil
		}
	}
	return true, nil
}

func (r *fakeKYCRepository) CheckUniqueCard(ctx context.Context, cardNum string, excludeID uint64) (bool, error) {
	for _, account := range r.bankAccounts {
		if account.CardNum == cardNum && account.ID != excludeID {
			return false, nil
		}
	}
	return true, nil
}

// fakeKYCUserRepository is a minimal mock for UserRepository
type fakeKYCUserRepository struct {
	users map[uint64]*models.User
}

func newFakeKYCUserRepository(users map[uint64]*models.User) *fakeKYCUserRepository {
	return &fakeKYCUserRepository{users: users}
}

func (r *fakeKYCUserRepository) FindByID(ctx context.Context, id uint64) (*models.User, error) {
	return r.users[id], nil
}

func (r *fakeKYCUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (r *fakeKYCUserRepository) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	for _, user := range r.users {
		if user.Phone == phone {
			return user, nil
		}
	}
	return nil, nil
}

func (r *fakeKYCUserRepository) Create(ctx context.Context, user *models.User) error {
	return nil
}

func (r *fakeKYCUserRepository) Update(ctx context.Context, user *models.User) error {
	return nil
}

func (r *fakeKYCUserRepository) UpdateLastSeen(ctx context.Context, userID uint64) error {
	return nil
}

func (r *fakeKYCUserRepository) UpdateScore(ctx context.Context, userID uint64, score int32) error {
	return nil
}

func (r *fakeKYCUserRepository) FindByCode(ctx context.Context, code string) (*models.User, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) FindReferrals(ctx context.Context, referrerID uint64) ([]*models.User, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) FindReferrer(ctx context.Context, userID uint64) (*models.User, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) CreateSettings(ctx context.Context, settings *models.Settings) error {
	return nil
}

func (r *fakeKYCUserRepository) GetSettings(ctx context.Context, userID uint64) (*models.Settings, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) GetKYC(ctx context.Context, userID uint64) (*models.KYC, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) GetUnreadNotificationsCount(ctx context.Context, userID uint64) (int32, error) {
	return 0, nil
}

func (r *fakeKYCUserRepository) MarkEmailAsVerified(ctx context.Context, userID uint64) error {
	return nil
}

func (r *fakeKYCUserRepository) UpdatePhone(ctx context.Context, userID uint64, phone string) error {
	return nil
}

func (r *fakeKYCUserRepository) MarkPhoneAsVerified(ctx context.Context, userID uint64) error {
	return nil
}

func (r *fakeKYCUserRepository) IsPhoneTaken(ctx context.Context, phone string, excludeUserID uint64) (bool, error) {
	return false, nil
}

func (r *fakeKYCUserRepository) ListUsers(ctx context.Context, search, orderBy string, page, pageSize int32) ([]*repository.UserWithRelations, int32, error) {
	return nil, 0, nil
}

func (r *fakeKYCUserRepository) GetFollowersCount(ctx context.Context, userID uint64) (int32, error) {
	return 0, nil
}

func (r *fakeKYCUserRepository) GetFollowingCount(ctx context.Context, userID uint64) (int32, error) {
	return 0, nil
}

func (r *fakeKYCUserRepository) GetLatestProfilePhotoURL(ctx context.Context, userID uint64) (string, error) {
	return "", nil
}

func (r *fakeKYCUserRepository) GetAllProfilePhotoURLs(ctx context.Context, userID uint64) ([]string, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) GetUserLatestLevel(ctx context.Context, userID uint64) (*repository.UserLevel, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) GetLevelsBelowScore(ctx context.Context, score int32) ([]*repository.UserLevel, error) {
	return nil, nil
}

func (r *fakeKYCUserRepository) GetNextLevelScore(ctx context.Context, score int32) (int32, error) {
	return 0, nil
}

func (r *fakeKYCUserRepository) GetFeatureCounts(ctx context.Context, userID uint64) (int32, int32, int32, error) {
	return 0, 0, 0, nil
}

func TestGetKYC_NotFound(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	kyc, err := service.GetKYC(ctx, 1)
	if err != nil {
		t.Fatalf("GetKYC returned error: %v", err)
	}
	if kyc != nil {
		t.Errorf("expected nil for non-existent KYC, got %v", kyc)
	}
}

func TestGetKYC_Found(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	existingKYC := &models.KYC{
		ID:        1,
		UserID:    1,
		Fname:     "Ali",
		Lname:     "Karimi",
		MelliCode: "1234567890",
		Status:    0,
		Birthdate: sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	kycRepo.kycs[1] = existingKYC
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	kyc, err := service.GetKYC(ctx, 1)
	if err != nil {
		t.Fatalf("GetKYC returned error: %v", err)
	}
	if kyc == nil {
		t.Fatalf("expected KYC to be found")
	}
	if kyc.Fname != "Ali" {
		t.Errorf("expected Fname 'Ali', got %q", kyc.Fname)
	}
}

func TestUpdateKYC_CreateNew(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	kyc, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789", // Valid Iranian national code format (10 digits)
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err != nil {
		t.Fatalf("UpdateKYC returned error: %v", err)
	}
	if kyc == nil {
		t.Fatalf("expected KYC to be created")
	}
	if kyc.Fname != "Ali" {
		t.Errorf("expected Fname 'Ali', got %q", kyc.Fname)
	}
	if kyc.Status != 0 {
		t.Errorf("expected Status 0 (pending), got %d", kyc.Status)
	}
	if kycRepo.createCount != 1 {
		t.Errorf("expected createCount 1, got %d", kycRepo.createCount)
	}
}

func TestUpdateKYC_UpdateRejected(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	existingKYC := &models.KYC{
		ID:        1,
		UserID:    1,
		Fname:     "Old",
		Lname:     "Name",
		MelliCode: "1234567890",
		Status:    -1, // Rejected
		Errors:    sql.NullString{String: "Some error", Valid: true},
		Birthdate: sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	kycRepo.kycs[1] = existingKYC
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	kyc, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err != nil {
		t.Fatalf("UpdateKYC returned error: %v", err)
	}
	if kyc.Fname != "Ali" {
		t.Errorf("expected Fname 'Ali', got %q", kyc.Fname)
	}
	if kyc.Status != 0 {
		t.Errorf("expected Status 0 (pending), got %d", kyc.Status)
	}
	if kyc.Errors.Valid {
		t.Errorf("expected errors to be cleared, got %v", kyc.Errors)
	}
	if kycRepo.updateCount != 1 {
		t.Errorf("expected updateCount 1, got %d", kycRepo.updateCount)
	}
}

func TestUpdateKYC_RejectPendingUpdate(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	existingKYC := &models.KYC{
		ID:        1,
		UserID:    1,
		Fname:     "Old",
		Lname:     "Name",
		MelliCode: "1234567890",
		Status:    0, // Pending - cannot update
		Birthdate: sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	kycRepo.kycs[1] = existingKYC
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error when updating pending KYC")
	}
	if !errors.Is(err, ErrKYCNotRejected) {
		t.Errorf("expected ErrKYCNotRejected, got %v", err)
	}
}

func TestUpdateKYC_RejectApprovedUpdate(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	existingKYC := &models.KYC{
		ID:        1,
		UserID:    1,
		Fname:     "Old",
		Lname:     "Name",
		MelliCode: "1234567890",
		Status:    1, // Approved - cannot update
		Birthdate: sql.NullTime{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
	kycRepo.kycs[1] = existingKYC
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error when updating approved KYC")
	}
	if !errors.Is(err, ErrKYCNotRejected) {
		t.Errorf("expected ErrKYCNotRejected, got %v", err)
	}
}

func TestUpdateKYC_InvalidFname(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"A", // Too short
		"Karimi",
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error for invalid fname")
	}
	if !errors.Is(err, ErrInvalidFname) {
		t.Errorf("expected ErrInvalidFname, got %v", err)
	}
}

func TestUpdateKYC_InvalidLname(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"K", // Too short
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error for invalid lname")
	}
	if !errors.Is(err, ErrInvalidLname) {
		t.Errorf("expected ErrInvalidLname, got %v", err)
	}
}

func TestUpdateKYC_InvalidGender(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789",
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"invalid", // Invalid gender
	)
	if err == nil {
		t.Fatalf("expected error for invalid gender")
	}
	if !errors.Is(err, ErrInvalidGender) {
		t.Errorf("expected ErrInvalidGender, got %v", err)
	}
}

func TestUpdateKYC_InvalidBirthdate(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789",
		"invalid-date", // Invalid date format
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error for invalid birthdate")
	}
	if !errors.Is(err, ErrInvalidBirthdate) {
		t.Errorf("expected ErrInvalidBirthdate, got %v", err)
	}
}

func TestUpdateKYC_DuplicateMelliCode(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	// Create existing KYC with melli_code
	existingKYC := &models.KYC{
		ID:        1,
		UserID:    2, // Different user
		MelliCode: "1234567890",
		Status:    -1,
	}
	kycRepo.kycs[2] = existingKYC
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	_, err := service.UpdateKYC(
		ctx,
		1,
		"Ali",
		"Karimi",
		"0123456789", // Same melli_code as user 2
		"1403/01/15",
		"Tehran",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"male",
	)
	if err == nil {
		t.Fatalf("expected error for duplicate melli_code")
	}
	if !errors.Is(err, ErrMelliCodeNotUnique) {
		t.Errorf("expected ErrMelliCodeNotUnique, got %v", err)
	}
}

func TestUpdateKYC_ValidGenders(t *testing.T) {
	ctx := context.Background()
	validGenders := []string{"male", "female", "other"}

	for _, gender := range validGenders {
		t.Run(gender, func(t *testing.T) {
			kycRepo := newFakeKYCRepository()
			userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
			service := NewKYCService(kycRepo, userRepo)

			_, err := service.UpdateKYC(
				ctx,
				1,
				"Ali",
				"Karimi",
				"0123456789",
				"1403/01/15",
				"Tehran",
				"/uploads/kyc/melli-card.jpg",
				"tmp/uploads",
				"video.mp4",
				1,
				gender,
			)
			if err != nil {
				t.Errorf("expected no error for valid gender %q, got %v", gender, err)
			}
		})
	}
}

func TestUpdateKYC_TrimsWhitespace(t *testing.T) {
	ctx := context.Background()
	kycRepo := newFakeKYCRepository()
	userRepo := newFakeKYCUserRepository(map[uint64]*models.User{1: {ID: 1}})
	service := NewKYCService(kycRepo, userRepo)

	kyc, err := service.UpdateKYC(
		ctx,
		1,
		"  Ali  ",
		"  Karimi  ",
		"  1234567890  ",
		"1403/01/15",
		"  Tehran  ",
		"/uploads/kyc/melli-card.jpg",
		"tmp/uploads",
		"video.mp4",
		1,
		"  male  ",
	)
	if err != nil {
		t.Fatalf("UpdateKYC returned error: %v", err)
	}
	if kyc.Fname != "Ali" {
		t.Errorf("expected trimmed Fname 'Ali', got %q", kyc.Fname)
	}
	if kyc.Lname != "Karimi" {
		t.Errorf("expected trimmed Lname 'Karimi', got %q", kyc.Lname)
	}
	if kyc.MelliCode != "1234567890" {
		t.Errorf("expected trimmed MelliCode '1234567890', got %q", kyc.MelliCode)
	}
	if kyc.Province != "Tehran" {
		t.Errorf("expected trimmed Province 'Tehran', got %q", kyc.Province)
	}
	if kyc.Gender.String != "male" {
		t.Errorf("expected trimmed Gender 'male', got %q", kyc.Gender.String)
	}
}
