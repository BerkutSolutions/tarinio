package services

import "strings"

// filterDismissedServiceErrors copies the only mutable dashboard branch before
// applying actor-local preferences, so another request can never observe an
// actor's dismissals through the shared snapshot cache.
func (s *DashboardService) filterDismissedServiceErrors(stats DashboardStats, actorID string) DashboardStats {
	if s == nil || strings.TrimSpace(actorID) == "" || len(stats.Services) == 0 {
		return stats
	}

	servicesCopy := make([]DashboardServiceStatus, len(stats.Services))
	copy(servicesCopy, stats.Services)
	for i := range servicesCopy {
		errors := servicesCopy[i].UpstreamErrors
		if len(errors) == 0 {
			continue
		}
		visible := make([]DashboardServiceError, 0, len(errors))
		for _, item := range errors {
			if !s.isDismissed(actorID, item.ID) {
				visible = append(visible, item)
			}
		}
		servicesCopy[i].UpstreamErrors = visible
		servicesCopy[i].HasErrors = len(visible) > 0
	}
	stats.Services = servicesCopy
	return stats
}

// StatsForActorWithProcessDetails preserves the dashboard API shape while
// removing host-process inventory unless the caller passed an already
// authorised administrative capability.
func (s *DashboardService) StatsForActorWithProcessDetails(actorID string, includeProcessDetails bool) (DashboardStats, error) {
	stats, err := s.StatsForActor(actorID)
	if err != nil || includeProcessDetails {
		return stats, err
	}
	stats.System.TopCPUProcesses = []DashboardProcessStats{}
	stats.System.TopMemoryProcesses = []DashboardProcessStats{}
	return stats, nil
}
