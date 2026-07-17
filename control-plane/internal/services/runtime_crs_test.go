package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRuntimeCRSServicePreservesStructuredRuntimeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":"crs_release_digest_invalid","error":"CRS update request failed"}`))
	}))
	defer server.Close()

	_, err := NewRuntimeCRSService(server.URL, "").CheckUpdates(context.Background(), true)
	if RuntimeCRSErrorCode(err) != "crs_release_digest_invalid" {
		t.Fatalf("expected structured runtime error, got %v", err)
	}
	var typed *RuntimeCRSError
	if !errors.As(err, &typed) || typed.Message != "CRS update request failed" {
		t.Fatalf("expected typed runtime error, got %#v", err)
	}
}
