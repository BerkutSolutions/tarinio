package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/wafpolicies"
)

type revisionStoreForApply interface {
	Get(revisionID string) (revisions.Revision, bool, error)
	MarkActive(revisionID string) error
	MarkFailed(revisionID string) error
}

type revisionSnapshotReader interface {
	Load(snapshotPath string) (revisionsnapshots.Snapshot, error)
	ReadMaterial(ref string) ([]byte, error)
}

type ApplyService struct {
	revisions      revisionStoreForApply
	snapshots      revisionSnapshotReader
	jobs           JobStore
	events         *EventService
	syntaxRunner   pipeline.RuntimeSyntaxRunner
	stager         pipeline.CandidateStager
	activator      pipeline.AtomicActivator
	reloadRunner   pipeline.ReloadHealthRunner
	rollbackRunner pipeline.RollbackRunner
	activationRoot string
	audits         *AuditService
}

func NewApplyService(runtimeRoot string, revisions revisionStoreForApply, snapshots revisionSnapshotReader, jobs JobStore, eventService *EventService, syntaxExecutor pipeline.CommandExecutor, reloadExecutor pipeline.CommandExecutor, healthChecker pipeline.HealthChecker, audits *AuditService) *ApplyService {
	root := strings.TrimSpace(runtimeRoot)
	return &ApplyService{
		revisions:      revisions,
		snapshots:      snapshots,
		jobs:           jobs,
		events:         eventService,
		syntaxRunner:   pipeline.RuntimeSyntaxRunner{Executor: syntaxExecutor},
		stager:         pipeline.CandidateStager{Root: root},
		activator:      pipeline.AtomicActivator{Root: root},
		reloadRunner:   pipeline.ReloadHealthRunner{Executor: reloadExecutor, HealthChecker: healthChecker},
		rollbackRunner: pipeline.RollbackRunner{Activator: pipeline.AtomicActivator{Root: root}},
		activationRoot: root,
		audits:         audits,
	}
}

func (s *ApplyService) Apply(ctx context.Context, revisionID string) (job jobs.Job, err error) {
	defer func() {
		details := map[string]any(nil)
		if err != nil {
			details = map[string]any{"error": err.Error()}
		}
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:            "revision.apply_trigger",
			ResourceType:      "revision",
			ResourceID:        revisionID,
			RelatedRevisionID: revisionID,
			RelatedJobID:      job.ID,
			Status:            auditStatus(err),
			Summary:           "revision apply trigger",
			Details:           details,
		})
	}()
	revisionID = strings.ToLower(strings.TrimSpace(revisionID))
	if revisionID == "" {
		return jobs.Job{}, errors.New("revision id is required")
	}

	revision, ok, err := s.revisions.Get(revisionID)
	if err != nil {
		return jobs.Job{}, err
	}
	if !ok {
		return jobs.Job{}, fmt.Errorf("revision %s not found", revisionID)
	}

	job, err = s.jobs.Create(jobs.Job{
		// Job IDs must be unique across retries. Dev fast start (and operators) may attempt to apply the same
		// revision multiple times if runtime is not ready yet. Use a time-based suffix to avoid "already exists".
		ID:               fmt.Sprintf("apply-%s-%d", revisionID, time.Now().UTC().UnixNano()),
		Type:             jobs.TypeApply,
		TargetRevisionID: revisionID,
	})
	if err != nil {
		return jobs.Job{}, err
	}
	if _, err := s.jobs.MarkRunning(job.ID); err != nil {
		return jobs.Job{}, err
	}
	s.emitEvent(events.Event{
		Type:              events.TypeApplyStarted,
		Severity:          events.SeverityInfo,
		SourceComponent:   "apply-runner",
		Summary:           "apply started",
		RelatedRevisionID: revisionID,
		RelatedJobID:      job.ID,
	})

	if err := s.runApply(ctx, revision, job.ID); err != nil {
		s.emitEvent(events.Event{
			Type:              events.TypeApplyFailed,
			Severity:          events.SeverityError,
			SourceComponent:   "apply-runner",
			Summary:           "apply failed",
			RelatedRevisionID: revisionID,
			RelatedJobID:      job.ID,
			Details:           map[string]any{"error": err.Error()},
		})
		_ = s.revisions.MarkFailed(revisionID)
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	if err := s.revisions.MarkActive(revisionID); err != nil {
		s.emitEvent(events.Event{
			Type:              events.TypeApplyFailed,
			Severity:          events.SeverityError,
			SourceComponent:   "apply-runner",
			Summary:           "apply failed",
			RelatedRevisionID: revisionID,
			RelatedJobID:      job.ID,
			Details:           map[string]any{"error": err.Error()},
		})
		_ = s.revisions.MarkFailed(revisionID)
		return s.jobs.MarkFailed(job.ID, err.Error())
	}
	s.emitEvent(events.Event{
		Type:              events.TypeApplySucceeded,
		Severity:          events.SeverityInfo,
		SourceComponent:   "apply-runner",
		Summary:           "apply succeeded",
		RelatedRevisionID: revisionID,
		RelatedJobID:      job.ID,
	})
	return s.jobs.MarkSucceeded(job.ID, "revision applied")
}

