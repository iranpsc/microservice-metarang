package handler

import (
	"encoding/json"
	"io"
	"net/http"
)

func decodeJSONBody(r *http.Request, v interface{}) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}

func decodeRequestBody(r *http.Request, v interface{}) error {
	if r.Body == nil || r.ContentLength == 0 {
		if r.Body != nil {
			_ = r.Body.Close()
		}
		return io.EOF
	}
	return decodeJSONBody(r, v)
}
