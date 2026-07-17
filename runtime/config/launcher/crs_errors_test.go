package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteCRSRuntimeErrorReturnsStableCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeCRSRuntimeError(recorder, newCRSError(crsErrorDigestInvalid, errors.New("missing digest")))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != crsErrorDigestInvalid {
		t.Fatalf("expected structured code, got %#v", body)
	}
}
