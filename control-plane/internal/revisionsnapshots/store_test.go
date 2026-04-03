package revisionsnapshots

import (
	"os"
	"path/filepath"
	"testing"

	"waf/control-plane/internal/sites"
)

func TestStore_SavePersistsDeterministicSnapshot(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	path, checksum, err := store.Save("REV-001", Snapshot{
		Sites: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
	}, nil)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if path != "snapshots/rev-001.json" {
		t.Fatalf("unexpected path: %s", path)
	}
	if checksum == "" {
		t.Fatal("expected checksum")
	}

	content, err := os.ReadFile(filepath.Join(root, "rev-001.json"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("expected snapshot content")
	}

	path2, checksum2, err := store.Save("rev-001", Snapshot{
		Sites: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
	}, nil)
	if err != nil {
		t.Fatalf("save second failed: %v", err)
	}
	if path != path2 || checksum != checksum2 {
		t.Fatalf("expected deterministic output, got %s/%s and %s/%s", path, checksum, path2, checksum2)
	}
}

func TestStore_SaveAndReadFrozenCertificateMaterials(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(filepath.Join(root, "revision-snapshots"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, _, err = store.Save("rev-001", Snapshot{}, []MaterialContent{{
		CertificateID:  "cert-a",
		CertificatePEM: []byte("CERT-1"),
		PrivateKeyPEM:  []byte("KEY-1"),
	}})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	snapshot, err := store.Load("snapshots/rev-001.json")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(snapshot.CertificateMaterials) != 1 {
		t.Fatalf("expected one material snapshot, got %+v", snapshot.CertificateMaterials)
	}

	content, err := store.ReadMaterial(snapshot.CertificateMaterials[0].CertificateRef)
	if err != nil {
		t.Fatalf("read material failed: %v", err)
	}
	if string(content) != "CERT-1" {
		t.Fatalf("unexpected frozen material: %s", string(content))
	}
}
