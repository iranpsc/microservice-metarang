package handler

import (
	"net/http"
	"strings"
)

// EffectiveHTTPMethod returns the HTTP method used for routing.
// in query parameters or form body (application/x-www-form-urlencoded or multipart).
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
	method := strings.ToUpper(strings.TrimSpace(values[0]))
	if method == "" {
		return ""
	}
	return method
}
