package models_test

import (
	"encoding/json"
	"testing"

	"metarang/auth-service/internal/models"
)

func TestProfileLimitationOptions_UnmarshalJSON(t *testing.T) {
	t.Run("null uses defaults", func(t *testing.T) {
		var o models.ProfileLimitationOptions
		if err := json.Unmarshal([]byte("null"), &o); err != nil {
			t.Fatal(err)
		}
		def := models.DefaultOptions()
		if o != def {
			t.Fatalf("got %+v want %+v", o, def)
		}
	})

	t.Run("empty bytes via UnmarshalJSON", func(t *testing.T) {
		var o models.ProfileLimitationOptions
		if err := o.UnmarshalJSON(nil); err != nil {
			t.Fatal(err)
		}
		if o != models.DefaultOptions() {
			t.Fatalf("got %+v", o)
		}
	})

	t.Run("double-encoded string", func(t *testing.T) {
		inner := `{"follow":false,"send_message":false}`
		raw, err := json.Marshal(inner)
		if err != nil {
			t.Fatal(err)
		}
		var o models.ProfileLimitationOptions
		if err := json.Unmarshal(raw, &o); err != nil {
			t.Fatal(err)
		}
		if o.Follow || o.SendMessage {
			t.Fatalf("expected false flags, got %+v", o)
		}
		if !o.Share {
			t.Fatal("unset flags should keep defaults (true)")
		}
	})

	t.Run("string and number bools", func(t *testing.T) {
		raw := []byte(`{
			"follow": "true",
			"send_message": "1",
			"share": "True",
			"send_ticket": "TRUE",
			"view_profile_images": 1,
			"view_features_locations": 0
		}`)
		var o models.ProfileLimitationOptions
		if err := json.Unmarshal(raw, &o); err != nil {
			t.Fatal(err)
		}
		if !o.Follow || !o.SendMessage || !o.Share || !o.SendTicket || !o.ViewProfileImages {
			t.Fatalf("expected truthy parses, got %+v", o)
		}
		if o.ViewFeaturesLocations {
			t.Fatal("expected 0 -> false")
		}
	})

	t.Run("unknown value type becomes false", func(t *testing.T) {
		raw := []byte(`{"follow":{"x":1},"send_message":"no"}`)
		var o models.ProfileLimitationOptions
		if err := json.Unmarshal(raw, &o); err != nil {
			t.Fatal(err)
		}
		if o.Follow || o.SendMessage {
			t.Fatalf("expected false, got %+v", o)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		var o models.ProfileLimitationOptions
		if err := json.Unmarshal([]byte(`"not-json-object"`), &o); err == nil {
			t.Fatal("expected error")
		}
	})
}
