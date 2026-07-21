package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const runtimeStartupBundleWaitEnv = "WAF_RUNTIME_STARTUP_BUNDLE_WAIT_SECONDS"

func startupBundleWaitDurationFromEnv() (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(runtimeStartupBundleWaitEnv))
	if raw == "" || raw == "0" {
		return 0, nil
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 1 || seconds > 3600 {
		return 0, fmt.Errorf("%s must be an integer between 1 and 3600", runtimeStartupBundleWaitEnv)
	}
	return time.Duration(seconds) * time.Second, nil
}

// waitForInitialBundle prevents an opt-in runtime from booting a fallback
// configuration while a control plane is atomically publishing its first
// revision. The pointer is written only after the complete candidate is staged.
func waitForInitialBundle(runtimeRoot string, timeout time.Duration) error {
	if timeout <= 0 {
		return nil
	}
	deadline := time.Now().Add(timeout)
	var lastValidationErr error
	for {
		pointer, err := loadActivePointer(runtimeRoot)
		if err == nil {
			candidatePath, resolveErr := resolveCandidatePath(runtimeRoot, pointer.CandidatePath)
			if resolveErr != nil {
				return fmt.Errorf("resolve initial runtime bundle: %w", resolveErr)
			}
			if validateErr := validateCandidateBundle(candidatePath); validateErr != nil {
				lastValidationErr = validateErr
			} else {
				return nil
			}
		}
		if err != nil && !errors.Is(err, errActivePointerMissing) {
			return fmt.Errorf("read initial runtime bundle: %w", err)
		}
		if time.Now().After(deadline) {
			if lastValidationErr != nil {
				return fmt.Errorf("timed out after %s waiting for complete initial runtime bundle: %w", timeout, lastValidationErr)
			}
			return fmt.Errorf("timed out after %s waiting for initial runtime bundle at %s", timeout, filepath.Join(runtimeRoot, "active", "current.json"))
		}
		time.Sleep(100 * time.Millisecond)
	}
}
