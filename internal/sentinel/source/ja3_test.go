package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAccessLine_JA3Present(t *testing.T) {
	line := `{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.10","site":"test.example.com","status":200,"method":"GET","uri":"/","user_agent":"curl/7.88","ja3":"abc123def456"}`
	event, ok := ParseAccessLine(line)
	if !ok {
		t.Fatal("expected ParseAccessLine to succeed")
	}
	if event.JA3 != "abc123def456" {
		t.Errorf("expected JA3=abc123def456, got %q", event.JA3)
	}
	if event.IP != "203.0.113.10" {
		t.Errorf("expected IP=203.0.113.10, got %q", event.IP)
	}
}

func TestParseAccessLine_JA3Absent(t *testing.T) {
	line := `{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.11","site":"test.example.com","status":200,"method":"GET","uri":"/","user_agent":"Mozilla/5.0"}`
	event, ok := ParseAccessLine(line)
	if !ok {
		t.Fatal("expected ParseAccessLine to succeed")
	}
	if event.JA3 != "" {
		t.Errorf("expected empty JA3, got %q", event.JA3)
	}
}

func TestParseAccessLine_JA3Empty(t *testing.T) {
	line := `{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.12","site":"test.example.com","status":200,"method":"GET","uri":"/","user_agent":"Mozilla/5.0","ja3":""}`
	event, ok := ParseAccessLine(line)
	if !ok {
		t.Fatal("expected ParseAccessLine to succeed")
	}
	if event.JA3 != "" {
		t.Errorf("expected empty JA3, got %q", event.JA3)
	}
}

func TestFileBackend_JA3InLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	line := `{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.20","site":"waf.example.test","status":200,"method":"GET","uri":"/","user_agent":"curl","ja3":"deadbeef1234"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	events, _, err := NewFileBackend(path).Read(0)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].JA3 != "deadbeef1234" {
		t.Errorf("expected JA3=deadbeef1234, got %q", events[0].JA3)
	}
}

func TestFileBackend_JA3AbsentNoError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	line := `{"timestamp":"2026-04-24T10:00:00Z","client_ip":"203.0.113.21","site":"waf.example.test","status":404,"method":"GET","uri":"/.env","user_agent":"sqlmap"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	events, _, err := NewFileBackend(path).Read(0)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].JA3 != "" {
		t.Errorf("expected empty JA3, got %q", events[0].JA3)
	}
}
