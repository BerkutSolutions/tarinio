package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/storage"
	"waf/control-plane/internal/tlsconfigs"
)

type tlsAutoRenewCertificateReader interface {
	List() ([]certificates.Certificate, error)
}

type tlsAutoRenewTLSConfigReader interface {
	List() ([]tlsconfigs.TLSConfig, error)
}

type tlsAutoRenewRenewer interface {
	Renew(ctx context.Context, certificateID string, options *ACMEIssueOptions) (jobs.Job, error)
}

type TLSAutoRenewService struct {
	store        *tlsAutoRenewSettingsStore
	certificates tlsAutoRenewCertificateReader
	tlsConfigs   tlsAutoRenewTLSConfigReader
	renewer      tlsAutoRenewRenewer
	interval     time.Duration
	coord        DistributedCoordinator

	attemptMu   sync.Mutex
	lastAttempt map[string]time.Time
}

func NewTLSAutoRenewService(root string, certificates tlsAutoRenewCertificateReader, tlsConfigs tlsAutoRenewTLSConfigReader, renewer tlsAutoRenewRenewer) (*TLSAutoRenewService, error) {
	return NewTLSAutoRenewServiceWithBackend(root, nil, certificates, tlsConfigs, renewer)
}

func NewTLSAutoRenewServiceWithBackend(root string, backend storage.Backend, certificates tlsAutoRenewCertificateReader, tlsConfigs tlsAutoRenewTLSConfigReader, renewer tlsAutoRenewRenewer) (*TLSAutoRenewService, error) {
	store, err := newTLSAutoRenewSettingsStoreWithBackend(root, backend)
	if err != nil {
		return nil, err
	}
	return &TLSAutoRenewService{
		store:        store,
		certificates: certificates,
		tlsConfigs:   tlsConfigs,
		renewer:      renewer,
		interval:     6 * time.Hour,
		coord:        NewNoopDistributedCoordinator(),
		lastAttempt:  map[string]time.Time{},
	}, nil
}

func (s *TLSAutoRenewService) Settings() (TLSAutoRenewSettings, error) {
	if s == nil || s.store == nil {
		return defaultTLSAutoRenewSettings(), nil
	}
	return s.store.Get()
}

func (s *TLSAutoRenewService) UpdateSettings(input TLSAutoRenewSettings) (TLSAutoRenewSettings, error) {
	if s == nil || s.store == nil {
		return TLSAutoRenewSettings{}, errors.New("tls auto-renew service unavailable")
	}
	return s.store.Put(input)
}

func (s *TLSAutoRenewService) Start() {
	if s == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		s.runOnce()
		for range ticker.C {
			s.runOnce()
		}
	}()
}

func (s *TLSAutoRenewService) SetCoordinator(coord DistributedCoordinator) {
	if coord == nil {
		s.coord = NewNoopDistributedCoordinator()
		return
	}
	s.coord = coord
}

func (s *TLSAutoRenewService) runOnce() {
	if s == nil || s.coord == nil {
		s.runOnceUnlocked(context.Background())
		return
	}
	if !s.coord.Enabled() {
		s.runOnceUnlocked(context.Background())
		return
	}
	_, _ = s.coord.TryRunLeader(context.Background(), "ha:leader:tls-auto-renew", s.coord.LeaderTTL(), func(ctx context.Context) error {
		s.runOnceUnlocked(ctx)
		return nil
	})
}

func (s *TLSAutoRenewService) runOnceUnlocked(ctx context.Context) {
	if s == nil || s.store == nil || s.certificates == nil || s.tlsConfigs == nil || s.renewer == nil {
		return
	}
	settings, err := s.store.Get()
	if err != nil || !settings.Enabled {
		return
	}
	tlsItems, err := s.tlsConfigs.List()
	if err != nil {
		return
	}
	linked := make(map[string]struct{}, len(tlsItems))
	for _, item := range tlsItems {
		id := strings.ToLower(strings.TrimSpace(item.CertificateID))
		if id != "" {
			linked[id] = struct{}{}
		}
	}
	if len(linked) == 0 {
		return
	}
	certificateItems, err := s.certificates.List()
	if err != nil {
		return
	}
	now := time.Now().UTC()
	threshold := time.Duration(settings.RenewBeforeDays) * 24 * time.Hour
	for _, item := range certificateItems {
		id := strings.ToLower(strings.TrimSpace(item.ID))
		if id == "" {
			continue
		}
		if _, ok := linked[id]; !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Status), "revoked") {
			continue
		}
		notAfter, parseErr := parseRFC3339Flexible(item.NotAfter)
		if parseErr != nil {
			continue
		}
		remaining := notAfter.Sub(now)
		if remaining <= 0 || remaining > threshold {
			continue
		}
		if s.wasAttemptedRecently(id, now) {
			continue
		}
		_, _ = s.renewer.Renew(ctx, id, nil)
		s.markAttempt(id, now)
	}
}

func parseRFC3339Flexible(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("invalid RFC3339 time")
}

func (s *TLSAutoRenewService) wasAttemptedRecently(certificateID string, now time.Time) bool {
	s.attemptMu.Lock()
	defer s.attemptMu.Unlock()
	last, ok := s.lastAttempt[certificateID]
	if !ok {
		return false
	}
	return now.Sub(last) < 12*time.Hour
}

func (s *TLSAutoRenewService) markAttempt(certificateID string, now time.Time) {
	s.attemptMu.Lock()
	defer s.attemptMu.Unlock()
	s.lastAttempt[certificateID] = now
}
