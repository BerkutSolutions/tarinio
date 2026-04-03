package services

import "waf/control-plane/internal/revisions"

// RevisionStore defines the minimal revision metadata dependency used by the
// control-plane skeleton.
type RevisionStore interface {
	List() ([]revisions.Revision, error)
}

// RevisionService keeps control-plane ownership of revision metadata access.
type RevisionService struct {
	store RevisionStore
}

func NewRevisionService(store RevisionStore) *RevisionService {
	return &RevisionService{store: store}
}

func (s *RevisionService) RevisionCount() (int, error) {
	if s.store == nil {
		return 0, nil
	}
	items, err := s.store.List()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}
