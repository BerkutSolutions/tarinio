package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/jobs"
)

type AntiDDoSStore interface {
	Get() (antiddos.Settings, error)
	Upsert(item antiddos.Settings) (antiddos.Settings, error)
}

type AntiDDoSService struct {
	store   AntiDDoSStore
	compile antiddosRevisionCompileService
	apply   antiddosRevisionApplyService
	audits  *AuditService
}

type antiddosRevisionCompileService interface {
	Create(ctx context.Context) (CompileRequestResult, error)
}

type antiddosRevisionApplyService interface {
	Apply(ctx context.Context, revisionID string) (jobs.Job, error)
}

func NewAntiDDoSService(store AntiDDoSStore, compile antiddosRevisionCompileService, apply antiddosRevisionApplyService, audits *AuditService) *AntiDDoSService {
	return &AntiDDoSService{
		store:   store,
		compile: compile,
		apply:   apply,
		audits:  audits,
	}
}

func (s *AntiDDoSService) Get() (antiddos.Settings, error) {
	return s.store.Get()
}

func (s *AntiDDoSService) Upsert(ctx context.Context, item antiddos.Settings) (updated antiddos.Settings, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "antiddos.settings.upsert",
			ResourceType: "antiddos",
			ResourceID:   "global",
			Status:       auditStatus(err),
			Summary:      "anti-ddos settings upsert",
		})
	}()
	updated, err = s.store.Upsert(item)
	if err != nil {
		return antiddos.Settings{}, err
	}
	if err := s.compileAndApply(ctx); err != nil {
		return antiddos.Settings{}, err
	}
	return updated, nil
}

func (s *AntiDDoSService) compileAndApply(ctx context.Context) error {
	if isAutoApplyDisabled(ctx) {
		return nil
	}
	if s.compile == nil || s.apply == nil {
		return nil
	}
	compileResult, err := s.compile.Create(ctx)
	if err != nil {
		return fmt.Errorf("revision compile failed: %w", err)
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		applyJob, applyErr := s.apply.Apply(ctx, compileResult.Revision.ID)
		if applyErr == nil && applyJob.Status == jobs.StatusSucceeded {
			return nil
		}
		if applyErr != nil {
			lastErr = fmt.Errorf("revision apply failed: %w", applyErr)
			continue
		}
		lastErr = fmt.Errorf("revision apply %s finished with %s: %s", applyJob.ID, applyJob.Status, strings.TrimSpace(applyJob.Result))
		if attempt < 2 {
			time.Sleep(time.Second)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("revision apply %s failed without details", compileResult.Revision.ID)
	}
	return lastErr
}
