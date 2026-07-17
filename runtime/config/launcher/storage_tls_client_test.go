package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorageHTTPClientTrustsConfiguredCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	cert, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("parse server certificate: %v", err)
	}
	caPath := filepath.Join(t.TempDir(), "storage-ca.crt")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), 0o600); err != nil {
		t.Fatalf("write CA: %v", err)
	}

	client, err := storageHTTPClient(&http.Client{Timeout: time.Second}, caPath)
	if err != nil {
		t.Fatalf("build TLS client: %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("trusted HTTPS request failed: %v", err)
	}
	_ = response.Body.Close()
}

func TestStorageHTTPClientRejectsInvalidCA(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-ca.crt")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("write invalid CA: %v", err)
	}
	if _, err := storageHTTPClient(&http.Client{}, path); err == nil {
		t.Fatal("expected invalid CA to be rejected")
	}
}
