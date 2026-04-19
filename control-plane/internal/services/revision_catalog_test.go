package services

import (
	"context"
	"errors"
	"testing"

	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
)

type fakeRevisionCatalogRevisionStore struct {
	items  []revisions.Revision
	active revisions.Revision
	ok     bool
}

func (f *fakeRevisionCatalogRevisionStore) List() ([]revisions.Revision, error) {
	return append([]revisions.Revision(nil), f.items...), nil
}

func (f *fakeRevisionCatalogRevisionStore) Get(revisionID string) (revisions.Revision, bool, error) {
	for _, item := range f.items {
		if item.ID == revisionID {
			return item, true, nil
		}
	}
	return revisions.Revision{}, false, nil
}

func (f *fakeRevisionCatalogRevisionStore) CurrentActive() (revisions.Revision, bool, error) {
	return f.active, f.ok, nil
}

func (f *fakeRevisionCatalogRevisionStore) Delete(revisionID string) error {
	for i := range f.items {
		if f.items[i].ID != revisionID {
			continue
		}
		f.items = append(f.items[:i], f.items[i+1:]...)
		return nil
	}
	return errors.New("not found")
}

func (f *fakeRevisionCatalogRevisionStore) ResetStatuses() error {
	for i := range f.items {
		if f.items[i].Status == revisions.StatusActive {
			continue
		}
		f.items[i].Status = revisions.StatusInactive
	}
	return nil
}

type fakeRevisionCatalogSnapshotStore struct {
	snapshots map[string]revisionsnapshots.Snapshot
	deleted   []string
}

func (f *fakeRevisionCatalogSnapshotStore) Load(snapshotPath string) (revisionsnapshots.Snapshot, error) {
	item, ok := f.snapshots[snapshotPath]
	if !ok {
		return revisionsnapshots.Snapshot{}, errors.New("not found")
	}
	return item, nil
}

func (f *fakeRevisionCatalogSnapshotStore) Delete(snapshotPath string) error {
	f.deleted = append(f.deleted, snapshotPath)
	delete(f.snapshots, snapshotPath)
	return nil
}

type fakeRevisionCatalogJobStore struct {
	items        []jobs.Job
	deletedByRev string
}

func (f *fakeRevisionCatalogJobStore) List() ([]jobs.Job, error) {
	return append([]jobs.Job(nil), f.items...), nil
}

func (f *fakeRevisionCatalogJobStore) DeleteByRevision(revisionID string) (int, error) {
	f.deletedByRev = revisionID
	count := 0
	filtered := make([]jobs.Job, 0, len(f.items))
	for _, item := range f.items {
		if item.TargetRevisionID == revisionID {
			count++
			continue
		}
		filtered = append(filtered, item)
	}
	f.items = filtered
	return count, nil
}

func (f *fakeRevisionCatalogJobStore) DeleteByTypes(types []jobs.Type) (int, error) {
	allowed := make(map[jobs.Type]struct{}, len(types))
	for _, item := range types {
		allowed[item] = struct{}{}
	}
	filtered := make([]jobs.Job, 0, len(f.items))
	deleted := 0
	for _, item := range f.items {
		if _, ok := allowed[item.Type]; ok {
			deleted++
			continue
		}
		filtered = append(filtered, item)
	}
	f.items = filtered
	return deleted, nil
}

type fakeRevisionCatalogEventStore struct {
	items []events.Event
}

func (f *fakeRevisionCatalogEventStore) List() ([]events.Event, error) {
	return append([]events.Event(nil), f.items...), nil
}

func (f *fakeRevisionCatalogEventStore) DeleteByTypes(types []events.Type) (int, error) {
	allowed := make(map[events.Type]struct{}, len(types))
	for _, item := range types {
		allowed[item] = struct{}{}
	}
	filtered := make([]events.Event, 0, len(f.items))
	deleted := 0
	for _, item := range f.items {
		if _, ok := allowed[item.Type]; ok {
			deleted++
			continue
		}
		filtered = append(filtered, item)
	}
	f.items = filtered
	return deleted, nil
}

type fakeRevisionCatalogSiteStore struct {
	items []sites.Site
}

func (f *fakeRevisionCatalogSiteStore) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), f.items...), nil
}

