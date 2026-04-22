package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
)

type revisionCatalogStore interface {
	List() ([]revisions.Revision, error)
	Get(revisionID string) (revisions.Revision, bool, error)
	CurrentActive() (revisions.Revision, bool, error)
	Delete(revisionID string) error
	ResetStatuses() error
}

type revisionCatalogSnapshotStore interface {
	Load(snapshotPath string) (revisionsnapshots.Snapshot, error)
	Delete(snapshotPath string) error
}

type revisionCatalogJobStore interface {
	List() ([]jobs.Job, error)
	DeleteByRevision(revisionID string) (int, error)
	DeleteByTypes(types []jobs.Type) (int, error)
}

type revisionCatalogEventReader interface {
	List() ([]events.Event, error)
	DeleteByTypes(types []events.Type) (int, error)
}

type revisionCatalogSiteReader interface {
	List() ([]sites.Site, error)
}

type RevisionCatalogResponse struct {
	Summary   RevisionApplySummary    `json:"summary"`
	Services  []RevisionServiceCard   `json:"services"`
	Revisions []RevisionCatalogItem   `json:"revisions"`
	Timeline  []RevisionTimelineEntry `json:"timeline"`
}

type RevisionCatalogProbe struct {
	ServiceCount  int `json:"service_count"`
	RevisionCount int `json:"revision_count"`
	TimelineCount int `json:"timeline_count"`
}

type RevisionServiceCard struct {
	SiteID               string `json:"site_id"`
	PrimaryHost          string `json:"primary_host"`
	Enabled              bool   `json:"enabled"`
	RevisionCount        int    `json:"revision_count"`
	ActiveRevisionCount  int    `json:"active_revision_count"`
	PendingRevisionCount int    `json:"pending_revision_count"`
	FailedRevisionCount  int    `json:"failed_revision_count"`
	LastRevisionID       string `json:"last_revision_id,omitempty"`
	LastRevisionCreated  string `json:"last_revision_created,omitempty"`
	LastRevisionStatus   string `json:"last_revision_status,omitempty"`
}

type RevisionCatalogSite struct {
	SiteID      string `json:"site_id"`
	PrimaryHost string `json:"primary_host"`
	Enabled     bool   `json:"enabled"`
}

type RevisionCatalogItem struct {
	ID                string                     `json:"id"`
	Version           int                        `json:"version"`
	CreatedAt         string                     `json:"created_at"`
	Status            string                     `json:"status"`
	Checksum          string                     `json:"checksum"`
	IsActive          bool                       `json:"is_active"`
	CompiledByUserID  string                     `json:"compiled_by_user_id,omitempty"`
	CompiledByName    string                     `json:"compiled_by_name,omitempty"`
	ApprovalStatus    string                     `json:"approval_status,omitempty"`
	RequiredApprovals int                        `json:"required_approvals,omitempty"`
	Approvals         []revisions.ApprovalRecord `json:"approvals,omitempty"`
	ApprovedAt        string                     `json:"approved_at,omitempty"`
	Signature         string                     `json:"signature,omitempty"`
	SignatureKeyID    string                     `json:"signature_key_id,omitempty"`
	Sites             []RevisionCatalogSite      `json:"sites"`
	SiteCount         int                        `json:"site_count"`
	LastApplyJobID    string                     `json:"last_apply_job_id,omitempty"`
	LastApplyStatus   string                     `json:"last_apply_status,omitempty"`
	LastApplyResult   string                     `json:"last_apply_result,omitempty"`
	LastApplyAt       string                     `json:"last_apply_at,omitempty"`
	LastEventType     string                     `json:"last_event_type,omitempty"`
	LastEventTime     string                     `json:"last_event_time,omitempty"`
	LastEventSummary  string                     `json:"last_event_summary,omitempty"`
	SnapshotError     string                     `json:"snapshot_error,omitempty"`
}

