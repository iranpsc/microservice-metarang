package handler

import (
	"encoding/json"
	"net/http"
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/shared/pkg/helpers"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}, skipWrap ...bool) {
	shouldSkipWrap := len(skipWrap) > 0 && skipWrap[0]

	if data == nil {
		data = map[string]interface{}{}
	}

	if !shouldSkipWrap {
		dataType := reflect.TypeOf(data)
		dataKind := dataType.Kind()

		if dataKind == reflect.Map {
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
				if _, hasURL := dataMap["url"]; hasURL {
					shouldSkipWrap = true
				}
				if _, hasLink := dataMap["link"]; hasLink {
					shouldSkipWrap = true
				}
			}
		}

		if !shouldSkipWrap {
			data = map[string]interface{}{
				"data": data,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeHandlerError(w http.ResponseWriter, err error, locale string) {
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
			helpers.WriteValidationErrorResponseFromMap(w, fields, locale)
		} else if fields, mapped := helpers.DecodeValidationError(errorMsg); mapped {
			helpers.WriteValidationErrorResponseFromMap(w, fields, locale)
		} else {
			helpers.WriteValidationErrorResponseFromString(w, errorMsg, locale)
		}
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
