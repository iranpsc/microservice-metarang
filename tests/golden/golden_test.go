package golden

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenJSON compares microservice responses with Laravel golden responses
func TestGoldenJSON(t *testing.T) {
	testCases := []struct {
		name       string
		goldenFile string
		skipFields []string // Fields to skip comparison (e.g., timestamps, UUIDs)
	}{
		{
			name:       "AuthGetMe",
			goldenFile: "auth_get_me.json",
			skipFields: []string{"last_seen", "updated_at"},
		},
		{
			name:       "UserWallet",
			goldenFile: "user_wallet.json",
			skipFields: []string{"updated_at"},
		},
		{
			name:       "FeaturesList",
			goldenFile: "features_list.json",
			skipFields: []string{"created_at", "updated_at"},
		},
		{
			name:       "FeatureDetail",
			goldenFile: "feature_detail.json",
			skipFields: []string{"created_at", "updated_at"},
		},
		{
			name:       "TransactionsList",
			goldenFile: "transactions_list.json",
			skipFields: []string{"created_at"},
		},
		{
			name:       "UserLevel",
			goldenFile: "user_level.json",
			skipFields: []string{"updated_at"},
		},
		{
			name:       "Notifications",
			goldenFile: "notifications_list.json",
			skipFields: []string{"created_at", "read_at"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load golden file
			goldenPath := filepath.Join("testdata", tc.goldenFile)
			goldenData, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "Failed to read golden file: %s", goldenPath)

			// In a real scenario, this would call the microservice endpoint
			// For now, we'll load the actual response from a separate file
			actualPath := filepath.Join("testdata", "actual", tc.goldenFile)
			actualData, err := os.ReadFile(actualPath)
			if os.IsNotExist(err) {
				t.Skip("Actual response file not found, run services first")
				return
			}
			require.NoError(t, err, "Failed to read actual response file")

			// Compare JSON structures
			err = compareJSON(goldenData, actualData, tc.skipFields)
			if err != nil {
				// Save diff for debugging
				diffPath := filepath.Join("diffs", tc.goldenFile+".diff")
				os.MkdirAll(filepath.Dir(diffPath), 0755)
				os.WriteFile(diffPath, []byte(err.Error()), 0644)
				
				t.Errorf("JSON mismatch: %v", err)
			}
		})
	}
}

// compareJSON compares two JSON byte arrays, ignoring specified fields
func compareJSON(golden, actual []byte, skipFields []string) error {
	var goldenObj, actualObj interface{}

	if err := json.Unmarshal(golden, &goldenObj); err != nil {
		return fmt.Errorf("failed to unmarshal golden JSON: %w", err)
	}

	if err := json.Unmarshal(actual, &actualObj); err != nil {
		return fmt.Errorf("failed to unmarshal actual JSON: %w", err)
	}

	// Remove skip fields
	if len(skipFields) > 0 {
		goldenObj = removeFields(goldenObj, skipFields)
		actualObj = removeFields(actualObj, skipFields)
	}

	// Normalize JSON (re-marshal and unmarshal to ensure consistent ordering)
	goldenNorm, _ := json.Marshal(goldenObj)
	actualNorm, _ := json.Marshal(actualObj)

	// Compare byte-by-byte
	if !bytes.Equal(goldenNorm, actualNorm) {
		// Pretty print for better error messages
		goldenPretty, _ := json.MarshalIndent(goldenObj, "", "  ")
		actualPretty, _ := json.MarshalIndent(actualObj, "", "  ")

		return fmt.Errorf("JSON mismatch:\n\nExpected (golden):\n%s\n\nActual:\n%s\n\nDiff:\n%s",
			string(goldenPretty), string(actualPretty), diff(goldenPretty, actualPretty))
	}

	return nil
}

// removeFields recursively removes specified fields from a JSON structure
func removeFields(obj interface{}, fields []string) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		for _, field := range fields {
			delete(v, field)
		}
		for key, val := range v {
			v[key] = removeFields(val, fields)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = removeFields(val, fields)
		}
		return v
	default:
		return v
	}
}

