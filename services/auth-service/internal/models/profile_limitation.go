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
	// Handle empty or null JSON
	if len(data) == 0 || string(data) == "null" {
		*o = DefaultOptions()
		return nil
	}

	// First, try to unmarshal as a string (in case JSON is double-encoded)
	var jsonStr string
	if err := json.Unmarshal(data, &jsonStr); err == nil {
		// It's a JSON string, unmarshal the inner JSON
		data = []byte(jsonStr)
	}

	// Use json.RawMessage to get raw JSON, then parse manually
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Set defaults first
	*o = DefaultOptions()

	// Helper function to parse json.RawMessage to bool
	parseBoolFromRaw := func(raw json.RawMessage) bool {
		// Try to unmarshal as bool first
		var b bool
		if err := json.Unmarshal(raw, &b); err == nil {
			return b
		}

		// Try to unmarshal as string
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s == "true" || s == "1" || s == "True" || s == "TRUE"
		}

		// Try to unmarshal as number
		var n float64
		if err := json.Unmarshal(raw, &n); err == nil {
			return n != 0
		}

		return false
	}

	// Override with provided values
	if val, ok := raw["follow"]; ok {
		o.Follow = parseBoolFromRaw(val)
	}
	if val, ok := raw["send_message"]; ok {
		o.SendMessage = parseBoolFromRaw(val)
	}
	if val, ok := raw["share"]; ok {
		o.Share = parseBoolFromRaw(val)
	}
	if val, ok := raw["send_ticket"]; ok {
		o.SendTicket = parseBoolFromRaw(val)
	}
	if val, ok := raw["view_profile_images"]; ok {
		o.ViewProfileImages = parseBoolFromRaw(val)
	}
	if val, ok := raw["view_features_locations"]; ok {
		o.ViewFeaturesLocations = parseBoolFromRaw(val)
	}

	return nil
}