func (s *ApplyService) runApply(ctx context.Context, revision revisions.Revision, jobID string) error {
	snapshot, err := s.snapshots.Load(revision.BundlePath)
	if err != nil {
		return err
	}

	bundle, err := s.compileBundle(revision, snapshot)
	if err != nil {
		return err
	}
	if err := pipeline.ValidateRevisionBundle(bundle); err != nil {
		return err
	}
	if err := s.syntaxRunner.Validate(bundle); err != nil {
		return err
	}

	knownGood, _ := pipeline.LoadActivePointer(s.activationRoot)

	if _, err := s.stager.Stage(bundle); err != nil {
		return err
	}
	active, err := s.activator.Activate(revision.ID)
	if err != nil {
		return err
	}

	result := s.reloadRunner.Run(active)
	if result.ReloadSucceeded && result.HealthCheckSucceeded {
		return nil
	}

	if result.ReloadError != nil {
		s.emitEvent(events.Event{
			Type:              events.TypeReloadFailed,
			Severity:          events.SeverityError,
			SourceComponent:   "runtime",
			Summary:           "reload failed",
			RelatedRevisionID: revision.ID,
			RelatedJobID:      jobID,
			Details:           map[string]any{"error": result.ReloadError.Error()},
		})
	}
	if result.HealthCheckError != nil {
		s.emitEvent(events.Event{
			Type:              events.TypeHealthCheckFailed,
			Severity:          events.SeverityError,
			SourceComponent:   "runtime",
			Summary:           "health-check failed",
			RelatedRevisionID: revision.ID,
			RelatedJobID:      jobID,
			Details:           map[string]any{"error": result.HealthCheckError.Error()},
		})
	}

	if knownGood != nil {
		rollbackResult, rollbackErr := s.rollbackRunner.Rollback(revision.ID, knownGood)
		if rollbackErr != nil {
			if result.HealthCheckError != nil {
				return fmt.Errorf("%v; rollback failed: %w", result.HealthCheckError, rollbackErr)
			}
			if result.ReloadError != nil {
				return fmt.Errorf("%v; rollback failed: %w", result.ReloadError, rollbackErr)
			}
			return fmt.Errorf("rollback failed: %w", rollbackErr)
		}
		if rollbackResult.Succeeded {
			s.emitEvent(events.Event{
				Type:              events.TypeRollbackPerformed,
				Severity:          events.SeverityWarning,
				SourceComponent:   "apply-runner",
				Summary:           "rollback performed",
				RelatedRevisionID: revision.ID,
				RelatedJobID:      jobID,
				Details: map[string]any{
					"rolled_back_to_revision_id": rollbackResult.RolledBackTo,
				},
			})
			recordAudit(ctx, s.audits, audits.AuditEvent{
				Action:            "revision.rollback_trigger",
				ResourceType:      "revision",
				ResourceID:        revision.ID,
				RelatedRevisionID: revision.ID,
				RelatedJobID:      jobID,
				Status:            audits.StatusSucceeded,
				Summary:           "rollback performed",
				Details: map[string]any{
					"rolled_back_to_revision_id": rollbackResult.RolledBackTo,
				},
			})
		}
	}
	if result.HealthCheckError != nil {
		return result.HealthCheckError
	}
	if result.ReloadError != nil {
		return result.ReloadError
	}
	return errors.New("apply failed")
}

