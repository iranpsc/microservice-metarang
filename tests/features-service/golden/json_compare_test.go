package golden

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// navigateJSON walks a decoded JSON value using dot-separated path segments.
// Segments may include array indices, e.g. "data[0].model_id", "data.building_models[0].building".
func navigateJSON(obj any, path string) any {
	current := obj
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return nil
		}
		remaining := segment
		for len(remaining) > 0 {
			switch {
			case strings.Contains(remaining, "[") && strings.IndexByte(remaining, '[') > 0:
				i := strings.IndexByte(remaining, '[')
				key := remaining[:i]
				m, ok := current.(map[string]any)
				if !ok {
					return nil
				}
				current = m[key]
				if current == nil {
					return nil
				}
				remaining = remaining[i:]
			case strings.HasPrefix(remaining, "["):
				end := strings.IndexByte(remaining, ']')
				if end < 0 {
					return nil
				}
				var idx int
				_, _ = fmt.Sscanf(remaining[1:end], "%d", &idx)
				arr, ok := current.([]any)
				if !ok || idx >= len(arr) {
					return nil
				}
				current = arr[idx]
				remaining = remaining[end+1:]
			default:
				m, ok := current.(map[string]any)
				if !ok {
					return nil
				}
				current = m[remaining]
				if current == nil {
					return nil
				}
				remaining = ""
			}
		}
	}
	return current
}

// getJSONType returns the type of a JSON-decoded value as a short label used by golden tests.
func getJSONType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

// compareJSON compares two JSON byte arrays, ignoring specified fields.
func compareJSON(golden, actual []byte, skipFields []string) error {
	var goldenObj, actualObj any

	if err := json.Unmarshal(golden, &goldenObj); err != nil {
		return fmt.Errorf("failed to unmarshal golden JSON: %w", err)
	}

	if err := json.Unmarshal(actual, &actualObj); err != nil {
		return fmt.Errorf("failed to unmarshal actual JSON: %w", err)
	}

	if len(skipFields) > 0 {
		goldenObj = removeFields(goldenObj, skipFields)
		actualObj = removeFields(actualObj, skipFields)
	}

	goldenNorm, _ := json.Marshal(goldenObj)
	actualNorm, _ := json.Marshal(actualObj)

	if !bytes.Equal(goldenNorm, actualNorm) {
		goldenPretty, _ := json.MarshalIndent(goldenObj, "", "  ")
		actualPretty, _ := json.MarshalIndent(actualObj, "", "  ")

		return fmt.Errorf("JSON mismatch:\n\nExpected (golden):\n%s\n\nActual:\n%s\n\nDiff:\n%s",
			string(goldenPretty), string(actualPretty), lineDiff(goldenPretty, actualPretty))
	}

	return nil
}

func removeFields(obj any, fields []string) any {
	switch v := obj.(type) {
	case map[string]any:
		for _, field := range fields {
			delete(v, field)
		}
		for key, val := range v {
			v[key] = removeFields(val, fields)
		}
		return v
	case []any:
		for i, val := range v {
			v[i] = removeFields(val, fields)
		}
		return v
	default:
		return v
	}
}

func lineDiff(a, b []byte) string {
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