// diff generates a simple diff between two JSON strings
func diff(a, b []byte) string {
	linesA := strings.Split(string(a), "\n")
	linesB := strings.Split(string(b), "\n")

	var result strings.Builder
	maxLen := len(linesA)
	if len(linesB) > maxLen {
		maxLen = len(linesB)
	}

	for i := 0; i < maxLen; i++ {
		lineA := ""
		lineB := ""
		if i < len(linesA) {
			lineA = linesA[i]
		}
		if i < len(linesB) {
			lineB = linesB[i]
		}

		if lineA != lineB {
			if lineA != "" {
				result.WriteString(fmt.Sprintf("- %s\n", lineA))
			}
			if lineB != "" {
				result.WriteString(fmt.Sprintf("+ %s\n", lineB))
			}
		}
	}

	return result.String()
}

// TestFieldTypes validates that specific fields have correct data types
func TestFieldTypes(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		fieldPath  string
		expectType string // "string", "number", "boolean", "array", "object"
	}{
		{
			name:       "WalletPSCIsString",
			file:       "user_wallet.json",
			fieldPath:  "data.psc",
			expectType: "string",
		},
		{
			name:       "WalletRGBIsString",
			file:       "user_wallet.json",
			fieldPath:  "data.rgb",
			expectType: "string",
		},
		{
			name:       "FeaturePriceIsString",
			file:       "feature_detail.json",
			fieldPath:  "data.property.price_psc",
			expectType: "string",
		},
		{
			name:       "TransactionIDIsString",
			file:       "transactions_list.json",
			fieldPath:  "data[0].id",
			expectType: "string",
		},
		{
			name:       "UserIDIsNumber",
			file:       "auth_get_me.json",
			fieldPath:  "data.id",
			expectType: "number",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", tc.file)
			data, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			// Navigate to field
			value := navigateJSON(jsonObj, tc.fieldPath)
			require.NotNil(t, value, "Field %s not found", tc.fieldPath)

			// Check type
			actualType := getJSONType(value)
			assert.Equal(t, tc.expectType, actualType, "Field %s has wrong type", tc.fieldPath)
		})
	}
}

// navigateJSON navigates to a field using dot notation (e.g., "data.user.id")
func navigateJSON(obj interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := obj

	for _, part := range parts {
		// Handle array indices (e.g., "[0]")
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			indexStr := strings.Trim(part, "[]")
			var index int
			fmt.Sscanf(indexStr, "%d", &index)

			arr, ok := current.([]interface{})
			if !ok || index >= len(arr) {
				return nil
			}
			current = arr[index]
			continue
		}

		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
		if current == nil {
			return nil
		}
	}

	return current
}

// getJSONType returns the type of a JSON value as a string
func getJSONType(v interface{}) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// TestJalaliDateFormat validates Jalali date formatting
func TestJalaliDateFormat(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		fieldPath string
		pattern   string // Expected pattern, e.g., "1402/10/15 14:30"
	}{
		{
			name:      "CreatedAtFormat",
			file:      "feature_detail.json",
			fieldPath: "data.created_at",
			pattern:   `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}$`, // YYYY/MM/DD HH:MM
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", tc.file)
			data, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tc.fieldPath)
			require.NotNil(t, value, "Field %s not found", tc.fieldPath)

			strValue, ok := value.(string)
			require.True(t, ok, "Field %s is not a string", tc.fieldPath)

			assert.Regexp(t, tc.pattern, strValue, "Date format mismatch")
		})
	}
}

// TestCompactNumberFormat validates compact number formatting (e.g., "1.2K", "3.5M")
func TestCompactNumberFormat(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		fieldPath string
	}{
		{
			name:      "FeaturePrice",
			file:      "feature_detail.json",
			fieldPath: "data.property.price_formatted",
		},
	}

	compactPattern := `^[\d,]+(\.\d+)?[KMB]?$` // Matches "1,234", "1.2K", "3.5M", "1B"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", tc.file)
			data, err := os.ReadFile(goldenPath)
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tc.fieldPath)
			if value == nil {
				t.Skip("Field not found, may not be in this response")
				return
			}

			strValue, ok := value.(string)
			require.True(t, ok, "Field %s is not a string", tc.fieldPath)

			assert.Regexp(t, compactPattern, strValue, "Compact number format mismatch")
		})
	}
}