type RevisionTimelineEntry struct {
	OccurredAt      string         `json:"occurred_at"`
	Type            string         `json:"type"`
	Severity        string         `json:"severity"`
	Summary         string         `json:"summary"`
	RevisionID      string         `json:"revision_id,omitempty"`
	SiteID          string         `json:"site_id,omitempty"`
	JobID           string         `json:"job_id,omitempty"`
	SourceComponent string         `json:"source_component,omitempty"`
	Details         map[string]any `json:"details,omitempty"`
}

type RevisionCatalogService struct {
	revisions revisionCatalogStore
	snapshots revisionCatalogSnapshotStore
	jobs      revisionCatalogJobStore
	events    revisionCatalogEventReader
	sites     revisionCatalogSiteReader
}

func NewRevisionCatalogService(
	revisions revisionCatalogStore,
	snapshots revisionCatalogSnapshotStore,
	jobs revisionCatalogJobStore,
	events revisionCatalogEventReader,
	sites revisionCatalogSiteReader,
) *RevisionCatalogService {
	return &RevisionCatalogService{
		revisions: revisions,
		snapshots: snapshots,
		jobs:      jobs,
		events:    events,
		sites:     sites,
	}
}

func (s *RevisionCatalogService) List(_ context.Context) (RevisionCatalogResponse, error) {
	revisionItems, err := s.revisions.List()
	if err != nil {
		return RevisionCatalogResponse{}, err
	}
	jobItems, err := s.jobs.List()
	if err != nil {
		return RevisionCatalogResponse{}, err
	}
	eventItems, err := s.events.List()
	if err != nil {
		return RevisionCatalogResponse{}, err
	}
	siteItems, err := s.sites.List()
	if err != nil {
		return RevisionCatalogResponse{}, err
	}
	activeRevision, activeExists, err := s.revisions.CurrentActive()
	if err != nil {
		return RevisionCatalogResponse{}, err
	}

	summary := RevisionApplySummary{
		TotalRevisions: len(revisionItems),
	}
	if activeExists {
		summary.ActiveRevisionID = activeRevision.ID
	}

	latestApplyJobByRevision := make(map[string]jobs.Job)
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
		current, ok := latestApplyJobByRevision[item.TargetRevisionID]
		if !ok || current.CreatedAt <= item.CreatedAt {
			latestApplyJobByRevision[item.TargetRevisionID] = item
		}
	}

	latestEventByRevision := make(map[string]events.Event)
	timeline := make([]RevisionTimelineEntry, 0, len(eventItems))
	for _, item := range eventItems {
		if strings.TrimSpace(item.RelatedRevisionID) == "" && !isRevisionTimelineEvent(item.Type) {
			continue
		}
		timeline = append(timeline, RevisionTimelineEntry{
			OccurredAt:      item.OccurredAt,
			Type:            string(item.Type),
			Severity:        string(item.Severity),
			Summary:         item.Summary,
			RevisionID:      item.RelatedRevisionID,
			SiteID:          item.SiteID,
			JobID:           item.RelatedJobID,
			SourceComponent: item.SourceComponent,
			Details:         cloneDetails(item.Details),
		})
		if strings.TrimSpace(item.RelatedRevisionID) == "" {
			continue
		}
		current, ok := latestEventByRevision[item.RelatedRevisionID]
		if !ok || current.OccurredAt <= item.OccurredAt {
			latestEventByRevision[item.RelatedRevisionID] = item
		}
	}
	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].OccurredAt == timeline[j].OccurredAt {
			return timeline[i].RevisionID > timeline[j].RevisionID
		}
		return timeline[i].OccurredAt > timeline[j].OccurredAt
	})
	if len(timeline) > 24 {
		timeline = timeline[:24]
	}

	siteRefByID := make(map[string]RevisionCatalogSite, len(siteItems))
	for _, item := range siteItems {
		siteRefByID[item.ID] = RevisionCatalogSite{
			SiteID:      item.ID,
			PrimaryHost: item.PrimaryHost,
			Enabled:     item.Enabled,
		}
	}

	revisionsDesc := append([]revisions.Revision(nil), revisionItems...)
	sort.Slice(revisionsDesc, func(i, j int) bool {
		if revisionsDesc[i].Version == revisionsDesc[j].Version {
			return revisionsDesc[i].ID > revisionsDesc[j].ID
		}
		return revisionsDesc[i].Version > revisionsDesc[j].Version
	})

	serviceCards := make([]RevisionServiceCard, 0, len(siteItems))
	revisionCards := make([]RevisionCatalogItem, 0, len(revisionsDesc))
	serviceIndexByID := make(map[string]int, len(siteItems))
	for i, item := range siteItems {
		serviceIndexByID[item.ID] = i
		serviceCards = append(serviceCards, RevisionServiceCard{
			SiteID:      item.ID,
			PrimaryHost: item.PrimaryHost,
			Enabled:     item.Enabled,
		})
	}

	for _, revision := range revisionsDesc {
		lastApplyJob, hasLastApplyJob := latestApplyJobByRevision[revision.ID]
		derivedStatus := deriveRevisionStatus(revision, hasLastApplyJob, lastApplyJob)
		switch derivedStatus {
		case revisions.StatusPending:
			summary.PendingRevisions++
		case revisions.StatusFailed:
			summary.FailedRevisions++
		}
		card := RevisionCatalogItem{
			ID:                revision.ID,
			Version:           revision.Version,
			CreatedAt:         revision.CreatedAt,
			Status:            string(derivedStatus),
			Checksum:          revision.Checksum,
			IsActive:          activeExists && activeRevision.ID == revision.ID,
			CompiledByUserID:  revision.CompiledByUserID,
			CompiledByName:    revision.CompiledByName,
			ApprovalStatus:    string(revision.ApprovalStatus),
			RequiredApprovals: revision.RequiredApprovals,
			Approvals:         append([]revisions.ApprovalRecord(nil), revision.Approvals...),
			ApprovedAt:        revision.ApprovedAt,
			Signature:         revision.Signature,
			SignatureKeyID:    revision.SignatureKeyID,
		}
		snapshot, loadErr := s.snapshots.Load(revision.BundlePath)
		if loadErr != nil {
			card.SnapshotError = loadErr.Error()
		} else {
			card.Sites = mapRevisionSites(snapshot.Sites, siteRefByID)
			card.SiteCount = len(card.Sites)
			for _, site := range card.Sites {
				index, ok := serviceIndexByID[site.SiteID]
				if !ok {
					serviceCards = append(serviceCards, RevisionServiceCard{
						SiteID:      site.SiteID,
						PrimaryHost: site.PrimaryHost,
						Enabled:     site.Enabled,
					})
					index = len(serviceCards) - 1
					serviceIndexByID[site.SiteID] = index
				}
				serviceCards[index].RevisionCount++
				switch derivedStatus {
				case revisions.StatusActive:
					serviceCards[index].ActiveRevisionCount++
				case revisions.StatusPending:
					serviceCards[index].PendingRevisionCount++
				case revisions.StatusFailed:
					serviceCards[index].FailedRevisionCount++
				}
				if serviceCards[index].LastRevisionID == "" || serviceCards[index].LastRevisionCreated <= revision.CreatedAt {
					serviceCards[index].LastRevisionID = revision.ID
					serviceCards[index].LastRevisionCreated = revision.CreatedAt
					serviceCards[index].LastRevisionStatus = string(derivedStatus)
				}
			}
		}
		if hasLastApplyJob {
			card.LastApplyJobID = lastApplyJob.ID
			card.LastApplyStatus = string(lastApplyJob.Status)
			card.LastApplyResult = lastApplyJob.Result
			card.LastApplyAt = coalesceJobTime(lastApplyJob)
		} else {
			card.LastApplyJobID = revision.LastApplyJobID
			card.LastApplyStatus = revision.LastApplyStatus
			card.LastApplyResult = revision.LastApplyResult
			card.LastApplyAt = revision.LastApplyAt
		}
		if latestEvent, ok := latestEventByRevision[revision.ID]; ok {
			card.LastEventType = string(latestEvent.Type)
			card.LastEventTime = latestEvent.OccurredAt
			card.LastEventSummary = latestEvent.Summary
		}
		revisionCards = append(revisionCards, card)
	}

	sort.Slice(serviceCards, func(i, j int) bool {
		return serviceCards[i].SiteID < serviceCards[j].SiteID
	})

	return RevisionCatalogResponse{
		Summary:   summary,
		Services:  serviceCards,
		Revisions: revisionCards,
		Timeline:  timeline,
	}, nil
}

