package models

import (
	"database/sql"
)

// PersonalInfo represents a user's personal information
type PersonalInfo struct {
	ID             uint64
	UserID         uint64
	Occupation     sql.NullString
	Education      sql.NullString
	Memory         sql.NullString
	LovedCity      sql.NullString
	LovedCountry   sql.NullString
	LovedLanguage  sql.NullString
	ProblemSolving sql.NullString
	Prediction     sql.NullString
	About          sql.NullString
	Passions       map[string]bool // JSON map of passion keys to boolean values
}

// HasData returns true if the personal info has any data (not all fields are null/empty)
func (p *PersonalInfo) HasData() bool {
	if p == nil {
		return false
	}

	// Check if any string field has a value
	if p.Occupation.Valid && p.Occupation.String != "" {
		return true
	}
	if p.Education.Valid && p.Education.String != "" {
		return true
	}
	if p.Memory.Valid && p.Memory.String != "" {
		return true
	}
	if p.LovedCity.Valid && p.LovedCity.String != "" {
		return true
	}
	if p.LovedCountry.Valid && p.LovedCountry.String != "" {
		return true
	}
	if p.LovedLanguage.Valid && p.LovedLanguage.String != "" {
		return true
	}
	if p.ProblemSolving.Valid && p.ProblemSolving.String != "" {
		return true
	}
	if p.Prediction.Valid && p.Prediction.String != "" {
		return true
	}
	if p.About.Valid && p.About.String != "" {
		return true
	}

	// Check if any passion is true
	if p.Passions != nil {
		for _, value := range p.Passions {
			if value {
				return true
			}
		}
	}

	return false
}

// DefaultPassions returns the default passions map with all values set to false
func DefaultPassions() map[string]bool {
	return map[string]bool{
		"music":              false,
		"sport_health":       false,
		"art":                false,
		"language_culture":   false,
		"philosophy":         false,
		"animals_nature":     false,
		"aliens":             false,
		"food_cooking":       false,
		"travel_leature":     false,
		"manufacturing":      false,
		"science_technology": false,
		"space_time":         false,
		"history":            false,
		"politics_economy":   false,
	}
}
