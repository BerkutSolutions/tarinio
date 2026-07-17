package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingRefreshLock struct{ refreshes int }

func (l *failingRefreshLock) Key() string   { return "test" }
func (l *failingRefreshLock) Token() string { return "token" }
func (l *failingRefreshLock) Refresh(context.Context, time.Duration) error {
	l.refreshes++
	return errors.New("lease lost")
}
func (l *failingRefreshLock) Release(context.Context) error { return nil }

func TestRunWithRefreshedLockCancelsCallbackAfterLeaseLoss(t *testing.T) {
	lock := &failingRefreshLock{}
	canceled := make(chan struct{}, 1)
	err := runWithRefreshedLock(context.Background(), lock, 30*time.Millisecond, func(ctx context.Context) error { <-ctx.Done(); canceled <- struct{}{}; return ctx.Err() })
	if err == nil {
		t.Fatal("expected lease-loss error")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("expected callback cancellation after lease loss")
	}
}