func TestRevisionCatalogService_ListBuildsServiceTilesAndTimeline(t *testing.T) {
	service := NewRevisionCatalogService(
		&fakeRevisionCatalogRevisionStore{
			items: []revisions.Revision{
				{ID: "rev-000001", Version: 1, CreatedAt: "2026-04-01T10:00:00Z", Status: revisions.StatusPending, Checksum: "a", BundlePath: "snapshots/rev-000001.json", LastApplyJobID: "persisted-apply-rev-000001", LastApplyStatus: "succeeded", LastApplyResult: "revision applied", LastApplyAt: "2026-04-01T10:01:30Z"},
				{ID: "rev-000002", Version: 2, CreatedAt: "2026-04-02T10:00:00Z", Status: revisions.StatusActive, Checksum: "b", BundlePath: "snapshots/rev-000002.json"},
			},
			active: revisions.Revision{ID: "rev-000002"},
			ok:     true,
		},
		&fakeRevisionCatalogSnapshotStore{
			snapshots: map[string]revisionsnapshots.Snapshot{
				"snapshots/rev-000001.json": {Sites: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}}},
				"snapshots/rev-000002.json": {Sites: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}, {ID: "site-b", PrimaryHost: "b.example.com", Enabled: false}}},
			},
		},
		&fakeRevisionCatalogJobStore{
			items: []jobs.Job{
				{ID: "apply-rev-000001", Type: jobs.TypeApply, TargetRevisionID: "rev-000001", Status: jobs.StatusSucceeded, CreatedAt: "2026-04-01T10:01:00Z", Result: "revision applied"},
				{ID: "apply-rev-000002", Type: jobs.TypeApply, TargetRevisionID: "rev-000002", Status: jobs.StatusSucceeded, CreatedAt: "2026-04-02T10:01:00Z", Result: "ok"},
			},
		},
		&fakeRevisionCatalogEventStore{
			items: []events.Event{
				{ID: "evt-1", Type: events.TypeApplySucceeded, Severity: events.SeverityInfo, OccurredAt: "2026-04-01T10:02:00Z", Summary: "apply succeeded", RelatedRevisionID: "rev-000001", RelatedJobID: "apply-rev-000001", SourceComponent: "apply-runner", Details: map[string]any{"result": "ok"}},
				{ID: "evt-2", Type: events.TypeApplySucceeded, Severity: events.SeverityInfo, OccurredAt: "2026-04-02T10:02:00Z", Summary: "apply succeeded", RelatedRevisionID: "rev-000002", RelatedJobID: "apply-rev-000002", SourceComponent: "apply-runner"},
			},
		},
		&fakeRevisionCatalogSiteStore{
			items: []sites.Site{
				{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true},
				{ID: "site-b", PrimaryHost: "b.example.com", Enabled: false},
			},
		},
	)

	payload, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if payload.Summary.TotalRevisions != 2 || payload.Summary.ActiveRevisionID != "rev-000002" {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}
	if len(payload.Services) != 2 || payload.Services[0].SiteID != "site-a" {
		t.Fatalf("unexpected services: %+v", payload.Services)
	}
	if payload.Services[0].RevisionCount != 2 {
		t.Fatalf("expected site-a to aggregate both revisions, got %+v", payload.Services[0])
	}
	if payload.Services[0].PendingRevisionCount != 0 {
		t.Fatalf("expected old succeeded revision not to stay pending, got %+v", payload.Services[0])
	}
	if len(payload.Revisions) != 2 || payload.Revisions[0].ID != "rev-000002" {
		t.Fatalf("expected descending revisions list, got %+v", payload.Revisions)
	}
	if payload.Revisions[1].Status != string(revisions.StatusInactive) {
		t.Fatalf("expected old succeeded revision to become inactive, got %+v", payload.Revisions[1])
	}
	if len(payload.Timeline) != 2 || payload.Timeline[0].RevisionID != "rev-000002" {
		t.Fatalf("unexpected timeline ordering: %+v", payload.Timeline)
	}
	if payload.Timeline[1].SourceComponent != "apply-runner" || payload.Timeline[1].Details["result"] != "ok" {
		t.Fatalf("expected timeline details to be preserved, got %+v", payload.Timeline[1])
	}
}

