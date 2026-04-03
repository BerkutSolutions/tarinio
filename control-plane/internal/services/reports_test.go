package services

import (
	"testing"

	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
)

type fakeReportJobStore struct {
	items []jobs.Job
}

func (f *fakeReportJobStore) List() ([]jobs.Job, error) {
	return append([]jobs.Job(nil), f.items...), nil
}

type fakeReportRevisionStore struct {
	items  []revisions.Revision
	active revisions.Revision
	ok     bool
}

func (f *fakeReportRevisionStore) List() ([]revisions.Revision, error) {
	return append([]revisions.Revision(nil), f.items...), nil
}

func (f *fakeReportRevisionStore) CurrentActive() (revisions.Revision, bool, error) {
	return f.active, f.ok, nil
}

func TestReportService_RevisionSummary(t *testing.T) {
	eventStore := &fakeEventStore{
		items: []events.Event{
			{ID: "evt-1", Type: events.TypeApplyStarted, Severity: events.SeverityInfo, SourceComponent: "apply-runner", OccurredAt: "2026-04-01T10:00:00Z", Summary: "started", RelatedRevisionID: "rev-1", RelatedJobID: "job-1"},
			{ID: "evt-2", Type: events.TypeApplyFailed, Severity: events.SeverityError, SourceComponent: "apply-runner", OccurredAt: "2026-04-01T10:01:00Z", Summary: "failed", RelatedRevisionID: "rev-1", RelatedJobID: "job-1"},
			{ID: "evt-3", Type: events.TypeRollbackPerformed, Severity: events.SeverityWarning, SourceComponent: "apply-runner", OccurredAt: "2026-04-01T10:02:00Z", Summary: "rollback", RelatedRevisionID: "rev-1", RelatedJobID: "job-1", Details: map[string]any{"rolled_back_to_revision_id": "rev-0"}, SiteID: "site-a"},
			{ID: "evt-4", Type: events.TypeApplySucceeded, Severity: events.SeverityInfo, SourceComponent: "apply-runner", OccurredAt: "2026-04-01T10:03:00Z", Summary: "ok", RelatedRevisionID: "rev-2", RelatedJobID: "job-2", SiteID: "site-a"},
		},
	}
	jobStore := &fakeReportJobStore{
		items: []jobs.Job{
			{ID: "job-1", Type: jobs.TypeApply, Status: jobs.StatusFailed},
			{ID: "job-2", Type: jobs.TypeApply, Status: jobs.StatusSucceeded},
		},
	}
	revisionStore := &fakeReportRevisionStore{
		items: []revisions.Revision{
			{ID: "rev-1", Version: 1, Status: revisions.StatusFailed},
			{ID: "rev-2", Version: 2, Status: revisions.StatusActive},
		},
		active: revisions.Revision{ID: "rev-2", Version: 2, Status: revisions.StatusActive},
		ok:     true,
	}

	service := NewReportService(eventStore, jobStore, revisionStore)
	summary, err := service.RevisionSummary()
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
	if summary.ApplySuccessCount != 1 || summary.ApplyFailureCount != 1 {
		t.Fatalf("unexpected apply counts: %+v", summary)
	}
	if len(summary.RecentFailedApplies) != 1 || len(summary.RecentRollbacks) != 1 {
		t.Fatalf("unexpected recent summaries: %+v", summary)
	}
	if len(summary.EventCountsByType) == 0 || summary.EventCountsByType[0].Count < 1 {
		t.Fatalf("unexpected event counts: %+v", summary.EventCountsByType)
	}
	if len(summary.TopAffectedSites) != 1 || summary.TopAffectedSites[0].SiteID != "site-a" {
		t.Fatalf("unexpected top sites: %+v", summary.TopAffectedSites)
	}
	if summary.RevisionApply.ActiveRevisionID != "rev-2" || summary.RevisionApply.TotalApplyJobs != 2 {
		t.Fatalf("unexpected revision/apply summary: %+v", summary.RevisionApply)
	}
}