func (s *RevisionCatalogService) Probe(ctx context.Context) (RevisionCatalogProbe, error) {
	payload, err := s.List(ctx)
	if err != nil {
		return RevisionCatalogProbe{}, err
	}
	return RevisionCatalogProbe{
		ServiceCount:  len(payload.Services),
		RevisionCount: len(payload.Revisions),
		TimelineCount: len(payload.Timeline),
	}, nil
}

func (s *RevisionCatalogService) Delete(_ context.Context, revisionID string) error {
	revisionID = strings.ToLower(strings.TrimSpace(revisionID))
	if revisionID == "" {
		return errors.New("revision id is required")
	}

	revision, ok, err := s.revisions.Get(revisionID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("revision %s not found", revisionID)
	}
	if revision.Status == revisions.StatusActive {
		return fmt.Errorf("revision %s is active and cannot be deleted", revisionID)
	}

	if err := s.snapshots.Delete(revision.BundlePath); err != nil {
		return err
	}
	if _, err := s.jobs.DeleteByRevision(revisionID); err != nil {
		return err
	}
	return s.revisions.Delete(revisionID)
}

func (s *RevisionCatalogService) ClearTimeline(_ context.Context) error {
	if _, err := s.events.DeleteByTypes([]events.Type{
		events.TypeApplyStarted,
		events.TypeApplySucceeded,
		events.TypeApplyFailed,
		events.TypeReloadFailed,
		events.TypeHealthCheckFailed,
		events.TypeRollbackPerformed,
	}); err != nil {
		return err
	}
	if _, err := s.jobs.DeleteByTypes([]jobs.Type{jobs.TypeApply}); err != nil {
		return err
	}
	return s.revisions.ResetStatuses()
}