func (s *ApplyService) emitEvent(event events.Event) {
	if s.events == nil {
		return
	}
	_, _ = s.events.Emit(event)
}

func (s *ApplyService) compileBundle(revision revisions.Revision, snapshot revisionsnapshots.Snapshot) (*pipeline.RevisionBundle, error) {
	siteInputs, upstreamInputs := mapSiteUpstreamInputs(snapshot.Sites, snapshot.Upstreams, snapshot.TLSConfigs, snapshot.EasySiteProfiles)
	antiDDoSSettings := antiddos.NormalizeSettings(snapshot.AntiDDoSSettings)
	tlsInputs, certInputs, tlsMaterialArtifacts, err := s.mapTLSInputs(snapshot.TLSConfigs, snapshot.CertificateMaterials)
	if err != nil {
		return nil, err
	}
	wafInputs := mapWAFInputs(snapshot.WAFPolicies)
	accessInputs := mapAccessInputs(snapshot.AccessPolicies)
	rateInputs := mapRateLimitInputs(snapshot.RateLimitPolicies)
	easyInputs := mapEasyInputs(snapshot.EasySiteProfiles)
	easyInputs = applyAntiDDoSDefaultEasyProfiles(siteInputs, easyInputs)
	rateInputs = applyAntiDDoSRateOverrides(siteInputs, rateInputs, antiDDoSSettings)

	siteArtifacts, err := pipeline.RenderSiteUpstreamArtifacts(siteInputs, upstreamInputs)
	if err != nil {
		return nil, err
	}
	tlsArtifacts, err := pipeline.RenderTLSArtifacts(siteInputs, tlsInputs, certInputs)
	if err != nil {
		return nil, err
	}
	wafArtifacts, err := pipeline.RenderWAFArtifacts(siteInputs, wafInputs)
	if err != nil {
		return nil, err
	}
	accessArtifacts, err := pipeline.RenderAccessRateLimitArtifacts(siteInputs, accessInputs, rateInputs)
	if err != nil {
		return nil, err
	}
	easyRateLimitArtifacts, err := pipeline.RenderEasyRateLimitArtifacts(siteInputs, upstreamInputs, easyInputs)
	if err != nil {
		return nil, err
	}
	easyArtifacts, err := pipeline.RenderEasyArtifacts(siteInputs, easyInputs)
	if err != nil {
		return nil, err
	}
	easyArtifacts, err = upsertAntiDDoSL4Artifact(easyArtifacts, antiDDoSSettings)
	if err != nil {
		return nil, err
	}
	easyArtifacts, err = upsertAdaptiveModelArtifact(easyArtifacts, antiDDoSSettings)
	if err != nil {
		return nil, err
	}

	return pipeline.AssembleRevisionBundle(
		pipeline.RevisionInput{
			ID:        revision.ID,
			Version:   revision.Version,
			CreatedAt: revision.CreatedAt,
		},
		siteArtifacts,
		tlsArtifacts,
		tlsMaterialArtifacts,
		wafArtifacts,
		accessArtifacts,
		easyRateLimitArtifacts,
		easyArtifacts,
	)
}

