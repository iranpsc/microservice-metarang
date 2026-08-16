package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON_WrapsAndSkipsExistingData(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]interface{}{"id": 1})
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok || data["id"].(float64) != 1 {
		t.Fatalf("expected wrap: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]interface{}{"data": []int{1}, "meta": map[string]interface{}{"total": 1}})
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["meta"]; !ok {
		t.Fatalf("should keep list meta at top level: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusBadRequest, map[string]string{"error": "nope"})
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"error"`)) || bytes.Contains(rr.Body.Bytes(), []byte(`"data"`)) {
		t.Fatalf("error payload %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, nil)
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatalf("nil payload should wrap: %s", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	writeJSON(rr, http.StatusUnprocessableEntity, map[string]interface{}{
		"message": "invalid",
		"errors":  map[string]interface{}{"field": []string{"required"}},
	})
	if bytes.Contains(rr.Body.Bytes(), []byte(`{"data":`)) {
		t.Fatalf("validation payload should not wrap: %s", rr.Body.String())
	}
}