func mapRevisionSites(snapshotSites []sites.Site, siteRefByID map[string]RevisionCatalogSite) []RevisionCatalogSite {
	items := make([]RevisionCatalogSite, 0, len(snapshotSites))
	seen := make(map[string]struct{}, len(snapshotSites))
	for _, item := range snapshotSites {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		if ref, ok := siteRefByID[item.ID]; ok {
			items = append(items, ref)
			continue
		}
		items = append(items, RevisionCatalogSite{
			SiteID:      item.ID,
			PrimaryHost: item.PrimaryHost,
			Enabled:     item.Enabled,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].SiteID < items[j].SiteID })
	return items
}

func coalesceJobTime(item jobs.Job) string {
	if strings.TrimSpace(item.FinishedAt) != "" {
		return item.FinishedAt
	}
	if strings.TrimSpace(item.StartedAt) != "" {
		return item.StartedAt
	}
	return item.CreatedAt
}

func isRevisionTimelineEvent(eventType events.Type) bool {
	switch eventType {
	case events.TypeApplyStarted, events.TypeApplySucceeded, events.TypeApplyFailed, events.TypeReloadFailed, events.TypeHealthCheckFailed, events.TypeRollbackPerformed:
		return true
	default:
		return false
	}
}

func cloneDetails(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func deriveRevisionStatus(revision revisions.Revision, hasLastApplyJob bool, lastApplyJob jobs.Job) revisions.Status {
	if revision.Status == revisions.StatusActive {
		return revisions.StatusActive
	}
	if revision.Status == revisions.StatusFailed {
		return revisions.StatusFailed
	}
	if revision.ApprovalStatus == revisions.ApprovalPending {
		return revisions.StatusPendingApproval
	}
	if hasLastApplyJob {
		switch lastApplyJob.Status {
		case jobs.StatusSucceeded:
			return revisions.StatusInactive
		case jobs.StatusFailed:
			return revisions.StatusFailed
		case jobs.StatusPending, jobs.StatusRunning:
			return revisions.StatusPending
		}
	}
	if revision.Status == revisions.StatusInactive {
		return revisions.StatusInactive
	}
	return revisions.StatusPending
}
