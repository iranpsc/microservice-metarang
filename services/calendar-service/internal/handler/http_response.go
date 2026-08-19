package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func writeFieldValidationError(w http.ResponseWriter, message string, errors map[string][]string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
		"message": message,
		"errors":  errors,
	})
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

// publicBaseURL returns the Kong/public gateway base (scheme+host) when APP_URL
// is set. Kong sets preserve_host=false, so r.Host is the upstream service host
// and must not be used in client-facing pagination links.
func publicBaseURL(r *http.Request) string {
	if appURL := strings.TrimSuffix(os.Getenv("APP_URL"), "/"); appURL != "" {
		return appURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

func requestBaseURL(r *http.Request) string {
	return publicBaseURL(r) + r.URL.Path
}

func requestPath(r *http.Request) string {
	return publicBaseURL(r) + r.URL.Path
}

func buildSimplePaginationLinks(r *http.Request, currentPage int32, hasMore bool) map[string]interface{} {
	baseURL := requestBaseURL(r)
	query := r.URL.Query()

	links := map[string]interface{}{}

	query.Set("page", "1")
	links["first"] = baseURL + "?" + query.Encode()
	links["last"] = nil

	if currentPage > 1 {
		query.Set("page", strconv.FormatInt(int64(currentPage-1), 10))
		links["prev"] = baseURL + "?" + query.Encode()
	} else {
		links["prev"] = nil
	}

	if hasMore {
		query.Set("page", strconv.FormatInt(int64(currentPage+1), 10))
		links["next"] = baseURL + "?" + query.Encode()
	} else {
		links["next"] = nil
	}

	return links
}
