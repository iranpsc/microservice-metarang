package handler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/shared/pkg/helpers"
)

// writeJSON writes a JSON response, wrapping non-error maps/slices in {"data": ...}
// unless the payload already has a top-level "data", "error", or validation shape.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	shouldSkipWrap := false
	dataType := reflect.TypeOf(data)
	if dataType != nil && dataType.Kind() == reflect.Map {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if _, hasData := dataMap["data"]; hasData {
				shouldSkipWrap = true
			}
			if _, hasError := dataMap["error"]; hasError {
				shouldSkipWrap = true
			}
			if _, hasMessage := dataMap["message"]; hasMessage {
				if _, hasErrors := dataMap["errors"]; hasErrors {
					shouldSkipWrap = true
				}
			}
		} else if dataMap, ok := data.(map[string]string); ok {
			if _, hasError := dataMap["error"]; hasError {
				shouldSkipWrap = true
			}
		}
	}

	if !shouldSkipWrap {
		data = map[string]interface{}{"data": data}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeHandlerError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.InvalidArgument:
		errorMsg := st.Message()
		if fields, decoded := helpers.DecodeValidationError(errorMsg); decoded {
			helpers.WriteValidationErrorResponseFromMap(w, fields, "en")
			return
		}
		helpers.WriteValidationErrorResponseFromString(w, errorMsg, "en")
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.FailedPrecondition:
		writeError(w, http.StatusPreconditionFailed, st.Message())
	case codes.Unavailable:
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable: "+st.Message())
	default:
		writeError(w, http.StatusInternalServerError, st.Message())
	}
}

func decodeJSONBody(r *http.Request, v interface{}) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}

func extractIDFromPath(path string, prefixes ...string) string {
	for _, prefix := range prefixes {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		id := strings.TrimPrefix(path, prefix)
		id = strings.TrimSuffix(id, "/")
		if idx := strings.Index(id, "/"); idx != -1 {
			id = id[:idx]
		}
		if idx := strings.Index(id, "?"); idx != -1 {
			id = id[:idx]
		}
		return id
	}
	return ""
}

func splitJalaliDateTime(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	parts := strings.Fields(value)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// EffectiveHTTPMethod returns the HTTP method used for routing, honoring _method spoofing.
func EffectiveHTTPMethod(r *http.Request) string {
	if r.Method != http.MethodPost {
		return r.Method
	}

	if method := spoofedMethodFromValues(r.URL.Query()["_method"]); method != "" {
		return method
	}

	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err == nil && r.MultipartForm != nil {
			if method := spoofedMethodFromValues(r.MultipartForm.Value["_method"]); method != "" {
				return method
			}
		}
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"), contentType == "":
		if err := r.ParseForm(); err == nil {
			if method := spoofedMethodFromValues(r.PostForm["_method"]); method != "" {
				return method
			}
		}
	}

	return r.Method
}

func spoofedMethodFromValues(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(values[0]))
}
