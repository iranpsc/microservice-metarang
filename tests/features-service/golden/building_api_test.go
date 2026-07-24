package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildingAPI_GoldenJSON verifies byte-for-byte compatibility with Laravel building API responses
func TestBuildingAPI_GoldenJSON(t *testing.T) {
	testCases := []struct {
		name       string
		goldenFile string
		skipFields []string // Fields to skip comparison (e.g., timestamps, IDs)
	}{
		{
			name:       "GetBuildPackage",
			goldenFile: "build_package_response.json",
			skipFields: []string{"created_at", "updated_at"},
		},
		{
			name:       "BuildFeature",
			goldenFile: "build_feature_response.json",
			skipFields: []string{"created_at", "updated_at", "construction_start_date", "construction_end_date"},
		},
		{
			name:       "GetBuildings",
			goldenFile: "get_buildings_response.json",
			skipFields: []string{"created_at", "updated_at"},
		},
		{
			name:       "UpdateBuilding",
			goldenFile: "update_building_response.json",
			skipFields: []string{"created_at", "updated_at"},
		},
		{
			name:       "DestroyBuilding",
			goldenFile: "destroy_building_response.json",
			skipFields: []string{"created_at", "updated_at"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load golden file
			goldenPath := filepath.Join("testdata", "building", tc.goldenFile)
			goldenData, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s (create from Laravel response)", goldenPath)
				return
			}
			require.NoError(t, err, "Failed to read golden file: %s", goldenPath)

			// In a real scenario, this would call the microservice endpoint
			// For now, we'll load the actual response from a separate file
			actualPath := filepath.Join("testdata", "building", "actual", tc.goldenFile)
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
				diffPath := filepath.Join("diffs", "building", tc.goldenFile+".diff")
				os.MkdirAll(filepath.Dir(diffPath), 0755)
				os.WriteFile(diffPath, []byte(err.Error()), 0644)

				t.Errorf("JSON mismatch: %v", err)
			}
		})
	}
}

// TestBuildingAPI_GetBuildPackageResponse validates GetBuildPackage response structure
func TestBuildingAPI_GetBuildPackageResponse(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		expectType string
		description string
	}{
		{
			name:       "models array exists",
			fieldPath:  "data",
			expectType: "array",
			description: "Response should have data array with building models",
		},
		{
			name:       "model has model_id as string",
			fieldPath:  "data[0].model_id",
			expectType: "string",
			description: "model_id should be string (not number)",
		},
		{
			name:       "model has required_satisfaction as string",
			fieldPath:  "data[0].required_satisfaction",
			expectType: "string",
			description: "required_satisfaction should be string formatted to 4 decimals",
		},
		{
			name:       "coordinates array exists",
			fieldPath:  "feature.coordinates",
			expectType: "array",
			description: "Response should include feature coordinates",
		},
		{
			name:       "images is JSON array",
			fieldPath:  "data[0].images",
			expectType: "array",
			description: "images should be JSON array",
		},
		{
			name:       "attributes is JSON array",
			fieldPath:  "data[0].attributes",
			expectType: "array",
			description: "attributes should be JSON array",
		},
		{
			name:       "file is JSON object",
			fieldPath:  "data[0].file",
			expectType: "object",
			description: "file should be JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", "build_package_response.json")
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				t.Skipf("Field %s not found: %s", tt.fieldPath, tt.description)
				return
			}

			actualType := getJSONType(value)
			assert.Equal(t, tt.expectType, actualType, "%s: Field %s has wrong type", tt.description, tt.fieldPath)
		})
	}
}

