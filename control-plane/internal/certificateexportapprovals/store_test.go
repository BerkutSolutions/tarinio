package certificateexportapprovals

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type atomicBackend struct {
	mu      sync.Mutex
	content []byte
}

func (b *atomicBackend) LoadDocument(string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.content) == 0 {
		return nil, errors.New("unused")
	}
	return append([]byte(nil), b.content...), nil
}
func (b *atomicBackend) SaveDocument(_ string, content []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.content = append([]byte(nil), content...)
	return nil
}
func (b *atomicBackend) DeleteDocument(string) error           { return nil }
func (b *atomicBackend) LoadBlob(string) ([]byte, error)       { return nil, errors.New("unused") }
func (b *atomicBackend) SaveBlob(string, []byte) error         { return nil }
func (b *atomicBackend) DeleteBlob(string) error               { return nil }
func (b *atomicBackend) DeleteBlobsByPrefix(string) error      { return nil }
func (b *atomicBackend) ListBlobKeys(string) ([]string, error) { return nil, nil }
func (b *atomicBackend) UpdateDocument(_ string, update func([]byte) ([]byte, error)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	next, err := update(append([]byte(nil), b.content...))
	if err != nil {
		return err
	}
	b.content = next
	return nil
}

func TestApprovalConsumeIsAtomicAndRequiresDistinctApprover(t *testing.T) {
	store, err := NewStore(t.TempDir(), &atomicBackend{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Request("approval-1", "requester", []string{"cert-b", "cert-a"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Approve("approval-1", "requester"); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("expected self approval rejection, got %v", err)
	}
	if _, err := store.Approve("approval-1", "reviewer"); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	for range 2 {
		go func() { results <- store.Consume("approval-1", "requester", []string{"cert-a", "cert-b"}) }()
	}
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("unexpected consume error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one consume, got %d", successes)
	}
}
