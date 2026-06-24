package app

import (
	"errors"
	"testing"
	"time"
)

func TestIsRetryablePostgresStartupError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "connection refused", err: errors.New("dial tcp 172.18.0.8:5432: connect: connection refused"), expected: true},
		{name: "shutting down", err: errors.New("FATAL: the database system is shutting down (SQLSTATE 57P03)"), expected: true},
		{name: "admin command", err: errors.New("FATAL: terminating connection due to administrator command (SQLSTATE 57P01)"), expected: true},
		{name: "non transient", err: errors.New("password authentication failed"), expected: false},
	}

	for _, tc := range cases {
		if got := isRetryablePostgresStartupError(tc.err); got != tc.expected {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.expected, got)
		}
	}
}

func TestWaitForPostgresReadyWithIntervalRetriesTransientErrors(t *testing.T) {
	attempts := 0
	err := waitForPostgresReadyWithInterval(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("dial tcp 172.18.0.8:5432: connect: connection refused")
		}
		return nil
	}, 50*time.Millisecond, time.Millisecond)
	if err != nil {
		t.Fatalf("expected retry loop to recover, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestWaitForPostgresReadyWithIntervalStopsOnNonRetryableError(t *testing.T) {
	attempts := 0
	err := waitForPostgresReadyWithInterval(func() error {
		attempts++
		return errors.New("password authentication failed")
	}, 50*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatalf("expected non-retryable error")
	}
	if attempts != 1 {
		t.Fatalf("expected a single attempt, got %d", attempts)
	}
}
