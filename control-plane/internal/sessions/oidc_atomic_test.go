package sessions

import (
	"sync"
	"testing"

	"waf/control-plane/internal/storage"
)

type atomicSessionBackend struct {
	mu      sync.Mutex
	content []byte
}

func (b *atomicSessionBackend) UpdateDocument(_ string, update func([]byte) ([]byte, error)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	next, err := update(append([]byte(nil), b.content...))
	if err == nil {
		b.content = next
	}
	return err
}
func (b *atomicSessionBackend) LoadDocument(string) ([]byte, error)   { return nil, storage.ErrNotFound }
func (b *atomicSessionBackend) SaveDocument(string, []byte) error     { return nil }
func (b *atomicSessionBackend) DeleteDocument(string) error           { return nil }
func (b *atomicSessionBackend) LoadBlob(string) ([]byte, error)       { return nil, storage.ErrNotFound }
func (b *atomicSessionBackend) SaveBlob(string, []byte) error         { return nil }
func (b *atomicSessionBackend) DeleteBlob(string) error               { return nil }
func (b *atomicSessionBackend) DeleteBlobsByPrefix(string) error      { return nil }
func (b *atomicSessionBackend) ListBlobKeys(string) ([]string, error) { return nil, nil }

func TestConsumeOIDCLoginChallengeAtomicAllowsOnlyOneConcurrentConsumer(t *testing.T) {
	backend := &atomicSessionBackend{content: []byte(`{"oidc_login_challenges":[{"id":"challenge","state":"state","nonce":"nonce","expires_at":"2999-01-01T00:00:00Z"}]}`)}
	store := &Store{atomic: backend, atomicKey: "sessions/sessions.json"}
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, found, err := store.ConsumeOIDCLoginChallenge("state")
			if err != nil {
				t.Errorf("consume: %v", err)
			}
			results <- found
		}()
	}
	wg.Wait()
	close(results)
	count := 0
	for found := range results {
		if found {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one consume, got %d", count)
	}
}

func TestAtomicSessionConsumesCoverLoginAndTOTPChallenges(t *testing.T) {
	backend := &atomicSessionBackend{content: []byte(`{"login_challenges":[{"id":"login","user_id":"u","expires_at":"2999-01-01T00:00:00Z"}],"totp_setup_challenges":[{"id":"totp","user_id":"u","secret":"s","expires_at":"2999-01-01T00:00:00Z"}]}`)}
	store := &Store{atomic: backend, atomicKey: "sessions/sessions.json"}
	if _, found, err := store.ConsumeLoginChallenge("login"); err != nil || !found {
		t.Fatalf("consume login = %v, %v", found, err)
	}
	if _, found, err := store.ConsumeTOTPSetupChallenge("totp"); err != nil || !found {
		t.Fatalf("consume TOTP = %v, %v", found, err)
	}
	if _, found, err := store.ConsumeLoginChallenge("login"); err != nil || found {
		t.Fatalf("replay login = %v, %v", found, err)
	}
	if _, found, err := store.ConsumeTOTPSetupChallenge("totp"); err != nil || found {
		t.Fatalf("replay TOTP = %v, %v", found, err)
	}
}
