package certificatematerials

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStore_PutAndGet(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "certificate-materials"))
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	record, err := store.Put("cert-a", []byte("CERT"), []byte("KEY"))
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if record.CertificateRef != "certificate-materials/files/cert-a/certificate.pem" {
		t.Fatalf("unexpected certificate ref: %s", record.CertificateRef)
	}

	got, err := store.Get("cert-a")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.CertificateID != "cert-a" {
		t.Fatalf("unexpected material record: %+v", got)
	}

	certBytes, err := os.ReadFile(filepath.Join(store.files, "cert-a", certificateFileName))
	if err != nil {
		t.Fatalf("read certificate file failed: %v", err)
	}
	if string(certBytes) != "CERT" {
		t.Fatalf("unexpected certificate file content: %s", string(certBytes))
	}

	privateKeyInfo, err := os.Stat(filepath.Join(store.files, "cert-a", privateKeyFileName))
	if err != nil {
		t.Fatalf("stat private key file failed: %v", err)
	}
	if runtime.GOOS != "windows" && privateKeyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("expected private key permissions 0600, got %#o", privateKeyInfo.Mode().Perm())
	}
}

func TestStore_PutReplacesExistingFiles(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	first, err := store.Put("cert-a", []byte("CERT-1"), []byte("KEY-1"))
	if err != nil {
		t.Fatalf("first put failed: %v", err)
	}
	second, err := store.Put("cert-a", []byte("CERT-2"), []byte("KEY-2"))
	if err != nil {
		t.Fatalf("second put failed: %v", err)
	}
	if first.CreatedAt != second.CreatedAt {
		t.Fatal("expected created_at to be preserved")
	}
	if first.UpdatedAt == second.UpdatedAt {
		t.Fatal("expected updated_at to change")
	}
}

func TestStore_RejectsMissingFiles(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	if _, err := store.Put("cert-a", nil, []byte("KEY")); err == nil {
		t.Fatal("expected missing certificate file error")
	}
	if _, err := store.Put("cert-a", []byte("CERT"), nil); err == nil {
		t.Fatal("expected missing private key file error")
	}
}

func TestStore_RejectsTraversalCertificateID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	cases := []string{
		"../cert-a",
		"..\\cert-a",
		"/tmp/cert-a",
		"cert/../../x",
		"..",
	}
	for _, id := range cases {
		if _, err := store.Put(id, []byte("CERT"), []byte("KEY")); err == nil {
			t.Fatalf("expected invalid certificate_id error for %q", id)
		}
	}
}
