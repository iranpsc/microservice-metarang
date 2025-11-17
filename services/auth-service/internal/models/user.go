package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID              uint64         `db:"id"`
	Name            string         `db:"name"`
	Email           string         `db:"email"`
	Phone           string         `db:"phone"`
	Password        string         `db:"password"`
	Code            string         `db:"code"`
	ReferrerID      sql.NullInt64  `db:"referrer_id"`
	Score           int32          `db:"score"`
	IP              string         `db:"ip"`
	LastSeen        sql.NullTime   `db:"last_seen"`
	EmailVerifiedAt sql.NullTime   `db:"email_verified_at"`
	PhoneVerifiedAt sql.NullTime   `db:"phone_verified_at"`
	AccessToken     sql.NullString `db:"access_token"`
	RefreshToken    sql.NullString `db:"refresh_token"`
	TokenType       sql.NullString `db:"token_type"`
	ExpiresIn       sql.NullInt64  `db:"expires_in"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

type PersonalAccessToken struct {
	ID            uint64       `db:"id"`
	TokenableType string       `db:"tokenable_type"`
	TokenableID   uint64       `db:"tokenable_id"`
	Name          string       `db:"name"`
	Token         string       `db:"token"`
	Abilities     string       `db:"abilities"`
	LastUsedAt    sql.NullTime `db:"last_used_at"`
	ExpiresAt     sql.NullTime `db:"expires_at"`
	CreatedAt     time.Time    `db:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at"`
}

type KYC struct {
	ID           uint64       `db:"id"`
	UserID       uint64       `db:"user_id"`
	Fname        string       `db:"fname"`
	Lname        string       `db:"lname"`
	NationalCode string       `db:"national_code"`
	Status       int32        `db:"status"`
	Birthdate    sql.NullTime `db:"birthdate"`
	CreatedAt    time.Time    `db:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"`
}

func (k *KYC) FullName() string {
	return k.Fname + " " + k.Lname
}

type Settings struct {
	ID              uint64 `db:"id"`
	UserID          uint64 `db:"user_id"`
	AutomaticLogout int32  `db:"automatic_logout"`
}

type AccountSecurity struct {
	ID           uint64        `db:"id"`
	UserID       uint64        `db:"user_id"`
	Unlocked     bool          `db:"unlocked"`
	Until        sql.NullInt64 `db:"until"`
	Length       int64         `db:"length"`
	LastActivity sql.NullInt64 `db:"last_activity"`
	CreatedAt    time.Time     `db:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at"`
}

type Otp struct {
	ID             uint64    `db:"id"`
	UserID         uint64    `db:"user_id"`
	VerifiableType string    `db:"verifiable_type"`
	VerifiableID   uint64    `db:"verifiable_id"`
	Code           string    `db:"code"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type BankAccount struct {
	ID           uint64         `db:"id"`
	BankableType string         `db:"bankable_type"`
	BankableID   uint64         `db:"bankable_id"`
	BankName     string         `db:"bank_name"`
	ShabaNum     string         `db:"shaba_num"`
	CardNum      string         `db:"card_num"`
	Status       int32          `db:"status"`
	Errors       sql.NullString `db:"errors"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

// UserEvent represents login/logout and other user events
type UserEvent struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	Event     string    `db:"event"`
	IP        string    `db:"ip"`
	Device    string    `db:"device"`
	Status    int32     `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// UserActivity represents user session activity tracking
type UserActivity struct {
	ID        uint64       `db:"id"`
	UserID    uint64       `db:"user_id"`
	Start     time.Time    `db:"start"`
	End       sql.NullTime `db:"end"`
	Total     int32        `db:"total"` // Total minutes
	IP        string       `db:"ip"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

// UserLog represents user scoring and activity statistics
type UserLog struct {
	ID                uint64    `db:"id"`
	UserID            uint64    `db:"user_id"`
	TransactionsCount float64   `db:"transactions_count"`
	FollowersCount    float64   `db:"followers_count"`
	DepositAmount     float64   `db:"deposit_amount"`
	ActivityHours     float64   `db:"activity_hours"`
	Score             float64   `db:"score"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// UserVariables represents per-user settings and limits
type UserVariables struct {
	ID             uint64    `db:"id"`
	UserID         uint64    `db:"user_id"`
	WithdrawProfit int32     `db:"withdraw_profit"` // Days
	ReferralProfit float64   `db:"referral_profit"` // Limit amount
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
