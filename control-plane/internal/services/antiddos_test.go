package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
)

type fakeAntiDDoSStore struct {
	item antiddos.Settings
}

func (f *fakeAntiDDoSStore) Get() (antiddos.Settings, error) {
	if f.item.ConnLimit == 0 {
		f.item = antiddos.DefaultSettings()
	}
	return f.item, nil
}

func (f *fakeAntiDDoSStore) Upsert(item antiddos.Settings) (antiddos.Settings, error) {
	f.item = item
	return item, nil
}

type fakeAntiDDoSCompileService struct {
	calls int
}

func (f *fakeAntiDDoSCompileService) Create(ctx context.Context) (CompileRequestResult, error) {
	f.calls++
	return CompileRequestResult{Revision: revisions.Revision{ID: "rev-000001"}}, nil
}

type fakeAntiDDoSApplyService struct {
	calls int
}

func (f *fakeAntiDDoSApplyService) Apply(ctx context.Context, revisionID string) (jobs.Job, error) {
	f.calls++
	return jobs.Job{ID: "apply-" + revisionID, Status: jobs.StatusSucceeded, Result: "revision applied"}, nil
}

func TestAntiDDoSService_GetAndUpsert(t *testing.T) {
	service := NewAntiDDoSService(&fakeAntiDDoSStore{}, nil, nil, nil)
	current, err := service.Get()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if current.ConnLimit != antiddos.DefaultSettings().ConnLimit {
		t.Fatalf("unexpected defaults: %+v", current)
	}
	current.ConnLimit = 350
	updated, err := service.Upsert(context.Background(), current)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if updated.ConnLimit != 350 {
		t.Fatalf("unexpected upsert result: %+v", updated)
	}
}

func TestAntiDDoSService_UpsertTriggersCompileAndApply(t *testing.T) {
	compile := &fakeAntiDDoSCompileService{}
	apply := &fakeAntiDDoSApplyService{}
	service := NewAntiDDoSService(&fakeAntiDDoSStore{}, compile, apply, nil)
	current := antiddos.DefaultSettings()
	if _, err := service.Upsert(context.Background(), current); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if compile.calls != 1 {
		t.Fatalf("expected compile call, got %d", compile.calls)
	}
	if apply.calls != 1 {
		t.Fatalf("expected apply call, got %d", apply.calls)
	}
}

func TestAntiDDoSService_UpsertSkipsCompileAndApplyWhenDisabledInContext(t *testing.T) {
	compile := &fakeAntiDDoSCompileService{}
	apply := &fakeAntiDDoSApplyService{}
	service := NewAntiDDoSService(&fakeAntiDDoSStore{}, compile, apply, nil)
	current := antiddos.DefaultSettings()
	if _, err := service.Upsert(withAutoApplyDisabled(context.Background()), current); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if compile.calls != 0 {
		t.Fatalf("expected compile call to be skipped, got %d", compile.calls)
	}
	if apply.calls != 0 {
		t.Fatalf("expected apply call to be skipped, got %d", apply.calls)
	}
}
