package services

import (
	"testing"
	"time"
)

func TestDashboardService_ColdBackgroundRefreshReturnsCompleteSnapshot(t *testing.T) {
	service := NewDashboardService(&fakeDashboardEventReader{}, &fakeDashboardRequestCollector{}, nil)
	service.sampler.mu.Lock()
	service.sampler.started = true
	service.sampler.running = true
	service.sampler.mu.Unlock()

	started := time.Now()
	stats, err := service.StatsForActor("e2e-admin")
	if err != nil {
		t.Fatalf("StatsForActor() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cold dashboard response blocked for %s", elapsed)
	}
	if got := len(stats.RequestsSeries); got != 24 {
		t.Fatalf("requests series buckets = %d, want 24", got)
	}
	if got := len(stats.BlockedSeries); got != 24 {
		t.Fatalf("blocked series buckets = %d, want 24", got)
	}
	if got := len(stats.AttacksSeries); got != 24 {
		t.Fatalf("attacks series buckets = %d, want 24", got)
	}
}
