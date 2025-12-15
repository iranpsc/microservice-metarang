package models

import "time"

// ReceivedPrize represents a prize awarded to a user
type ReceivedPrize struct {
	ID        uint64        `db:"id"`
	UserID    uint64        `db:"user_id"`
	PrizeID   uint64        `db:"prize_id"`
	Message   string        `db:"message"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
	Prize     *DynastyPrize `db:"-"` // Not from DB, joined
}
