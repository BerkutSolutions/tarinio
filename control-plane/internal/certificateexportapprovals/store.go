package certificateexportapprovals

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/storage"
)

const stateKey = "certificateexportapprovals/approvals.json"

type state struct {
	Approvals []Approval `json:"approvals"`
}

type Store struct {
	state  *storage.JSONState
	atomic storage.AtomicDocumentBackend
	mu     sync.Mutex
}

func NewStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("certificate export approvals root is required")
	}
	path := strings.TrimRight(root, "/\\") + "/certificate_export_approvals.json"
	store := &Store{}
	if atomic, ok := backend.(storage.AtomicDocumentBackend); ok && !storage.IsNilBackend(backend) {
		store.state = storage.NewBackendJSONState(backend, stateKey, path)
		store.atomic = atomic
	} else {
		store.state = storage.NewFileJSONState(path)
	}
	return store, nil
}

func (s *Store) Request(id, requesterID string, certificateIDs []string, ttl time.Duration) (Approval, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	approval := Approval{ID: strings.TrimSpace(id), RequesterID: strings.TrimSpace(requesterID), CertificateIDs: normalizeIDs(certificateIDs)}
	if approval.ID == "" || approval.RequesterID == "" || len(approval.CertificateIDs) == 0 {
		return Approval{}, errors.New("approval id, requester, and certificate ids are required")
	}
	now := time.Now().UTC()
	approval.CreatedAt, approval.ExpiresAt = now.Format(time.RFC3339Nano), now.Add(ttl).Format(time.RFC3339Nano)
	err := s.update(func(current *state) error {
		for _, item := range current.Approvals {
			if item.ID == approval.ID {
				return fmt.Errorf("certificate export approval %s already exists", approval.ID)
			}
		}
		current.Approvals = append(current.Approvals, approval)
		return nil
	})
	return approval, err
}

func (s *Store) Approve(id, actorID string) (Approval, error) {
	var approved Approval
	err := s.update(func(current *state) error {
		item, err := find(current, id)
		if err != nil {
			return err
		}
		if item.RequesterID == strings.TrimSpace(actorID) {
			return ErrSelfApproval
		}
		if item.ApprovedByID != "" {
			return ErrAlreadyApproved
		}
		if expired(*item) {
			return ErrExpired
		}
		item.ApprovedByID, item.ApprovedAt = strings.TrimSpace(actorID), time.Now().UTC().Format(time.RFC3339Nano)
		if item.ApprovedByID == "" {
			return errors.New("approver id is required")
		}
		approved = *item
		return nil
	})
	return approved, err
}

func (s *Store) Consume(id, requesterID string, certificateIDs []string) error {
	return s.update(func(current *state) error {
		item, err := find(current, id)
		if err != nil {
			return err
		}
		if item.RequesterID != strings.TrimSpace(requesterID) {
			return ErrUnauthorisedActor
		}
		if !sameIDs(item.CertificateIDs, normalizeIDs(certificateIDs)) {
			return ErrExportMismatch
		}
		if expired(*item) {
			return ErrExpired
		}
		if item.ConsumedAt != "" {
			return ErrAlreadyConsumed
		}
		if item.ApprovedByID == "" || item.ApprovedByID == item.RequesterID {
			return ErrSelfApproval
		}
		item.ConsumedAt = time.Now().UTC().Format(time.RFC3339Nano)
		return nil
	})
}

func (s *Store) update(fn func(*state) error) error {
	if s == nil || s.state == nil {
		return errors.New("certificate export approval store is unavailable")
	}
	if s.atomic != nil {
		return s.atomic.UpdateDocument(stateKey, func(raw []byte) ([]byte, error) {
			current, err := decode(raw)
			if err != nil {
				return nil, err
			}
			if err := fn(current); err != nil {
				return nil, err
			}
			return encode(current)
		})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := s.state.Load()
	if errors.Is(err, storage.ErrNotFound) {
		raw = nil
	} else if err != nil {
		return err
	}
	current, err := decode(raw)
	if err != nil {
		return err
	}
	if err := fn(current); err != nil {
		return err
	}
	content, err := encode(current)
	if err != nil {
		return err
	}
	return s.state.Save(content)
}

func decode(raw []byte) (*state, error) {
	current := &state{}
	if len(raw) == 0 || string(raw) == "{}" {
		return current, nil
	}
	if err := json.Unmarshal(raw, current); err != nil {
		return nil, err
	}
	return current, nil
}
func encode(current *state) ([]byte, error) {
	sort.Slice(current.Approvals, func(i, j int) bool { return current.Approvals[i].ID < current.Approvals[j].ID })
	content, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	return content, nil
}
func find(current *state, id string) (*Approval, error) {
	for i := range current.Approvals {
		if current.Approvals[i].ID == strings.TrimSpace(id) {
			return &current.Approvals[i], nil
		}
	}
	return nil, ErrNotFound
}
func expired(item Approval) bool {
	expires, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
	return err != nil || !time.Now().UTC().Before(expires)
}
func normalizeIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" {
			seen[id] = struct{}{}
		}
	}
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func sameIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