func TestRevisionCatalogService_ListFallsBackToPersistedApplyMetadata(t *testing.T) {
	service := NewRevisionCatalogService(
		&fakeRevisionCatalogRevisionStore{
			items: []revisions.Revision{
				{
					ID:              "rev-000010",
					Version:         10,
					CreatedAt:       "2026-04-10T10:00:00Z",
					Status:          revisions.StatusInactive,
					Checksum:        "persisted",
					BundlePath:      "snapshots/rev-000010.json",
					LastApplyJobID:  "apply-rev-000010",
					LastApplyStatus: "succeeded",
					LastApplyResult: "revision applied",
					LastApplyAt:     "2026-04-10T10:05:00Z",
				},
			},
		},
		&fakeRevisionCatalogSnapshotStore{
			snapshots: map[string]revisionsnapshots.Snapshot{
				"snapshots/rev-000010.json": {Sites: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}}},
			},
		},
		&fakeRevisionCatalogJobStore{},
		&fakeRevisionCatalogEventStore{},
		&fakeRevisionCatalogSiteStore{
			items: []sites.Site{{ID: "site-a", PrimaryHost: "a.example.com", Enabled: true}},
		},
	)

	payload, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(payload.Revisions) != 1 {
		t.Fatalf("expected one revision, got %+v", payload.Revisions)
	}
	if payload.Revisions[0].LastApplyStatus != "succeeded" || payload.Revisions[0].LastApplyResult != "revision applied" || payload.Revisions[0].LastApplyAt != "2026-04-10T10:05:00Z" {
		t.Fatalf("expected persisted apply metadata fallback, got %+v", payload.Revisions[0])
	}
	if payload.Summary.TotalApplyJobs != 0 {
		t.Fatalf("expected summary apply jobs to stay based on timeline jobs, got %+v", payload.Summary)
	}
}

func TestRevisionCatalogService_DeleteRemovesSnapshotJobsAndMetadata(t *testing.T) {
	revisionStore := &fakeRevisionCatalogRevisionStore{
		items: []revisions.Revision{
			{ID: "rev-000001", Version: 1, Status: revisions.StatusFailed, BundlePath: "snapshots/rev-000001.json"},
		},
	}
	snapshotStore := &fakeRevisionCatalogSnapshotStore{
		snapshots: map[string]revisionsnapshots.Snapshot{
			"snapshots/rev-000001.json": {},
		},
	}
	jobStore := &fakeRevisionCatalogJobStore{
		items: []jobs.Job{{ID: "apply-rev-000001", Type: jobs.TypeApply, TargetRevisionID: "rev-000001"}},
	}

	service := NewRevisionCatalogService(
		revisionStore,
		snapshotStore,
		jobStore,
		&fakeRevisionCatalogEventStore{},
		&fakeRevisionCatalogSiteStore{},
	)

	if err := service.Delete(context.Background(), "rev-000001"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if len(revisionStore.items) != 0 {
		t.Fatalf("expected revision metadata to be removed, got %+v", revisionStore.items)
	}
	if len(snapshotStore.deleted) != 1 || snapshotStore.deleted[0] != "snapshots/rev-000001.json" {
		t.Fatalf("expected snapshot delete, got %+v", snapshotStore.deleted)
	}
	if jobStore.deletedByRev != "rev-000001" {
		t.Fatalf("expected jobs delete for revision, got %q", jobStore.deletedByRev)
	}
}

func TestRevisionCatalogService_ClearTimeline(t *testing.T) {
	eventStore := &fakeRevisionCatalogEventStore{
		items: []events.Event{
			{ID: "evt-1", Type: events.TypeApplyStarted},
			{ID: "evt-2", Type: events.TypeRollbackPerformed},
			{ID: "evt-3", Type: events.TypeSecurityWAF},
		},
	}
	revisionStore := &fakeRevisionCatalogRevisionStore{
		items: []revisions.Revision{
			{ID: "rev-1", Status: revisions.StatusPending},
			{ID: "rev-2", Status: revisions.StatusFailed},
			{ID: "rev-3", Status: revisions.StatusActive},
		},
	}
	service := NewRevisionCatalogService(
		revisionStore,
		&fakeRevisionCatalogSnapshotStore{},
		&fakeRevisionCatalogJobStore{},
		eventStore,
		&fakeRevisionCatalogSiteStore{},
	)

	if err := service.ClearTimeline(context.Background()); err != nil {
		t.Fatalf("clear timeline failed: %v", err)
	}
	if len(eventStore.items) != 1 || eventStore.items[0].Type != events.TypeSecurityWAF {
		t.Fatalf("unexpected retained events: %+v", eventStore.items)
	}
	for _, item := range revisionStore.items {
		if item.Status != revisions.StatusInactive && item.Status != revisions.StatusActive {
			t.Fatalf("expected statuses to be reset to inactive/active, got %+v", revisionStore.items)
		}
	}
}