func applyAntiDDoSDefaultEasyProfiles(siteInputs []pipeline.SiteInput, items []pipeline.EasyProfileInput) []pipeline.EasyProfileInput {
	bySite := make(map[string]pipeline.EasyProfileInput, len(items))
	for _, item := range items {
		bySite[item.SiteID] = item
	}
	for _, site := range siteInputs {
		if !site.Enabled {
			continue
		}
		if _, ok := bySite[site.ID]; ok {
			continue
		}
		bySite[site.ID] = pipeline.EasyProfileInput{
			SiteID:                   site.ID,
			SecurityMode:             easysiteprofiles.SecurityModeBlock,
			AllowedMethods:           []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"},
			MaxClientSize:            "100m",
			UseModSecurity:           true,
			UseModSecurityCRSPlugins: true,
			UseLimitConn:             true,
			LimitConnMaxHTTP1:        200,
			UseLimitReq:              true,
			LimitReqRate:             "120r/s",
			CustomLimitRules:         nil,
			AuthBasicText:            "Restricted area",
			AntibotChallenge:         "no",
			AntibotURI:               "/challenge",
			PassHostHeader:           true,
			SendXForwardedFor:        true,
			SendXForwardedProto:      true,
			SendXRealIP:              false,
			ModSecurityCRSVersion:    "4",
			ModSecurityCustomPath:    "modsec/anomaly_score.conf",
		}
	}
	out := make([]pipeline.EasyProfileInput, 0, len(bySite))
	for _, item := range bySite {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SiteID < out[j].SiteID })
	return out
}

func applyAntiDDoSRateOverrides(siteInputs []pipeline.SiteInput, items []pipeline.RateLimitPolicyInput, settings antiddos.Settings) []pipeline.RateLimitPolicyInput {
	if !settings.EnforceL7Rate {
		return items
	}
	bySite := make(map[string]pipeline.RateLimitPolicyInput, len(items))
	for _, item := range items {
		bySite[item.SiteID] = item
	}
	for _, site := range siteInputs {
		if !site.Enabled {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(site.ID), "control-plane-access") {
			// Never enforce global anti-ddos L7 override on the management UI site.
			continue
		}
		bySite[site.ID] = pipeline.RateLimitPolicyInput{
			ID:            "antiddos-global-" + site.ID,
			SiteID:        site.ID,
			Enabled:       true,
			Requests:      settings.L7RequestsPS,
			WindowSeconds: 1,
			Burst:         settings.L7Burst,
			StatusCode:    settings.L7StatusCode,
		}
	}
	out := make([]pipeline.RateLimitPolicyInput, 0, len(bySite))
	for _, item := range bySite {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SiteID < out[j].SiteID })
	return out
}

func upsertAntiDDoSL4Artifact(items []pipeline.ArtifactOutput, settings antiddos.Settings) ([]pipeline.ArtifactOutput, error) {
	normalized := antiddos.NormalizeSettings(settings)
	if err := antiddos.ValidateSettings(normalized); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"enabled":         normalized.UseL4Guard,
		"chain_mode":      normalized.ChainMode,
		"conn_limit":      normalized.ConnLimit,
		"rate_per_second": normalized.RatePerSecond,
		"rate_burst":      normalized.RateBurst,
		"ports":           normalized.Ports,
		"target":          normalized.Target,
		"destination_ip":  normalized.DestinationIP,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	artifact := pipeline.NewArtifact("l4guard/config.json", pipeline.ArtifactKindNginxConfig, raw)
	out := make([]pipeline.ArtifactOutput, 0, len(items)+1)
	replaced := false
	for _, item := range items {
		if item.Path == "l4guard/config.json" {
			out = append(out, artifact)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, artifact)
	}
	return out, nil
}

