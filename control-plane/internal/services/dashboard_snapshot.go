package services

import (
	"sync"
	"time"

	"waf/control-plane/internal/events"
)

type dashboardSnapshotSampler struct {
	mu        sync.RWMutex
	snapshot  DashboardStats
	hasValue  bool
	started   bool
	running   bool
	stopCh    chan struct{}
	interval  time.Duration
	lastError string
}

func (s *DashboardService) StartBackgroundRefresh() {
	s.startBackgroundRefresh(s.sampler.interval)
}

func (s *DashboardService) startBackgroundRefresh(interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	s.sampler.mu.Lock()
	if s.sampler.started {
		s.sampler.mu.Unlock()
		return
	}
	s.sampler.started = true
	s.sampler.interval = interval
	s.sampler.stopCh = make(chan struct{})
	s.sampler.mu.Unlock()

	go s.runBackgroundRefreshLoop()
}

func (s *DashboardService) stopBackgroundRefresh() {
	if s == nil {
		return
	}
	s.sampler.mu.Lock()
	stopCh := s.sampler.stopCh
	if stopCh != nil {
		close(stopCh)
		s.sampler.stopCh = nil
	}
	s.sampler.started = false
	s.sampler.mu.Unlock()
}

func (s *DashboardService) runBackgroundRefreshLoop() {
	_ = s.refreshSnapshot()

	s.sampler.mu.RLock()
	interval := s.sampler.interval
	stopCh := s.sampler.stopCh
	s.sampler.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = s.refreshSnapshot()
		case <-stopCh:
			return
		}
	}
}

func (s *DashboardService) snapshot() (DashboardStats, bool) {
	if s == nil {
		return DashboardStats{}, false
	}
	s.sampler.mu.RLock()
	defer s.sampler.mu.RUnlock()
	if !s.sampler.hasValue {
		return DashboardStats{}, false
	}
	return s.sampler.snapshot, true
}

func (s *DashboardService) storeSnapshot(stats DashboardStats) {
	s.sampler.mu.Lock()
	s.sampler.snapshot = stats
	s.sampler.hasValue = true
	s.sampler.lastError = ""
	s.sampler.mu.Unlock()
}

func (s *DashboardService) refreshSnapshot() error {
	s.sampler.mu.Lock()
	if s.sampler.running {
		s.sampler.mu.Unlock()
		return nil
	}
	s.sampler.running = true
	s.sampler.mu.Unlock()
	defer func() {
		s.sampler.mu.Lock()
		s.sampler.running = false
		s.sampler.mu.Unlock()
	}()

	stats, err := s.buildSnapshot()
	if err != nil {
		s.sampler.mu.Lock()
		s.sampler.lastError = err.Error()
		s.sampler.mu.Unlock()
		return err
	}
	s.storeSnapshot(stats)
	return nil
}