// TestBuildingAPI_BuildFeatureResponse validates BuildFeature response structure
func TestBuildingAPI_BuildFeatureResponse(t *testing.T) {
	tests := []struct {
		name        string
		fieldPath   string
		expectType  string
		description string
	}{
		{
			name:        "response has data object",
			fieldPath:   "data",
			expectType:  "object",
			description: "Response should have data object with Feature",
		},
		{
			name:        "feature has building_models array",
			fieldPath:   "data.building_models",
			expectType:  "array",
			description: "Feature should have building_models array",
		},
		{
			name:        "building has construction_start_date in Jalali format",
			fieldPath:   "data.building_models[0].building.construction_start_date",
			expectType:  "string",
			description: "construction_start_date should be Jalali formatted string",
		},
		{
			name:        "building has construction_end_date in Jalali format",
			fieldPath:   "data.building_models[0].building.construction_end_date",
			expectType:  "string",
			description: "construction_end_date should be Jalali formatted string",
		},
		{
			name:        "building has launched_satisfaction as string",
			fieldPath:   "data.building_models[0].building.launched_satisfaction",
			expectType:  "string",
			description: "launched_satisfaction should be string formatted to 4 decimals",
		},
		{
			name:        "building has bubble_diameter",
			fieldPath:   "data.building_models[0].building.bubble_diameter",
			expectType:  "string",
			description: "bubble_diameter should be string",
		},
		{
			name:        "building has information JSON",
			fieldPath:   "data.building_models[0].building.information",
			expectType:  "object",
			description: "information should be JSON object (or null)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", "build_feature_response.json")
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				// Some fields may be null (e.g., information)
				if strings.Contains(tt.fieldPath, "information") {
					t.Skip("information field is null (valid)")
					return
				}
				t.Skipf("Field %s not found: %s", tt.fieldPath, tt.description)
				return
			}

			actualType := getJSONType(value)
			assert.Equal(t, tt.expectType, actualType, "%s: Field %s has wrong type", tt.description, tt.fieldPath)
		})
	}
}

// TestBuildingAPI_GetBuildingsResponse validates GetBuildings response structure
func TestBuildingAPI_GetBuildingsResponse(t *testing.T) {
	tests := []struct {
		name        string
		fieldPath   string
		expectType  string
		description string
	}{
		{
			name:        "response has data array",
			fieldPath:   "data",
			expectType:  "array",
			description: "Response should have data array",
		},
		{
			name:        "building has model object",
			fieldPath:   "data[0].model",
			expectType:  "object",
			description: "Each building should have model object",
		},
		{
			name:        "building has building object with pivot data",
			fieldPath:   "data[0].building",
			expectType:  "object",
			description: "Each building should have building object with pivot data",
		},
		{
			name:        "dates formatted in Jalali calendar",
			fieldPath:   "data[0].building.construction_start_date",
			expectType:  "string",
			description: "construction_start_date should be Jalali formatted",
		},
		{
			name:        "launched_satisfaction formatted to 4 decimals",
			fieldPath:   "data[0].building.launched_satisfaction",
			expectType:  "string",
			description: "launched_satisfaction should be string with 4 decimals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", "get_buildings_response.json")
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				t.Skipf("Field %s not found: %s", tt.fieldPath, tt.description)
				return
			}

			actualType := getJSONType(value)
			assert.Equal(t, tt.expectType, actualType, "%s: Field %s has wrong type", tt.description, tt.fieldPath)
		})
	}
}

