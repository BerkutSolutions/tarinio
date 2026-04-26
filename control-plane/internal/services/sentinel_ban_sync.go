package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type sentinelAdaptivePayload struct {
	Entries []sentinelAdaptiveEntry `json:"entries"`
}

type sentinelAdaptiveEntry struct {
	IP          string   `json:"ip"`
	Action      string   `json:"action"`
	Score       float64  `json:"score"`
	ReasonCodes []string `json:"reason_codes,omitempty"`
}

type sentinelBanSyncState struct {
	Promoted map[string]string `json:"promoted,omitempty"`
}

type SentinelBanSyncService struct {
	manualBans           *ManualBanService
	adaptivePath         string
	statePath            string
	pollInterval         time.Duration
	minScore             float64
	maxPromotionsPerTick int

	mu      sync.Mutex
	state   sentinelBanSyncState
	started bool
}

func NewSentinelBanSyncService(
	manualBans *ManualBanService,
	adaptivePath string,
	statePath string,
	pollInterval time.Duration,
	minScore float64,
	maxPromotionsPerTick int,
) *SentinelBanSyncService {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	if minScore <= 0 {
		minScore = 10
	}
	if maxPromotionsPerTick <= 0 {
		maxPromotionsPerTick = 5
	}
	service := &SentinelBanSyncService{
		manualBans:           manualBans,
		adaptivePath:         strings.TrimSpace(adaptivePath),
		statePath:            strings.TrimSpace(statePath),
		pollInterval:         pollInterval,
		minScore:             minScore,
		maxPromotionsPerTick: maxPromotionsPerTick,
		state: sentinelBanSyncState{
			Promoted: map[string]string{},
		},
	}
	service.loadState()
	return service
}

func (s *SentinelBanSyncService) Start() {
	if s == nil || s.manualBans == nil {
		return
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	interval := s.pollInterval
	s.mu.Unlock()

	go func() {
		if err := s.syncOnce(context.Background()); err != nil {
			log.Printf("[warn] sentinel-ban-sync: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := s.syncOnce(context.Background()); err != nil {
				log.Printf("[warn] sentinel-ban-sync: %v", err)
			}
		}
	}()
}

func (s *SentinelBanSyncService) syncOnce(ctx context.Context) error {
	if s == nil || s.manualBans == nil {
		return nil
	}
	if strings.TrimSpace(s.adaptivePath) == "" {
		return nil
	}
	raw, err := os.ReadFile(s.adaptivePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var payload sentinelAdaptivePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	candidates := make([]sentinelAdaptiveEntry, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		ip := strings.TrimSpace(entry.IP)
		action := strings.ToLower(strings.TrimSpace(entry.Action))
		if ip == "" || net.ParseIP(ip) == nil {
			continue
		}
		if action != "drop" && action != "temp_ban" && action != "ban" {
			continue
		}
		if entry.Score < s.minScore {
			continue
		}
		candidates = append(candidates, entry)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].IP < candidates[j].IP
		}
		return candidates[i].Score > candidates[j].Score
	})

	promotedNow := 0
	changed := false
	now := time.Now().UTC().Format(time.RFC3339)
	manualCtx := withAutoApplyDisabled(ctx)
	for _, candidate := range candidates {
		if promotedNow >= s.maxPromotionsPerTick {
			break
		}
		if s.isAlreadyPromoted(candidate.IP) {
			continue
		}
		if _, err := s.manualBans.Ban(manualCtx, "__all__", candidate.IP); err != nil {
			return err
		}
		s.markPromoted(candidate.IP, now)
		promotedNow++
		changed = true
	}
	if changed {
		if err := runAutoApply(ctx); err != nil {
			return err
		}
		if err := s.saveState(); err != nil {
			return err
		}
	}
	return nil
}

func (s *SentinelBanSyncService) isAlreadyPromoted(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.state.Promoted[strings.TrimSpace(ip)]
	return ok
}

func (s *SentinelBanSyncService) markPromoted(ip string, timestamp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Promoted == nil {
		s.state.Promoted = map[string]string{}
	}
	s.state.Promoted[strings.TrimSpace(ip)] = strings.TrimSpace(timestamp)
}

func (s *SentinelBanSyncService) loadState() {
	if s == nil || strings.TrimSpace(s.statePath) == "" {
		return
	}
	raw, err := os.ReadFile(s.statePath)
	if err != nil {
		return
	}
	var state sentinelBanSyncState
	if err := json.Unmarshal(raw, &state); err != nil {
		return
	}
	if state.Promoted == nil {
		state.Promoted = map[string]string{}
	}
	s.state = state
}

func (s *SentinelBanSyncService) saveState() error {
	if s == nil || strings.TrimSpace(s.statePath) == "" {
		return nil
	}
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(s.statePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.statePath, raw, 0o644)
}
