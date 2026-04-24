package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileBackendReadParsesJSONAccessLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := os.WriteFile(path, []byte(`{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.10","site":"waf.example.test","status":404,"method":"GET","uri":"/.env","user_agent":"sqlmap"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	events, offset, err := NewFileBackend(path).Read(0)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if offset <= 0 {
		t.Fatalf("expected positive offset, got %d", offset)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if events[0].Path != "/.env" || events[0].Site != "waf.example.test" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}
