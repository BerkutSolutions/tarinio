package services

import (
	"context"
	"fmt"
	"sync"

	"waf/control-plane/internal/jobs"
)

type autoApplyCompileService interface {
	Create(ctx context.Context) (CompileRequestResult, error)
}

type autoApplyApplyService interface {
	Apply(ctx context.Context, revisionID string) (job jobs.Job, err error)
}

var autoApplyRegistry struct {
	mu      sync.RWMutex
	compile autoApplyCompileService
	apply   autoApplyApplyService
}

func ConfigureAutoApply(compile autoApplyCompileService, apply autoApplyApplyService) {
	autoApplyRegistry.mu.Lock()
	defer autoApplyRegistry.mu.Unlock()
	autoApplyRegistry.compile = compile
	autoApplyRegistry.apply = apply
}

func runAutoApply(ctx context.Context) error {
	if isAutoApplyDisabled(ctx) {
		return nil
	}
	autoApplyRegistry.mu.RLock()
	compile := autoApplyRegistry.compile
	apply := autoApplyRegistry.apply
	autoApplyRegistry.mu.RUnlock()
	if compile == nil || apply == nil {
		return nil
	}

	compileResult, err := compile.Create(withAutoApplyDisabled(ctx))
	if err != nil {
		return fmt.Errorf("revision compile failed: %w", err)
	}
	applyJob, applyErr := apply.Apply(withAutoApplyDisabled(ctx), compileResult.Revision.ID)
	if applyErr != nil {
		return fmt.Errorf("revision apply failed: %w", applyErr)
	}
	if applyJob.Status != jobs.StatusSucceeded {
		if applyJob.Result != "" {
			return fmt.Errorf("revision apply %s failed: %s", compileResult.Revision.ID, applyJob.Result)
		}
		return fmt.Errorf("revision apply %s failed without details", compileResult.Revision.ID)
	}
	return nil
}
