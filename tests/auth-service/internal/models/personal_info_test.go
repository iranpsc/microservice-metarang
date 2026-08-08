package models_test

import (
	"database/sql"
	"testing"

	"metarang/auth-service/internal/models"
)

func TestPersonalInfo_HasData(t *testing.T) {
	ns := func(s string) sql.NullString {
		return sql.NullString{String: s, Valid: true}
	}
	emptyValid := sql.NullString{String: "", Valid: true}

	t.Run("nil receiver", func(t *testing.T) {
		var p *models.PersonalInfo
		if p.HasData() {
			t.Fatal("expected false for nil")
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		p := &models.PersonalInfo{}
		if p.HasData() {
			t.Fatal("expected false for empty")
		}
	})

	t.Run("valid empty strings are ignored", func(t *testing.T) {
		p := &models.PersonalInfo{
			Occupation:     emptyValid,
			Education:      emptyValid,
			Memory:         emptyValid,
			LovedCity:      emptyValid,
			LovedCountry:   emptyValid,
			LovedLanguage:  emptyValid,
			ProblemSolving: emptyValid,
			Prediction:     emptyValid,
			About:          emptyValid,
			Passions:       map[string]bool{"music": false, "art": false},
		}
		if p.HasData() {
			t.Fatal("expected false when all values empty/false")
		}
	})

	t.Run("nil passions", func(t *testing.T) {
		p := &models.PersonalInfo{Passions: nil}
		if p.HasData() {
			t.Fatal("expected false")
		}
	})

	fields := []struct {
		name string
		set  func(*models.PersonalInfo)
	}{
		{"occupation", func(p *models.PersonalInfo) { p.Occupation = ns("eng") }},
		{"education", func(p *models.PersonalInfo) { p.Education = ns("bs") }},
		{"memory", func(p *models.PersonalInfo) { p.Memory = ns("m") }},
		{"loved_city", func(p *models.PersonalInfo) { p.LovedCity = ns("tehran") }},
		{"loved_country", func(p *models.PersonalInfo) { p.LovedCountry = ns("ir") }},
		{"loved_language", func(p *models.PersonalInfo) { p.LovedLanguage = ns("fa") }},
		{"problem_solving", func(p *models.PersonalInfo) { p.ProblemSolving = ns("ps") }},
		{"prediction", func(p *models.PersonalInfo) { p.Prediction = ns("pr") }},
		{"about", func(p *models.PersonalInfo) { p.About = ns("about") }},
		{"passion true", func(p *models.PersonalInfo) { p.Passions = map[string]bool{"music": true} }},
	}

	for _, tc := range fields {
		t.Run(tc.name, func(t *testing.T) {
			p := &models.PersonalInfo{}
			tc.set(p)
			if !p.HasData() {
				t.Fatalf("expected true for %s", tc.name)
			}
		})
	}
}
