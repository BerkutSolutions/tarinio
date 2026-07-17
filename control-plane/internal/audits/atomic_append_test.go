package audits

import (
	"sync"
	"testing"

	"waf/control-plane/internal/storage"
)

type atomicAuditBackend struct {
	mu      sync.Mutex
	content []byte
}

func (b *atomicAuditBackend) UpdateDocument(_ string, update func([]byte) ([]byte, error)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	next, err := update(b.content)
	if err == nil {
		b.content = next
	}
	return err
}
func (b *atomicAuditBackend) LoadDocument(string) ([]byte, error)   { return nil, storage.ErrNotFound }
func (b *atomicAuditBackend) SaveDocument(string, []byte) error     { return nil }
func (b *atomicAuditBackend) DeleteDocument(string) error           { return nil }
func (b *atomicAuditBackend) LoadBlob(string) ([]byte, error)       { return nil, storage.ErrNotFound }
func (b *atomicAuditBackend) SaveBlob(string, []byte) error         { return nil }
func (b *atomicAuditBackend) DeleteBlob(string) error               { return nil }
func (b *atomicAuditBackend) DeleteBlobsByPrefix(string) error      { return nil }
func (b *atomicAuditBackend) ListBlobKeys(string) ([]string, error) { return nil, nil }

func TestAppendAtomicMaintainsSingleAuditHashChainAcrossStores(t *testing.T) {
	backend := &atomicAuditBackend{content: []byte(`{}`)}
	stores := []*Store{{atomic: backend, atomicKey: "audits/audit_events.json"}, {atomic: backend, atomicKey: "audits/audit_events.json"}}
	var wg sync.WaitGroup
	for index, store := range stores {
		wg.Add(1)
		go func(i int, s *Store) {
			defer wg.Done()
			if err := s.Append(AuditEvent{ID: string(rune('a' + i)), Action: "test.append", ResourceType: "test", ResourceID: string(rune('a' + i)), Status: StatusSucceeded, OccurredAt: "2026-07-16T12:00:00Z", Summary: "append"}); err != nil {
				t.Errorf("append: %v", err)
			}
		}(index, store)
	}
	wg.Wait()
	items, err := (&Store{state: storage.NewFileJSONState("unused")}).loadFromContent(backend.content)
	if err != nil {
		t.Fatalf("decode append result: %v", err)
	}
	if summary := SummarizeChain(items.Events); !summary.Valid || summary.EventCount != 2 {
		t.Fatalf("expected valid two-event chain, got %+v", summary)
	}
}