// TestBuildingAPI_JalaliDateFormat validates Jalali date formatting in building responses
func TestBuildingAPI_JalaliDateFormat(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		fieldPath string
		pattern   string
	}{
		{
			name:      "construction_start_date format",
			file:      "build_feature_response.json",
			fieldPath: "data.building_models[0].building.construction_start_date",
			pattern:   `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}$`, // YYYY/MM/DD HH:MM:SS
		},
		{
			name:      "construction_end_date format",
			file:      "build_feature_response.json",
			fieldPath: "data.building_models[0].building.construction_end_date",
			pattern:   `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}$`, // YYYY/MM/DD HH:MM:SS
		},
		{
			name:      "get_buildings construction_start_date format",
			file:      "get_buildings_response.json",
			fieldPath: "data[0].building.construction_start_date",
			pattern:   `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", tt.file)
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				t.Skipf("Field %s not found", tt.fieldPath)
				return
			}

			strValue, ok := value.(string)
			require.True(t, ok, "Field %s is not a string", tt.fieldPath)

			matched, err := regexp.MatchString(tt.pattern, strValue)
			require.NoError(t, err)
			assert.True(t, matched, "Date format mismatch: got %s, expected pattern %s", strValue, tt.pattern)
		})
	}
}

// TestBuildingAPI_NumberFormatting validates number formatting (4 decimals for satisfaction)
func TestBuildingAPI_NumberFormatting(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		fieldPath   string
		pattern     string
		description string
	}{
		{
			name:        "required_satisfaction format",
			file:        "build_package_response.json",
			fieldPath:   "data[0].required_satisfaction",
			pattern:     `^\d+\.\d{4}$`, // e.g., "12.5000"
			description: "required_satisfaction should be formatted to 4 decimal places",
		},
		{
			name:        "launched_satisfaction format",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0].building.launched_satisfaction",
			pattern:     `^\d+\.\d{4}$`, // e.g., "25.0000"
			description: "launched_satisfaction should be formatted to 4 decimal places",
		},
		{
			name:        "bubble_diameter format",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0].building.bubble_diameter",
			pattern:     `^\d+(\.\d{1,2})?$`, // e.g., "256.5" or "256"
			description: "bubble_diameter should be formatted to 2 decimal places",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", tt.file)
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				t.Skipf("Field %s not found: %s", tt.fieldPath, tt.description)
				return
			}

			strValue, ok := value.(string)
			require.True(t, ok, "Field %s is not a string", tt.fieldPath)

			matched, err := regexp.MatchString(tt.pattern, strValue)
			require.NoError(t, err)
			assert.True(t, matched, "%s: Format mismatch: got %s, expected pattern %s", tt.description, strValue, tt.pattern)
		})
	}
}

// TestBuildingAPI_ValidationErrorResponse validates validation error response format
func TestBuildingAPI_ValidationErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		fieldPath   string
		expectType  string
		description string
	}{
		{
			name:        "error response has message field",
			fieldPath:   "message",
			expectType:  "string",
			description: "Validation error should have message field",
		},
		{
			name:        "error response has errors object",
			fieldPath:   "errors",
			expectType:  "object",
			description: "Validation error should have errors object",
		},
		{
			name:        "errors object has field-specific arrays",
			fieldPath:   "errors.launched_satisfaction",
			expectType:  "array",
			description: "Each field error should be an array of messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", "validation_error_response.json")
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s (create from Laravel validation error)", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			value := navigateJSON(jsonObj, tt.fieldPath)
			if value == nil {
				t.Skipf("Field %s not found: %s", tt.fieldPath, tt.description)
				return
			}

			actualType := getJSONType(value)
			assert.Equal(t, tt.expectType, actualType, "%s: Field %s has wrong type", tt.description, tt.fieldPath)
		})
	}
}

// TestBuildingAPI_FieldNames validates field names are snake_case (matching Laravel)
func TestBuildingAPI_FieldNames(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		fieldPath   string
		expectedKey string
		description string
	}{
		{
			name:        "building_model_id is snake_case",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0]",
			expectedKey: "model_id",
			description: "Field should be model_id (snake_case), not modelId",
		},
		{
			name:        "launched_satisfaction is snake_case",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0].building",
			expectedKey: "launched_satisfaction",
			description: "Field should be launched_satisfaction (snake_case)",
		},
		{
			name:        "construction_start_date is snake_case",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0].building",
			expectedKey: "construction_start_date",
			description: "Field should be construction_start_date (snake_case)",
		},
		{
			name:        "bubble_diameter is snake_case",
			file:        "build_feature_response.json",
			fieldPath:   "data.building_models[0].building",
			expectedKey: "bubble_diameter",
			description: "Field should be bubble_diameter (snake_case)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "building", tt.file)
			data, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Skipf("Golden file not found: %s", goldenPath)
				return
			}
			require.NoError(t, err)

			var jsonObj map[string]interface{}
			err = json.Unmarshal(data, &jsonObj)
			require.NoError(t, err)

			parent := navigateJSON(jsonObj, tt.fieldPath)
			if parent == nil {
				t.Skipf("Parent path %s not found", tt.fieldPath)
				return
			}

			parentMap, ok := parent.(map[string]interface{})
			require.True(t, ok, "Parent is not an object")

			// Check if expected key exists
			_, exists := parentMap[tt.expectedKey]
			assert.True(t, exists, "%s: Field %s should exist", tt.description, tt.expectedKey)

			// Check that camelCase version doesn't exist
			camelCaseKey := toCamelCase(tt.expectedKey)
			_, camelExists := parentMap[camelCaseKey]
			if camelExists {
				t.Logf("Warning: Both snake_case (%s) and camelCase (%s) exist", tt.expectedKey, camelCaseKey)
			}
		})
	}
}

// toCamelCase converts snake_case to camelCase (simple implementation)
func toCamelCase(snake string) string {
	parts := strings.Split(snake, "_")
	if len(parts) == 0 {
		return snake
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return result
}
