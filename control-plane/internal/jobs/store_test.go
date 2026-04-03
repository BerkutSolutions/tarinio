package jobs

import "testing"

func TestStore_CreateLifecycleAndGet(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Job{
		ID:                  "job-a",
		Type:                TypeCertificateIssue,
		TargetCertificateID: "cert-a",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("expected pending, got %s", created.Status)
	}

	running, err := store.MarkRunning("job-a")
	if err != nil {
		t.Fatalf("mark running failed: %v", err)
	}
	if running.Status != StatusRunning || running.StartedAt == "" {
		t.Fatalf("unexpected running job: %+v", running)
	}

	succeeded, err := store.MarkSucceeded("job-a", "ok")
	if err != nil {
		t.Fatalf("mark succeeded failed: %v", err)
	}
	if succeeded.Status != StatusSucceeded || succeeded.FinishedAt == "" || succeeded.Result != "ok" {
		t.Fatalf("unexpected succeeded job: %+v", succeeded)
	}

	got, ok, err := store.Get("job-a")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok || got.ID != "job-a" {
		t.Fatalf("unexpected get result: ok=%v job=%+v", ok, got)
	}
}

func TestStore_CreateCompileJob(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Job{
		ID:               "compile-rev-001",
		Type:             TypeCompile,
		TargetRevisionID: "rev-001",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Type != TypeCompile || created.TargetRevisionID != "rev-001" || created.Status != StatusPending {
		t.Fatalf("unexpected compile job: %+v", created)
	}
}

func TestStore_CreateApplyJob(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Job{
		ID:               "apply-rev-001",
		Type:             TypeApply,
		TargetRevisionID: "rev-001",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Type != TypeApply || created.TargetRevisionID != "rev-001" || created.Status != StatusPending {
		t.Fatalf("unexpected apply job: %+v", created)
	}
}

func TestStore_RejectsInvalidJob(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	if _, err := store.Create(Job{ID: "job-a", TargetCertificateID: "cert-a"}); err == nil {
		t.Fatal("expected missing type error")
	}
}
