package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/shared/pkg/helpers"
)

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeValidationError(w http.ResponseWriter, message string) {
	helpers.WriteValidationErrorResponseFromString(w, message, "en")
}

func writeGRPCErrorTraining(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.InvalidArgument:
		writeValidationError(w, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, st.Message())
	}
}

func decodeRequestBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return io.EOF
	}
	if r.ContentLength == 0 {
		return io.EOF
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	if len(bodyBytes) == 0 {
		return io.EOF
	}
	return json.Unmarshal(bodyBytes, v)
}
