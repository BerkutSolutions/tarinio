package compiler

import (
	"errors"
	"fmt"
	"strings"
)

// RollbackResult is the explicit outcome of one rollback attempt.
type RollbackResult struct {
	FailedRevisionID  string
	RolledBackTo      string
	Succeeded         bool
	ActivationPointer *ActivePointer
}

// RollbackRunner restores the last known good revision using the same atomic
// activation mechanism as forward activation.
type RollbackRunner struct {
	Activator AtomicActivator
}

// Rollback switches the active pointer from a failed revision back to the
// provided known-good revision reference.
func (r RollbackRunner) Rollback(failedRevisionID string, knownGood *ActivePointer) (RollbackResult, error) {
	result := RollbackResult{
		FailedRevisionID: strings.TrimSpace(failedRevisionID),
	}
	if result.FailedRevisionID == "" {
		return result, errors.New("failed revision id is required")
	}
	if knownGood == nil {
		return result, errors.New("known-good reference is required")
	}
	if strings.TrimSpace(knownGood.RevisionID) == "" {
		return result, errors.New("known-good revision id is required")
	}
	if strings.TrimSpace(knownGood.CandidatePath) == "" {
		return result, errors.New("known-good candidate path is required")
	}
	if result.FailedRevisionID == knownGood.RevisionID {
		return result, errors.New("known-good revision must differ from failed revision")
	}

	activated, err := r.Activator.Activate(knownGood.RevisionID)
	if err != nil {
		return result, fmt.Errorf("rollback activation failed: %w", err)
	}

	result.RolledBackTo = knownGood.RevisionID
	result.Succeeded = true
	result.ActivationPointer = activated
	return result, nil
}