func upsertAdaptiveModelArtifact(items []pipeline.ArtifactOutput, settings antiddos.Settings) ([]pipeline.ArtifactOutput, error) {
	normalized := antiddos.NormalizeSettings(settings)
	if err := antiddos.ValidateSettings(normalized); err != nil {
		return nil, err
	}
	payload := map[string]any{
		"model_enabled":                  normalized.ModelEnabled,
		"conn_limit":                     normalized.ConnLimit,
		"rate_per_second":                normalized.RatePerSecond,
		"rate_burst":                     normalized.RateBurst,
		"enforce_l7_rate_limit":          normalized.EnforceL7Rate,
		"l7_requests_per_second":         normalized.L7RequestsPS,
		"l7_burst":                       normalized.L7Burst,
		"model_poll_interval_seconds":    normalized.ModelPollIntervalSeconds,
		"model_decay_lambda":             normalized.ModelDecayLambda,
		"model_throttle_threshold":       normalized.ModelThrottleThreshold,
		"model_drop_threshold":           normalized.ModelDropThreshold,
		"model_hold_seconds":             normalized.ModelHoldSeconds,
		"model_throttle_rate_per_second": normalized.ModelThrottleRatePerSecond,
		"model_throttle_burst":           normalized.ModelThrottleBurst,
		"model_throttle_target":          normalized.ModelThrottleTarget,
		"model_weight_429":               normalized.ModelWeight429,
		"model_weight_403":               normalized.ModelWeight403,
		"model_weight_444":               normalized.ModelWeight444,
		"model_emergency_rps":            normalized.ModelEmergencyRPS,
		"model_emergency_unique_ips":     normalized.ModelEmergencyUniqueIPs,
		"model_emergency_per_ip_rps":     normalized.ModelEmergencyPerIPRPS,
		"model_weight_emergency_botnet":  normalized.ModelWeightEmergencyBotnet,
		"model_weight_emergency_single":  normalized.ModelWeightEmergencySingle,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	artifact := pipeline.NewArtifact("ddos-model/config.json", pipeline.ArtifactKindNginxConfig, raw)
	out := make([]pipeline.ArtifactOutput, 0, len(items)+1)
	replaced := false
	for _, item := range items {
		if item.Path == "ddos-model/config.json" {
			out = append(out, artifact)
			replaced = true
			continue
		}
		out = append(out, item)
	}
	if !replaced {
		out = append(out, artifact)
	}
	return out, nil
}

func (s *ApplyService) mapTLSInputs(configs []tlsconfigs.TLSConfig, materials []revisionsnapshots.CertificateMaterialSnapshot) ([]pipeline.TLSConfigInput, []pipeline.CertificateInput, []pipeline.ArtifactOutput, error) {
	sortedConfigs := append([]tlsconfigs.TLSConfig(nil), configs...)
	sort.Slice(sortedConfigs, func(i, j int) bool { return sortedConfigs[i].SiteID < sortedConfigs[j].SiteID })

	materialByCertificateID := make(map[string]revisionsnapshots.CertificateMaterialSnapshot, len(materials))
	for _, item := range materials {
		materialByCertificateID[item.CertificateID] = item
	}

	tlsInputs := make([]pipeline.TLSConfigInput, 0, len(sortedConfigs))
	certInputs := make([]pipeline.CertificateInput, 0, len(sortedConfigs))
	tlsMaterialArtifacts := make([]pipeline.ArtifactOutput, 0, len(sortedConfigs)*2)
	for _, config := range sortedConfigs {
		tlsInputs = append(tlsInputs, pipeline.TLSConfigInput{
			ID:                  config.SiteID + "-tls",
			SiteID:              config.SiteID,
			CertificateID:       config.CertificateID,
			RedirectHTTPToHTTPS: true,
		})

		material, ok := materialByCertificateID[config.CertificateID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("certificate material %s not found in revision snapshot", config.CertificateID)
		}

		certificatePEM, err := s.snapshots.ReadMaterial(material.CertificateRef)
		if err != nil {
			return nil, nil, nil, err
		}
		privateKeyPEM, err := s.snapshots.ReadMaterial(material.PrivateKeyRef)
		if err != nil {
			return nil, nil, nil, err
		}

		certInputs = append(certInputs, pipeline.CertificateInput{
			ID:            config.CertificateID,
			SiteID:        config.SiteID,
			StorageRef:    fmt.Sprintf("/etc/waf/tls/materials/%s/certificate.pem", config.CertificateID),
			PrivateKeyRef: fmt.Sprintf("/etc/waf/tls/materials/%s/private.key", config.CertificateID),
		})
		tlsMaterialArtifacts = append(tlsMaterialArtifacts,
			pipeline.NewArtifact(fmt.Sprintf("tls/materials/%s/certificate.pem", config.CertificateID), pipeline.ArtifactKindTLSRef, certificatePEM),
			pipeline.NewArtifact(fmt.Sprintf("tls/materials/%s/private.key", config.CertificateID), pipeline.ArtifactKindTLSRef, privateKeyPEM),
		)
	}

	return tlsInputs, certInputs, tlsMaterialArtifacts, nil
}

func mapSiteUpstreamInputs(siteItems []sites.Site, upstreamItems []upstreams.Upstream, tlsItems []tlsconfigs.TLSConfig, easyItems []easysiteprofiles.EasySiteProfile) ([]pipeline.SiteInput, []pipeline.UpstreamInput) {
	sortedSites := append([]sites.Site(nil), siteItems...)
	sort.Slice(sortedSites, func(i, j int) bool { return sortedSites[i].ID < sortedSites[j].ID })

	sortedUpstreams := append([]upstreams.Upstream(nil), upstreamItems...)
	sort.Slice(sortedUpstreams, func(i, j int) bool { return sortedUpstreams[i].ID < sortedUpstreams[j].ID })

	tlsSites := make(map[string]struct{}, len(tlsItems))
	for _, item := range tlsItems {
		tlsSites[item.SiteID] = struct{}{}
	}

	defaultUpstreamBySite := make(map[string]string)
	for _, item := range sortedUpstreams {
		if _, exists := defaultUpstreamBySite[item.SiteID]; !exists {
			defaultUpstreamBySite[item.SiteID] = item.ID
		}
	}
	hostHeaderEnabledBySite := make(map[string]bool, len(easyItems))
	for _, item := range easyItems {
		hostHeaderEnabledBySite[item.SiteID] = !item.UpstreamRouting.DisableHostHeader
	}

	siteInputs := make([]pipeline.SiteInput, 0, len(sortedSites))
	for _, item := range sortedSites {
		_, hasTLS := tlsSites[item.ID]
		siteInputs = append(siteInputs, pipeline.SiteInput{
			ID:                item.ID,
			Name:              item.ID,
			Enabled:           item.Enabled,
			PrimaryHost:       item.PrimaryHost,
			ListenHTTP:        !hasTLS,
			ListenHTTPS:       hasTLS,
			DefaultUpstreamID: defaultUpstreamBySite[item.ID],
		})
	}

	upstreamInputs := make([]pipeline.UpstreamInput, 0, len(sortedUpstreams))
	for _, item := range sortedUpstreams {
		passHostHeader := true
		if configured, ok := hostHeaderEnabledBySite[item.SiteID]; ok {
			passHostHeader = configured
		}
		upstreamInputs = append(upstreamInputs, pipeline.UpstreamInput{
			ID:             item.ID,
			SiteID:         item.SiteID,
			Name:           item.ID,
			Scheme:         item.Scheme,
			Host:           item.Host,
			Port:           item.Port,
			BasePath:       "/",
			PassHostHeader: passHostHeader,
		})
	}
	return siteInputs, upstreamInputs
}

func mapWAFInputs(items []wafpolicies.WAFPolicy) []pipeline.WAFPolicyInput {
	sorted := append([]wafpolicies.WAFPolicy(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out := make([]pipeline.WAFPolicyInput, 0, len(sorted))
	for _, item := range sorted {
		mode := pipeline.WAFModeDetection
		if string(item.Mode) == string(wafpolicies.ModePrevention) {
			mode = pipeline.WAFModePrevention
		}
		out = append(out, pipeline.WAFPolicyInput{
			ID:                 item.ID,
			SiteID:             item.SiteID,
			Enabled:            item.Enabled,
			Mode:               mode,
			CRSEnabled:         item.CRSEnabled,
			CustomRuleIncludes: append([]string(nil), item.CustomRuleIncludes...),
		})
	}
	return out
}

func mapAccessInputs(items []accesspolicies.AccessPolicy) []pipeline.AccessPolicyInput {
	sorted := append([]accesspolicies.AccessPolicy(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out := make([]pipeline.AccessPolicyInput, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, pipeline.AccessPolicyInput{
			ID:            item.ID,
			SiteID:        item.SiteID,
			DefaultAction: "allow",
			AllowCIDRs:    append([]string(nil), item.AllowList...),
			DenyCIDRs:     append([]string(nil), item.DenyList...),
		})
	}
	return out
}

func mapRateLimitInputs(items []ratelimitpolicies.RateLimitPolicy) []pipeline.RateLimitPolicyInput {
	sorted := append([]ratelimitpolicies.RateLimitPolicy(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out := make([]pipeline.RateLimitPolicyInput, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, pipeline.RateLimitPolicyInput{
			ID:            item.ID,
			SiteID:        item.SiteID,
			Enabled:       item.Enabled,
			Requests:      item.Limits.RequestsPerSecond,
			WindowSeconds: 1,
			Burst:         item.Limits.Burst,
			StatusCode:    429,
		})
	}
	return out
}

func mapCustomLimitRules(items []easysiteprofiles.CustomLimitRule) []pipeline.CustomRateLimitRuleInput {
	out := make([]pipeline.CustomRateLimitRuleInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.CustomRateLimitRuleInput{
			Path: item.Path,
			Rate: item.Rate,
		})
	}
	return out
}

func mapEasyInputs(items []easysiteprofiles.EasySiteProfile) []pipeline.EasyProfileInput {
	sorted := append([]easysiteprofiles.EasySiteProfile(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SiteID < sorted[j].SiteID })
	out := make([]pipeline.EasyProfileInput, 0, len(sorted))
	for _, item := range sorted {
		out = append(out, pipeline.EasyProfileInput{
			SiteID:       item.SiteID,
			SecurityMode: item.FrontService.SecurityMode,

			AllowedMethods: item.HTTPBehavior.AllowedMethods,
			MaxClientSize:  item.HTTPBehavior.MaxClientSize,

			ReferrerPolicy:        item.HTTPHeaders.ReferrerPolicy,
			ContentSecurityPolicy: item.HTTPHeaders.ContentSecurityPolicy,
			PermissionsPolicy:     item.HTTPHeaders.PermissionsPolicy,
			UseCORS:               item.HTTPHeaders.UseCORS,
			CORSAllowedOrigins:    item.HTTPHeaders.CORSAllowedOrigins,

			ReverseProxyCustomHost: item.UpstreamRouting.ReverseProxyCustomHost,
			ReverseProxySSLSNI:     item.UpstreamRouting.ReverseProxySSLSNI,
			ReverseProxySSLSNIName: item.UpstreamRouting.ReverseProxySSLSNIName,
			ReverseProxyWebsocket:  item.UpstreamRouting.ReverseProxyWebsocket,
			ReverseProxyKeepalive:  item.UpstreamRouting.ReverseProxyKeepalive,
			PassHostHeader:         !item.UpstreamRouting.DisableHostHeader,
			SendXForwardedFor:      !item.UpstreamRouting.DisableXForwardedFor,
			SendXForwardedProto:    !item.UpstreamRouting.DisableXForwardedProto,
			SendXRealIP:            item.UpstreamRouting.EnableXRealIP,

			UseAuthBasic:      item.SecurityAuthBasic.UseAuthBasic,
			AuthBasicUser:     item.SecurityAuthBasic.AuthBasicUser,
			AuthBasicPassword: item.SecurityAuthBasic.AuthBasicPassword,
			AuthBasicText:     item.SecurityAuthBasic.AuthBasicText,

			AntibotChallenge:      item.SecurityAntibot.AntibotChallenge,
			AntibotURI:            item.SecurityAntibot.AntibotURI,
			AntibotRecaptchaScore: item.SecurityAntibot.AntibotRecaptchaScore,
			AntibotRecaptchaKey:   item.SecurityAntibot.AntibotRecaptchaSitekey,
			AntibotHcaptchaKey:    item.SecurityAntibot.AntibotHcaptchaSitekey,
			AntibotTurnstileKey:   item.SecurityAntibot.AntibotTurnstileSitekey,

			UseLimitConn:              item.SecurityBehaviorAndLimits.UseLimitConn,
			LimitConnMaxHTTP1:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP1,
			LimitConnMaxHTTP2:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP2,
			LimitConnMaxHTTP3:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP3,
			UseLimitReq:               item.SecurityBehaviorAndLimits.UseLimitReq,
			LimitReqRate:              item.SecurityBehaviorAndLimits.LimitReqRate,
			CustomLimitRules:          mapCustomLimitRules(item.SecurityBehaviorAndLimits.CustomLimitRules),
			UseBadBehavior:            item.SecurityBehaviorAndLimits.UseBadBehavior,
			BadBehaviorStatusCodes:    append([]int(nil), item.SecurityBehaviorAndLimits.BadBehaviorStatusCodes...),
			BadBehaviorBanTimeSeconds: item.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds,

			BlacklistIP:        item.SecurityBehaviorAndLimits.BlacklistIP,
			BlacklistUserAgent: item.SecurityBehaviorAndLimits.BlacklistUserAgent,
			BlacklistURI:       item.SecurityBehaviorAndLimits.BlacklistURI,

			BlacklistCountry: item.SecurityCountryPolicy.BlacklistCountry,
			WhitelistCountry: item.SecurityCountryPolicy.WhitelistCountry,

			UseModSecurity:                    item.SecurityModSecurity.UseModSecurity,
			UseModSecurityCRSPlugins:          item.SecurityModSecurity.UseModSecurityCRSPlugins,
			UseModSecurityCustomConfiguration: item.SecurityModSecurity.UseCustomConfiguration,
			ModSecurityCRSVersion:             item.SecurityModSecurity.ModSecurityCRSVersion,
			ModSecurityCRSPlugins:             item.SecurityModSecurity.ModSecurityCRSPlugins,
			ModSecurityCustomPath:             item.SecurityModSecurity.CustomConfiguration.Path,
			ModSecurityCustomContent:          item.SecurityModSecurity.CustomConfiguration.Content,
		})
	}
	return out
}

type OSCommandExecutor struct{}

func (OSCommandExecutor) Run(name string, args []string, workdir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workdir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// NoopCommandExecutor is used when syntax checks are delegated to runtime-side reload.
// This keeps apply functional in deployments where the control-plane container does not
// include nginx binaries.
type NoopCommandExecutor struct{}

func (NoopCommandExecutor) Run(name string, args []string, workdir string) error {
	return nil
}

type HTTPHealthChecker struct {
	URL string
}

func (h HTTPHealthChecker) Check(active *pipeline.ActivePointer) error {
	url := strings.TrimSpace(h.URL)
	if url == "" {
		url = "http://127.0.0.1:8081/readyz"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %d", resp.StatusCode)
	}
	return nil
}
