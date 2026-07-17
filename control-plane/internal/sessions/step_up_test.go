package sessions

import (
	"sync"
	"testing"
	"time"
)

func TestStepUpFailureCounterEscalatesAtomicallyAndGrantsSessionBoundAssertion(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.July, 17, 9, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for range stepUpFirstLockFailures {
		wg.Go(func() {
			if _, err := store.RecordStepUpFailure("admin", now); err != nil {
				t.Errorf("record failure: %v", err)
			}
		})
	}
	wg.Wait()
	status, err := store.StepUpStatus("session-a", "admin", now)
	if err != nil || !status.Locked || status.RetryAfterSeconds < 14*60 {
		t.Fatalf("first lock status=%+v err=%v", status, err)
	}

	for i := 0; i < 5; i++ {
		now = now.Add(15*time.Minute + time.Second)
		if _, err := store.RecordStepUpFailure("admin", now); err != nil {
			t.Fatal(err)
		}
	}
	status, err = store.StepUpStatus("session-a", "admin", now)
	if err != nil || !status.Locked || status.RetryAfterSeconds < 59*60 {
		t.Fatalf("second lock status=%+v err=%v", status, err)
	}

	now = now.Add(time.Hour + time.Second)
	status, err = store.GrantStepUp("session-a", "admin", 10*time.Minute, now)
	if err != nil || !status.Verified {
		t.Fatalf("grant status=%+v err=%v", status, err)
	}
	status, err = store.StepUpStatus("session-a", "admin", now.Add(time.Minute))
	if err != nil || !status.Verified {
		t.Fatalf("expected assertion for its session, status=%+v err=%v", status, err)
	}
	status, err = store.StepUpStatus("session-b", "admin", now.Add(time.Minute))
	if err != nil || status.Verified {
		t.Fatalf("assertion leaked to another session, status=%+v err=%v", status, err)
	}
}
