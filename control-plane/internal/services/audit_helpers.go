package services

import (
	"context"
	"strings"

	"waf/control-plane/internal/audits"
)

func recordAudit(ctx context.Context, service *AuditService, event audits.AuditEvent) {
	if service == nil {
		return
	}
	actorUserID, actorIP := auditActorFromContext(ctx)
	if strings.TrimSpace(event.ActorUserID) == "" {
		event.ActorUserID = actorUserID
	}
	if strings.TrimSpace(event.ActorIP) == "" {
		event.ActorIP = actorIP
	}
	service.Emit(event)
}
