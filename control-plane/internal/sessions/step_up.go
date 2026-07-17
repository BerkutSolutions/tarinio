package sessions

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	stepUpFirstLockFailures = 5
	stepUpSecondLockFails   = 10
)

var ErrStepUpLocked = errors.New("step-up verification is temporarily locked")

type StepUpFailure struct {
	UserID       string `json:"user_id"`
	FailureCount int    `json:"failure_count"`
	LastFailedAt string `json:"last_failed_at"`
	LockedUntil  string `json:"locked_until,omitempty"`
}

type StepUpAssertion struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

type StepUpStatus struct {
	Verified          bool `json:"verified"`
	Locked            bool `json:"locked"`
	RetryAfterSeconds int  `json:"retry_after_seconds,omitempty"`
}

func (s *Store) StepUpStatus(sessionID, userID string, now time.Time) (StepUpStatus, error) {
	var status StepUpStatus
	err := s.updateStepUpState(now, false, func(current *state) error {
		status = stepUpStatus(current, sessionID, userID, now)
		return nil
	})
	return status, err
}

func (s *Store) RecordStepUpFailure(userID string, now time.Time) (StepUpStatus, error) {
	userID = normalize(userID)
	if userID == "" {
		return StepUpStatus{}, errors.New("step-up user id is required")
	}
	var status StepUpStatus
	err := s.updateStepUpState(now, true, func(current *state) error {
		failure := findStepUpFailure(current, userID)
		if locked, retry := stepUpLocked(failure, now); locked {
			status = StepUpStatus{Locked: true, RetryAfterSeconds: retry}
			return nil
		}
		if failure == nil {
			current.StepUpFailures = append(current.StepUpFailures, StepUpFailure{UserID: userID})
			failure = &current.StepUpFailures[len(current.StepUpFailures)-1]
		}
		if last, err := time.Parse(time.RFC3339Nano, failure.LastFailedAt); err != nil || now.Sub(last) > time.Hour {
			failure.FailureCount = 0
		}
		failure.FailureCount++
		failure.LastFailedAt = now.UTC().Format(time.RFC3339Nano)
		if failure.FailureCount >= stepUpSecondLockFails {
			failure.LockedUntil = now.Add(time.Hour).UTC().Format(time.RFC3339Nano)
		} else if failure.FailureCount >= stepUpFirstLockFailures {
			failure.LockedUntil = now.Add(15 * time.Minute).UTC().Format(time.RFC3339Nano)
		}
		locked, retry := stepUpLocked(failure, now)
		status = StepUpStatus{Locked: locked, RetryAfterSeconds: retry}
		return nil
	})
	return status, err
}

func (s *Store) GrantStepUp(sessionID, userID string, ttl time.Duration, now time.Time) (StepUpStatus, error) {
	sessionID = strings.TrimSpace(sessionID)
	userID = normalize(userID)
	if sessionID == "" || userID == "" || ttl <= 0 {
		return StepUpStatus{}, errors.New("step-up session, user, and ttl are required")
	}
	var status StepUpStatus
	err := s.updateStepUpState(now, true, func(current *state) error {
		failure := findStepUpFailure(current, userID)
		if locked, retry := stepUpLocked(failure, now); locked {
			status = StepUpStatus{Locked: true, RetryAfterSeconds: retry}
			return ErrStepUpLocked
		}
		current.StepUpFailures = removeStepUpFailure(current.StepUpFailures, userID)
		current.StepUpAssertions = removeStepUpAssertion(current.StepUpAssertions, sessionID)
		current.StepUpAssertions = append(current.StepUpAssertions, StepUpAssertion{
			SessionID: sessionID,
			UserID:    userID,
			ExpiresAt: now.Add(ttl).UTC().Format(time.RFC3339Nano),
		})
		status = StepUpStatus{Verified: true}
		return nil
	})
	return status, err
}

func (s *Store) updateStepUpState(now time.Time, persist bool, update func(*state) error) error {
	if s.atomic != nil {
		return s.atomic.UpdateDocument(s.atomicKey, func(raw []byte) ([]byte, error) {
			current, err := decodeStepUpState(raw)
			if err != nil {
				return nil, err
			}
			pruneStepUp(current, now)
			if err := update(current); err != nil {
				return nil, err
			}
			return encodeStepUpState(current)
		})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	pruneStepUp(current, now)
	if err := update(current); err != nil {
		return err
	}
	if persist {
		return s.saveLocked(current)
	}
	return nil
}

func decodeStepUpState(raw []byte) (*state, error) {
	current := &state{}
	if len(raw) == 0 || string(raw) == "{}" {
		return current, nil
	}
	if err := json.Unmarshal(raw, current); err != nil {
		return nil, fmt.Errorf("decode sessions store: %w", err)
	}
	return current, nil
}

func encodeStepUpState(current *state) ([]byte, error) {
	raw, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func stepUpStatus(current *state, sessionID, userID string, now time.Time) StepUpStatus {
	if locked, retry := stepUpLocked(findStepUpFailure(current, normalize(userID)), now); locked {
		return StepUpStatus{Locked: true, RetryAfterSeconds: retry}
	}
	for _, assertion := range current.StepUpAssertions {
		if assertion.SessionID == strings.TrimSpace(sessionID) && assertion.UserID == normalize(userID) {
			return StepUpStatus{Verified: true}
		}
	}
	return StepUpStatus{}
}

func stepUpLocked(failure *StepUpFailure, now time.Time) (bool, int) {
	if failure == nil || strings.TrimSpace(failure.LockedUntil) == "" {
		return false, 0
	}
	until, err := time.Parse(time.RFC3339Nano, failure.LockedUntil)
	if err != nil || !until.After(now) {
		return false, 0
	}
	seconds := int(until.Sub(now).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return true, seconds
}

func findStepUpFailure(current *state, userID string) *StepUpFailure {
	for i := range current.StepUpFailures {
		if current.StepUpFailures[i].UserID == userID {
			return &current.StepUpFailures[i]
		}
	}
	return nil
}

func removeStepUpFailure(items []StepUpFailure, userID string) []StepUpFailure {
	out := items[:0]
	for _, item := range items {
		if item.UserID != userID {
			out = append(out, item)
		}
	}
	return out
}

func removeStepUpAssertion(items []StepUpAssertion, sessionID string) []StepUpAssertion {
	out := items[:0]
	for _, item := range items {
		if item.SessionID != sessionID {
			out = append(out, item)
		}
	}
	return out
}

func pruneStepUp(current *state, now time.Time) {
	assertions := current.StepUpAssertions[:0]
	for _, item := range current.StepUpAssertions {
		expiresAt, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
		if err == nil && expiresAt.After(now) {
			assertions = append(assertions, item)
		}
	}
	current.StepUpAssertions = assertions
}
