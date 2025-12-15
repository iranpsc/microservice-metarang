package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ProfileLimitationOptions represents the six boolean flags for profile limitations
type ProfileLimitationOptions struct {
	Follow                bool `json:"follow"`
	SendMessage           bool `json:"send_message"`
	Share                 bool `json:"share"`
	SendTicket            bool `json:"send_ticket"`
	ViewProfileImages     bool `json:"view_profile_images"`
	ViewFeaturesLocations bool `json:"view_features_locations"`
}

// DefaultOptions returns options with all flags enabled (default state)
func DefaultOptions() ProfileLimitationOptions {
	return ProfileLimitationOptions{
		Follow:                true,
		SendMessage:           true,
		Share:                 true,
		SendTicket:            true,
		ViewProfileImages:     true,
		ViewFeaturesLocations: true,
	}
}

// ProfileLimitation represents a profile limitation record
type ProfileLimitation struct {
	ID            uint64                   `db:"id"`
	LimiterUserID uint64                   `db:"limiter_user_id"`
	LimitedUserID uint64                   `db:"limited_user_id"`
	Options       ProfileLimitationOptions `db:"options"`
	Note          sql.NullString           `db:"note"`
	CreatedAt     time.Time                `db:"created_at"`
	UpdatedAt     time.Time                `db:"updated_at"`
}

// OptionsJSON is a helper type for JSON marshaling/unmarshaling
type OptionsJSON struct {
	Options ProfileLimitationOptions
}

// MarshalJSON implements json.Marshaler
func (o ProfileLimitationOptions) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]bool{
		"follow":                  o.Follow,
		"send_message":            o.SendMessage,
		"share":                   o.Share,
		"send_ticket":             o.SendTicket,
		"view_profile_images":     o.ViewProfileImages,
		"view_features_locations": o.ViewFeaturesLocations,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (o *ProfileLimitationOptions) UnmarshalJSON(data []byte) error {
	var m map[string]bool
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Set defaults first
	*o = DefaultOptions()

	// Override with provided values
	if val, ok := m["follow"]; ok {
		o.Follow = val
	}
	if val, ok := m["send_message"]; ok {
		o.SendMessage = val
	}
	if val, ok := m["share"]; ok {
		o.Share = val
	}
	if val, ok := m["send_ticket"]; ok {
		o.SendTicket = val
	}
	if val, ok := m["view_profile_images"]; ok {
		o.ViewProfileImages = val
	}
	if val, ok := m["view_features_locations"]; ok {
		o.ViewFeaturesLocations = val
	}

	return nil
}
