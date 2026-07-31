package handler

import (
	"encoding/json"
	"net/http"
	"reflect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// writeJSON preserves the grpc-gateway contract by wrapping bare values in data.
func writeJSON(w http.ResponseWriter, statusCode int, value interface{}, skipWrap ...bool) {
	skip := len(skipWrap) > 0 && skipWrap[0]
	if value == nil {
		value = map[string]interface{}{}
	}
	if !skip {
		kind := reflect.TypeOf(value).Kind()
		if kind == reflect.Map {
			switch data := value.(type) {
			case map[string]interface{}:
				_, hasData := data["data"]
				_, hasError := data["error"]
				_, hasMessage := data["message"]
				_, hasErrors := data["errors"]
				skip = hasData || hasError || (hasMessage && hasErrors)
			case map[string]string:
				_, hasError := data["error"]
				_, hasURL := data["url"]
				_, hasLink := data["link"]
				skip = hasError || hasURL || hasLink
			}
		}
		if !skip {
			value = map[string]interface{}{"data": value}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
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
		writeError(w, http.StatusBadRequest, st.Message())
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
