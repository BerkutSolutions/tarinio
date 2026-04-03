package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
)

type AuditStore interface {
	Append(event audits.AuditEvent) error
	List(query audits.Query) (audits.ListResult, error)
}

type AuditService struct {
	store AuditStore
}

func NewAuditService(store AuditStore) *AuditService {
	return &AuditService{store: store}
}

func (s *AuditService) Emit(event audits.AuditEvent) {
	if s == nil || s.store == nil {
		return
	}
	event = normalizeAuditEvent(event)
	_ = s.store.Append(event)
}

func (s *AuditService) List(query audits.Query) (audits.ListResult, error) {
	if s == nil || s.store == nil {
		return audits.ListResult{}, nil
	}
	return s.store.List(query)
}

func normalizeAuditEvent(event audits.AuditEvent) audits.AuditEvent {
	if strings.TrimSpace(event.ID) == "" {
		sum := sha256.Sum256([]byte(strings.Join([]string{
			event.Action,
			event.ResourceType,
			event.ResourceID,
			event.ActorUserID,
			time.Now().UTC().Format(time.RFC3339Nano),
		}, "|")))
		event.ID = fmt.Sprintf("audit-%x", sum[:6])
	}
	if strings.TrimSpace(event.OccurredAt) == "" {
		event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}
	return event
}

func auditActorFromContext(ctx context.Context) (string, string) {
	session, ok := auth.SessionFromContext(ctx)
	if !ok {
		return "", audits.ActorIPFromContext(ctx)
	}
	return session.UserID, audits.ActorIPFromContext(ctx)
}

func auditStatus(err error) audits.Status {
	if err != nil {
		return audits.StatusFailed
	}
	return audits.StatusSucceeded
}
