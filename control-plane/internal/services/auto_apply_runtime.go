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
	coord   DistributedCoordinator
}

func ConfigureAutoApply(compile autoApplyCompileService, apply autoApplyApplyService, coord DistributedCoordinator) {
	autoApplyRegistry.mu.Lock()
	defer autoApplyRegistry.mu.Unlock()
	autoApplyRegistry.compile = compile
	autoApplyRegistry.apply = apply
	if coord == nil {
		autoApplyRegistry.coord = NewNoopDistributedCoordinator()
		return
	}
	autoApplyRegistry.coord = coord
}

func runAutoApply(ctx context.Context) error {
	if isAutoApplyDisabled(ctx) {
		return nil
	}
	autoApplyRegistry.mu.RLock()
	compile := autoApplyRegistry.compile
	apply := autoApplyRegistry.apply
	coord := autoApplyRegistry.coord
	autoApplyRegistry.mu.RUnlock()
	if compile == nil || apply == nil {
		return nil
	}
	if coord == nil {
		coord = NewNoopDistributedCoordinator()
	}
	if !coord.Enabled() {
		return runAutoApplyUnlocked(ctx, compile, apply)
	}
	acquired, err := coord.TryRunLeader(ctx, "ha:leader:auto-apply", coord.LeaderTTL(), func(lockCtx context.Context) error {
		return runAutoApplyUnlocked(lockCtx, compile, apply)
	})
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	return nil
}

func runAutoApplyUnlocked(ctx context.Context, compile autoApplyCompileService, apply autoApplyApplyService) error {
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
