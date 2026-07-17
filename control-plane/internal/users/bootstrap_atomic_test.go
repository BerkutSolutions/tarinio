package users

import (
	"encoding/json"
	"sync"
	"testing"

	"waf/control-plane/internal/storage"
)

type bootstrapAtomicBackend struct {
	mu      sync.Mutex
	content []byte
}

func (b *bootstrapAtomicBackend) UpdateDocument(_ string, update func([]byte) ([]byte, error)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	next, err := update(b.content)
	if err == nil {
		b.content = next
	}
	return err
}
func (b *bootstrapAtomicBackend) LoadDocument(string) ([]byte, error) {
	return nil, storage.ErrNotFound
}
func (b *bootstrapAtomicBackend) SaveDocument(string, []byte) error     { return nil }
func (b *bootstrapAtomicBackend) DeleteDocument(string) error           { return nil }
func (b *bootstrapAtomicBackend) LoadBlob(string) ([]byte, error)       { return nil, storage.ErrNotFound }
func (b *bootstrapAtomicBackend) SaveBlob(string, []byte) error         { return nil }
func (b *bootstrapAtomicBackend) DeleteBlob(string) error               { return nil }
func (b *bootstrapAtomicBackend) DeleteBlobsByPrefix(string) error      { return nil }
func (b *bootstrapAtomicBackend) ListBlobKeys(string) ([]string, error) { return nil, nil }

func TestSeedBootstrapAtomicCreatesOnlyOneInitialAdminAcrossStores(t *testing.T) {
	backend := &bootstrapAtomicBackend{content: []byte(`{}`)}
	bootstrap := BootstrapUser{Enabled: true, ID: "admin", Username: "admin", Email: "admin@example.test", Password: "correct horse battery staple", RoleIDs: []string{"admin"}}
	stores := []*Store{{atomic: backend, atomicKey: "users/users.json"}, {atomic: backend, atomicKey: "users/users.json"}}
	var wg sync.WaitGroup
	for _, store := range stores {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			if err := s.seedBootstrap(bootstrap); err != nil {
				t.Errorf("seed bootstrap: %v", err)
			}
		}(store)
	}
	wg.Wait()
	var current state
	if err := json.Unmarshal(backend.content, &current); err != nil {
		t.Fatalf("decode users state: %v", err)
	}
	if len(current.Users) != 1 || current.Users[0].ID != "admin" {
		t.Fatalf("expected one initial admin, got %+v", current.Users)
	}
}
