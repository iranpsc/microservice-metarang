package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID           uint64          `db:"id"`
	UserID       uint64          `db:"user_id"`
	PSC          decimal.Decimal `db:"psc"`
	IRR          decimal.Decimal `db:"irr"`
	Red          decimal.Decimal `db:"red"`
	Blue         decimal.Decimal `db:"blue"`
	Yellow       decimal.Decimal `db:"yellow"`
	Satisfaction decimal.Decimal `db:"satisfaction"`
	Effect       decimal.Decimal `db:"effect"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

type Transaction struct {
	ID          string    `db:"id"` // VARCHAR PK like TR-xxxxx
	UserID      uint64    `db:"user_id"`
	Asset       string    `db:"asset"`
	Amount      float64   `db:"amount"`
	Action      string    `db:"action"` // deposit, withdraw
	Status      int32     `db:"status"`
	Token       *int64    `db:"token"`
	RefID       *int64    `db:"ref_id"`
	PayableType *string   `db:"payable_type"`
	PayableID   *uint64   `db:"payable_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Order struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	Asset     string    `db:"asset"`
	Amount    float64   `db:"amount"`
	Status    int32     `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Payment struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	RefID     int64     `db:"ref_id"`
	CardPan   string    `db:"card_pan"`
	Gateway   string    `db:"gateway"`
	Amount    float64   `db:"amount"`
	Product   string    `db:"product"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Variable struct {
	ID    uint64  `db:"id"`
	Key   string  `db:"key"`
	Value float64 `db:"value"`
}

type FirstOrder struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	Type      string    `db:"type"`
	Amount    float64   `db:"amount"`
	Date      string    `db:"date"` // Jalali date format Y/m/d
	Bonus     float64   `db:"bonus"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ReferralOrderHistory struct {
	ID         uint64    `db:"id"`
	UserID     uint64    `db:"user_id"`     // The referrer who receives the commission
	ReferralID uint64    `db:"referral_id"` // The user who was referred
	Amount     float64   `db:"amount"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type LockedAsset struct {
	ID        uint64          `db:"id"`
	UserID    uint64          `db:"user_id"`
	Asset     string          `db:"asset"`
	Amount    decimal.Decimal `db:"amount"`
	Reason    string          `db:"reason"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt time.Time       `db:"updated_at"`
}