func (s *DashboardService) buildSnapshot() (DashboardStats, error) {
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	out := DashboardStats{
		GeneratedAt:            now.Format(time.RFC3339Nano),
		ObservationWindowHours: 24,
	}

	out.Services = s.collectServiceStatus(now)
	for _, item := range out.Services {
		if item.Up {
			out.ServicesUp++
		} else {
			out.ServicesDown++
		}
	}

	var (
		requestRows []map[string]any
		eventItems  []events.Event
		requestErr  error
		eventErr    error
	)
	var fetchWG sync.WaitGroup
	fetchWG.Add(2)
	go func() {
		defer fetchWG.Done()
		requestRows, requestErr = s.collectRequests()
	}()
	go func() {
		defer fetchWG.Done()
		eventItems, eventErr = s.collectEvents()
	}()
	fetchWG.Wait()
	if requestErr != nil {
		return DashboardStats{}, requestErr
	}
	if eventErr != nil {
		return DashboardStats{}, eventErr
	}

	requestsDay, requestUniqueIPsDay, requestTopSites, requestTopURLs, requestsSeries, blockedFromRequestsSeries, blockedFromRequestsDay, popularErrors := summarizeRequests(requestRows, cutoff, now)
	// collectRequestsDay() fetches an exact server-side count (size:0, no
	// document transfer). We prefer it whenever the backend reports more
	// events than the collector actually returned — this covers both the
	// classic cap case (len >= 50 000) and partial-sync situations where
	// the collector returned fewer rows than really exist. When exactDay <=
	// len(requestRows) the windows are aligned and the summarised count is
	// already accurate, so we keep it (avoids overwriting filtered counts in
	// tests and low-traffic scenarios).
	if exactDay, ok := s.collectRequestsDay(); ok && exactDay > len(requestRows) {
		out.RequestsDay = exactDay
	} else {
		out.RequestsDay = requestsDay
	}
	out.RequestUniqueIPsDay = requestUniqueIPsDay
	out.RequestTopSites = requestTopSites
	out.RequestTopURLs = requestTopURLs
	out.RequestsSeries = requestsSeries
	out.BlockedSeries = blockedFromRequestsSeries
	out.PopularErrors = popularErrors

	attacksDay, blockedAttacksDay, uniqueIPsDay, topIPs, topCountries, topURLs, blockedFromEventsSeries, eventErrors := summarizeAttackEvents(eventItems, cutoff, now)
	out.AttacksDay = attacksDay
	out.BlockedAttacksDay = blockedAttacksDay
	out.UniqueAttackerIPsDay = uniqueIPsDay
	out.TopAttackerIPs = topIPs
	out.TopAttackerCountries = topCountries
	out.MostAttackedURLs = topURLs

	requestAttackFallback := summarizeRequestAttacks(requestRows, cutoff)
	eventAttackBreakdownPartial := (out.AttacksDay > 0 || out.BlockedAttacksDay > 0 || out.UniqueAttackerIPsDay > 0) &&
		(len(out.TopAttackerCountries) == 0 || len(out.MostAttackedURLs) == 0)
	if requestAttackFallback.AttacksDay > out.AttacksDay {
		out.AttacksDay = requestAttackFallback.AttacksDay
	} else if eventAttackBreakdownPartial && requestAttackFallback.AttacksDay > 0 {
		out.AttacksDay += requestAttackFallback.AttacksDay - 1
	}
	if requestAttackFallback.BlockedAttacksDay > out.BlockedAttacksDay {
		out.BlockedAttacksDay = requestAttackFallback.BlockedAttacksDay
	} else if eventAttackBreakdownPartial && requestAttackFallback.BlockedAttacksDay > 0 {
		out.BlockedAttacksDay += requestAttackFallback.BlockedAttacksDay - 1
	}
	if requestAttackFallback.UniqueIPsDay > out.UniqueAttackerIPsDay {
		out.UniqueAttackerIPsDay = requestAttackFallback.UniqueIPsDay
	} else if eventAttackBreakdownPartial && requestAttackFallback.UniqueIPsDay > 0 {
		out.UniqueAttackerIPsDay += requestAttackFallback.UniqueIPsDay - 1
	}
	if len(out.TopAttackerIPs) == 0 {
		out.TopAttackerIPs = requestAttackFallback.TopIPs
	}
	if len(requestAttackFallback.TopCountries) > 0 {
		if len(out.TopAttackerCountries) == 0 {
			out.TopAttackerCountries = requestAttackFallback.TopCountries
		} else if len(out.TopAttackerCountries) < 10 {
			out.TopAttackerCountries = mergeKeyCountsSum(out.TopAttackerCountries, requestAttackFallback.TopCountries, 10)
		}
	}
	if len(out.MostAttackedURLs) < 10 && len(requestAttackFallback.TopURLs) > 0 {
		out.MostAttackedURLs = mergeKeyCountsSum(out.MostAttackedURLs, requestAttackFallback.TopURLs, 10)
	}
	out.BlockedSeries = blockedFromEventsSeries
	if len(out.BlockedSeries) == 0 {
		out.BlockedSeries = blockedFromRequestsSeries
	} else if out.BlockedAttacksDay <= 0 {
		out.BlockedSeries = mergeSeriesMax(out.BlockedSeries, blockedFromRequestsSeries)
	}
	if out.BlockedAttacksDay <= 0 && blockedFromRequestsDay > 0 {
		out.BlockedAttacksDay = blockedFromRequestsDay
	}
	out.PopularErrors = mergeKeyCountsSum(out.PopularErrors, eventErrors, 7)
	out.System = collectSystemStats()
	out.UpstreamHealth = summarizeUpstreamHealth(requestRows, now)
	return out, nil
}
