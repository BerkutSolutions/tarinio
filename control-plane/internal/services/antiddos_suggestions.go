package services

import (
	"context"

	"waf/control-plane/internal/antiddossuggestions"
	"waf/control-plane/internal/audits"
)

type AntiDDoSRuleSuggestionsStore interface {
	List() ([]antiddossuggestions.Suggestion, error)
	Upsert(item antiddossuggestions.Suggestion) (antiddossuggestions.Suggestion, error)
	SetStatus(id, status string) (antiddossuggestions.Suggestion, error)
}

type AntiDDoSRuleSuggestionsService struct {
	store  AntiDDoSRuleSuggestionsStore
	audits *AuditService
}

func NewAntiDDoSRuleSuggestionsService(store AntiDDoSRuleSuggestionsStore, audits *AuditService) *AntiDDoSRuleSuggestionsService {
	return &AntiDDoSRuleSuggestionsService{store: store, audits: audits}
}

func (s *AntiDDoSRuleSuggestionsService) List() ([]antiddossuggestions.Suggestion, error) {
	return s.store.List()
}

func (s *AntiDDoSRuleSuggestionsService) Upsert(ctx context.Context, item antiddossuggestions.Suggestion) (updated antiddossuggestions.Suggestion, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "antiddos.rule_suggestion.upsert",
			ResourceType: "antiddos_rule_suggestion",
			ResourceID:   item.ID,
			Status:       auditStatus(err),
			Summary:      "anti-ddos rule suggestion upsert",
		})
	}()
	return s.store.Upsert(item)
}

func (s *AntiDDoSRuleSuggestionsService) SetStatus(ctx context.Context, id, status string) (updated antiddossuggestions.Suggestion, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "antiddos.rule_suggestion.status",
			ResourceType: "antiddos_rule_suggestion",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "anti-ddos rule suggestion status update",
		})
	}()
	return s.store.SetStatus(id, status)
}
