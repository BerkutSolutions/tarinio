package services

import (
	"sort"

	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
)

type reportEventReader interface {
	List() ([]events.Event, error)
}

type reportJobReader interface {
	List() ([]jobs.Job, error)
}

type reportRevisionReader interface {
	List() ([]revisions.Revision, error)
	CurrentActive() (revisions.Revision, bool, error)
}

type TypeCount struct {
	Type  events.Type `json:"type"`
	Count int         `json:"count"`
}

type SiteCount struct {
	SiteID string `json:"site_id"`
	Count  int    `json:"count"`
}

type FailedApplySummary struct {
	OccurredAt string `json:"occurred_at"`
	RevisionID string `json:"revision_id"`
	JobID      string `json:"job_id"`
	Summary    string `json:"summary"`
}

type RollbackSummary struct {
	OccurredAt   string `json:"occurred_at"`
	RevisionID   string `json:"revision_id"`
	JobID        string `json:"job_id"`
	RolledBackTo string `json:"rolled_back_to_revision_id"`
	Summary      string `json:"summary"`
}

type RevisionApplySummary struct {
	TotalRevisions     int    `json:"total_revisions"`
	PendingRevisions   int    `json:"pending_revisions"`
	FailedRevisions    int    `json:"failed_revisions"`
	ActiveRevisionID   string `json:"active_revision_id,omitempty"`
	TotalApplyJobs     int    `json:"total_apply_jobs"`
	FailedApplyJobs    int    `json:"failed_apply_jobs"`
	SucceededApplyJobs int    `json:"succeeded_apply_jobs"`
}

type ReportSummary struct {
	ApplySuccessCount   int                  `json:"apply_success_count"`
	ApplyFailureCount   int                  `json:"apply_failure_count"`
	RecentFailedApplies []FailedApplySummary `json:"recent_failed_applies"`
	RecentRollbacks     []RollbackSummary    `json:"recent_rollbacks"`
	EventCountsByType   []TypeCount          `json:"event_counts_by_type"`
	TopAffectedSites    []SiteCount          `json:"top_affected_sites"`
	RevisionApply       RevisionApplySummary `json:"revision_apply"`
}

type ReportService struct {
	events    reportEventReader
	jobs      reportJobReader
	revisions reportRevisionReader
}

func NewReportService(events reportEventReader, jobs reportJobReader, revisions reportRevisionReader) *ReportService {
	return &ReportService{
		events:    events,
		jobs:      jobs,
		revisions: revisions,
	}
}

func (s *ReportService) RevisionSummary() (ReportSummary, error) {
	eventItems, err := s.events.List()
	if err != nil {
		return ReportSummary{}, err
	}
	jobItems, err := s.jobs.List()
	if err != nil {
		return ReportSummary{}, err
	}
	revisionItems, err := s.revisions.List()
	if err != nil {
		return ReportSummary{}, err
	}
	activeRevision, activeExists, err := s.revisions.CurrentActive()
	if err != nil {
		return ReportSummary{}, err
	}

	typeCounts := make(map[events.Type]int)
	siteCounts := make(map[string]int)
	var recentFailed []FailedApplySummary
	var recentRollbacks []RollbackSummary
	applySuccessCount := 0
	applyFailureCount := 0

	for _, item := range eventItems {
		typeCounts[item.Type]++
		if item.SiteID != "" {
			siteCounts[item.SiteID]++
		}
		switch item.Type {
		case events.TypeApplySucceeded:
			applySuccessCount++
		case events.TypeApplyFailed:
			applyFailureCount++
			recentFailed = append(recentFailed, FailedApplySummary{
				OccurredAt: item.OccurredAt,
				RevisionID: item.RelatedRevisionID,
				JobID:      item.RelatedJobID,
				Summary:    item.Summary,
			})
		case events.TypeRollbackPerformed:
			rolledBackTo := ""
			if value, ok := item.Details["rolled_back_to_revision_id"].(string); ok {
				rolledBackTo = value
			}
			recentRollbacks = append(recentRollbacks, RollbackSummary{
				OccurredAt:   item.OccurredAt,
				RevisionID:   item.RelatedRevisionID,
				JobID:        item.RelatedJobID,
				RolledBackTo: rolledBackTo,
				Summary:      item.Summary,
			})
		}
	}

	sort.Slice(recentFailed, func(i, j int) bool {
		if recentFailed[i].OccurredAt == recentFailed[j].OccurredAt {
			return recentFailed[i].RevisionID < recentFailed[j].RevisionID
		}
		return recentFailed[i].OccurredAt > recentFailed[j].OccurredAt
	})
	if len(recentFailed) > 10 {
		recentFailed = recentFailed[:10]
	}

	sort.Slice(recentRollbacks, func(i, j int) bool {
		if recentRollbacks[i].OccurredAt == recentRollbacks[j].OccurredAt {
			return recentRollbacks[i].RevisionID < recentRollbacks[j].RevisionID
		}
		return recentRollbacks[i].OccurredAt > recentRollbacks[j].OccurredAt
	})
	if len(recentRollbacks) > 10 {
		recentRollbacks = recentRollbacks[:10]
	}

	eventCountsByType := make([]TypeCount, 0, len(typeCounts))
	for eventType, count := range typeCounts {
		eventCountsByType = append(eventCountsByType, TypeCount{Type: eventType, Count: count})
	}
	sort.Slice(eventCountsByType, func(i, j int) bool {
		if eventCountsByType[i].Count == eventCountsByType[j].Count {
			return eventCountsByType[i].Type < eventCountsByType[j].Type
		}
		return eventCountsByType[i].Count > eventCountsByType[j].Count
	})

	topAffectedSites := make([]SiteCount, 0, len(siteCounts))
	for siteID, count := range siteCounts {
		topAffectedSites = append(topAffectedSites, SiteCount{SiteID: siteID, Count: count})
	}
	sort.Slice(topAffectedSites, func(i, j int) bool {
		if topAffectedSites[i].Count == topAffectedSites[j].Count {
			return topAffectedSites[i].SiteID < topAffectedSites[j].SiteID
		}
		return topAffectedSites[i].Count > topAffectedSites[j].Count
	})
	if len(topAffectedSites) > 10 {
		topAffectedSites = topAffectedSites[:10]
	}

	summary := RevisionApplySummary{
		TotalRevisions: len(revisionItems),
	}
	if activeExists {
		summary.ActiveRevisionID = activeRevision.ID
	}
	for _, item := range revisionItems {
		switch item.Status {
		case revisions.StatusPending:
			summary.PendingRevisions++
		case revisions.StatusFailed:
			summary.FailedRevisions++
		}
	}
	for _, item := range jobItems {
		if item.Type != jobs.TypeApply {
			continue
		}
		summary.TotalApplyJobs++
		switch item.Status {
		case jobs.StatusSucceeded:
			summary.SucceededApplyJobs++
		case jobs.StatusFailed:
			summary.FailedApplyJobs++
		}
	}

	return ReportSummary{
		ApplySuccessCount:   applySuccessCount,
		ApplyFailureCount:   applyFailureCount,
		RecentFailedApplies: recentFailed,
		RecentRollbacks:     recentRollbacks,
		EventCountsByType:   eventCountsByType,
		TopAffectedSites:    topAffectedSites,
		RevisionApply:       summary,
	}, nil
}
