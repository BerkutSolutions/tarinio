package services

import "waf/control-plane/internal/revisions"

type setupUserCounter interface {
	Count() (int, error)
}

type setupRevisionReader interface {
	CurrentActive() (revisions.Revision, bool, error)
}

type SetupStatus struct {
	NeedsBootstrap    bool `json:"needs_bootstrap"`
	HasUsers          bool `json:"has_users"`
	HasSites          bool `json:"has_sites"`
	HasActiveRevision bool `json:"has_active_revision"`
}

type SetupService struct {
	users     setupUserCounter
	sites     SiteStore
	revisions setupRevisionReader
}

func NewSetupService(users setupUserCounter, sites SiteStore, revisions setupRevisionReader) *SetupService {
	return &SetupService{users: users, sites: sites, revisions: revisions}
}

func (s *SetupService) Status() (SetupStatus, error) {
	userCount, err := s.users.Count()
	if err != nil {
		return SetupStatus{}, err
	}
	sites, err := s.sites.List()
	if err != nil {
		return SetupStatus{}, err
	}
	hasActiveRevision := false
	if s.revisions != nil {
		_, ok, err := s.revisions.CurrentActive()
		if err != nil {
			return SetupStatus{}, err
		}
		hasActiveRevision = ok
	}
	return SetupStatus{
		NeedsBootstrap:    userCount == 0,
		HasUsers:          userCount > 0,
		HasSites:          len(sites) > 0,
		HasActiveRevision: hasActiveRevision,
	}, nil
}
