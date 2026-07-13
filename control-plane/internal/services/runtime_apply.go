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
	"waf/control-plane/internal/telemetry"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/virtualpatches"
	"waf/control-plane/internal/wafpolicies"
)

type revisionStoreForApply interface {
	Get(revisionID string) (revisions.Revision, bool, error)
	MarkActive(revisionID string) error
	MarkFailed(revisionID string) error
	RecordApplyResult(revisionID string, jobID string, status string, result string, appliedAt string) error
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
	coord          DistributedCoordinator
	governance     interface {
		EnsureRevisionCanApply(revision revisions.Revision) error
	}
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
		coord:          NewNoopDistributedCoordinator(),
	}
}

func (s *ApplyService) Apply(ctx context.Context, revisionID string) (job jobs.Job, err error) {
	if s.coord == nil {
		s.coord = NewNoopDistributedCoordinator()
	}
	err = s.coord.WithLock(ctx, "ha:revision:apply", s.coord.OperationTTL(), func(lockCtx context.Context) error {
		job, err = s.applyUnlocked(lockCtx, revisionID)
		return err
	})
	if err != nil {
		telemetry.Default().RecordRevisionApply(s.coord.NodeID(), "failed")
	} else {
		telemetry.Default().RecordRevisionApply(s.coord.NodeID(), strings.ToLower(string(job.Status)))
	}
	return job, err
}

func (s *ApplyService) SetCoordinator(coord DistributedCoordinator) {
	if coord == nil {
		s.coord = NewNoopDistributedCoordinator()
		return
	}
	s.coord = coord
}

func (s *ApplyService) SetGovernance(governance interface {
	EnsureRevisionCanApply(revision revisions.Revision) error
}) {
	s.governance = governance
}

func (s *ApplyService) applyUnlocked(ctx context.Context, revisionID string) (job jobs.Job, err error) {
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
	if s.governance != nil {
		if err := s.governance.EnsureRevisionCanApply(revision); err != nil {
			return jobs.Job{}, err
		}
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
		failedJob, markErr := s.jobs.MarkFailed(job.ID, err.Error())
		if markErr != nil {
			return jobs.Job{}, markErr
		}
		if recordErr := s.revisions.RecordApplyResult(revisionID, failedJob.ID, string(failedJob.Status), failedJob.Result, coalesceJobTime(failedJob)); recordErr != nil {
			return failedJob, recordErr
		}
		return failedJob, nil
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
		failedJob, markErr := s.jobs.MarkFailed(job.ID, err.Error())
		if markErr != nil {
			return jobs.Job{}, markErr
		}
		if recordErr := s.revisions.RecordApplyResult(revisionID, failedJob.ID, string(failedJob.Status), failedJob.Result, coalesceJobTime(failedJob)); recordErr != nil {
			return failedJob, recordErr
		}
		return failedJob, nil
	}
	s.emitEvent(events.Event{
		Type:              events.TypeApplySucceeded,
		Severity:          events.SeverityInfo,
		SourceComponent:   "apply-runner",
		Summary:           "apply succeeded",
		RelatedRevisionID: revisionID,
		RelatedJobID:      job.ID,
	})
	succeededJob, markErr := s.jobs.MarkSucceeded(job.ID, "revision applied")
	if markErr != nil {
		return jobs.Job{}, markErr
	}
	if recordErr := s.revisions.RecordApplyResult(revisionID, succeededJob.ID, string(succeededJob.Status), succeededJob.Result, coalesceJobTime(succeededJob)); recordErr != nil {
		return succeededJob, recordErr
	}
	return succeededJob, nil
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
	if err := validateManagementSafeguard(bundle, snapshot); err != nil {
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
	siteInputs, upstreamInputs := mapSiteUpstreamInputs(snapshot.Sites, snapshot.Upstreams, snapshot.TLSConfigs, snapshot.EasySiteProfiles, snapshot.ManagementHosts, snapshot.ManagementHostsConfigured)
	antiDDoSSettings := antiddos.NormalizeSettings(snapshot.AntiDDoSSettings)
	tlsInputs, certInputs, tlsMaterialArtifacts, err := s.mapTLSInputs(snapshot.TLSConfigs, snapshot.CertificateMaterials, snapshot.EasySiteProfiles)
	if err != nil {
		return nil, err
	}
	wafInputs := mapWAFInputs(snapshot.WAFPolicies)
	accessInputs := mapAccessInputs(snapshot.AccessPolicies, snapshot.EasySiteProfiles)
	rateInputs := mapRateLimitInputs(snapshot.RateLimitPolicies)
	easyInputs := mapEasyInputs(snapshot.EasySiteProfiles, snapshot.VirtualPatches)
	easyInputs = applyAntiDDoSDefaultEasyProfiles(siteInputs, easyInputs)
	siteInputs = applyEasySiteInputFlags(siteInputs, easyInputs)
	rateInputs = applyAntiDDoSRateOverrides(siteInputs, rateInputs, antiDDoSSettings)
	rateInputs = applyEasyLimitReqURLOverrides(rateInputs, easyInputs)

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
	easyArtifacts, err = upsertAdaptiveModelArtifact(easyArtifacts, antiDDoSSettings, snapshot.EasySiteProfiles)
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

func applyEasySiteInputFlags(siteInputs []pipeline.SiteInput, easyInputs []pipeline.EasyProfileInput) []pipeline.SiteInput {
	if len(easyInputs) == 0 {
		return siteInputs
	}
	easyBySite := make(map[string]pipeline.EasyProfileInput, len(easyInputs))
	for _, item := range easyInputs {
		easyBySite[item.SiteID] = item
	}
	out := make([]pipeline.SiteInput, len(siteInputs))
	copy(out, siteInputs)
	for i := range out {
		if easy, ok := easyBySite[out[i].ID]; ok {
			out[i].UseEasyConfig = true
			out[i].UseCustomErrorPages = easy.UseCustomErrorPages
		}
	}
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
		if _, exists := bySite[site.ID]; exists {
			continue
		}
		if isManagementSiteID(strings.TrimSpace(site.ID)) {
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

// applyEasyLimitReqURLOverrides copies LimitReqURL from easy-profile into the
// matching RateLimitPolicyInput so the nginx ratelimits/site.conf snippet
// applies limit_req only to the configured URI instead of every request.
func applyEasyLimitReqURLOverrides(items []pipeline.RateLimitPolicyInput, easyInputs []pipeline.EasyProfileInput) []pipeline.RateLimitPolicyInput {
	urlBySite := make(map[string]string, len(easyInputs))
	for _, e := range easyInputs {
		if e.UseLimitReq && strings.TrimSpace(e.LimitReqURL) != "" {
			urlBySite[e.SiteID] = strings.TrimSpace(e.LimitReqURL)
		}
	}
	if len(urlBySite) == 0 {
		return items
	}
	out := make([]pipeline.RateLimitPolicyInput, len(items))
	copy(out, items)
	for i, item := range out {
		if url, ok := urlBySite[item.SiteID]; ok {
			out[i].LimitReqURL = url
		}
	}
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

func upsertAdaptiveModelArtifact(items []pipeline.ArtifactOutput, settings antiddos.Settings, profiles []easysiteprofiles.EasySiteProfile) ([]pipeline.ArtifactOutput, error) {
	normalized := antiddos.NormalizeSettings(settings)
	if err := antiddos.ValidateSettings(normalized); err != nil {
		return nil, err
	}
	modelEnabled, enabledSites := adaptiveModelScope(normalized.ModelEnabled, profiles)
	ja3Fingerprints := aggregateJA3Blacklist(profiles)
	payload := map[string]any{
		"model_enabled":                  modelEnabled,
		"model_enabled_sites":            enabledSites,
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
	if len(ja3Fingerprints) > 0 {
		payload["ja3_blacklist_fingerprints"] = ja3Fingerprints
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

// aggregateJA3Blacklist collects all JA3 fingerprints from all site profiles
// into a deduplicated sorted slice. The result is written to ddos-model/config.json
// so sentinel can activate signal_ja3_risk only when the operator has explicitly
// configured JA3 blacklists on at least one site (opt-in behaviour).
func aggregateJA3Blacklist(profiles []easysiteprofiles.EasySiteProfile) []string {
	seen := map[string]struct{}{}
	for _, p := range profiles {
		for _, fp := range p.SecurityBehaviorAndLimits.BlacklistJA3 {
			fp = strings.ToLower(strings.TrimSpace(fp))
			if fp != "" {
				seen[fp] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for fp := range seen {
		out = append(out, fp)
	}
	sort.Strings(out)
	return out
}

func adaptiveModelScope(globalEnabled bool, profiles []easysiteprofiles.EasySiteProfile) (bool, []string) {
	if !globalEnabled {
		return false, nil
	}
	if len(profiles) == 0 {
		return true, nil
	}
	seen := map[string]struct{}{}
	for _, profile := range profiles {
		if !profile.FrontService.AdaptiveModelEnabled {
			continue
		}
		for _, value := range []string{profile.SiteID, profile.FrontService.ServerName} {
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "" {
				continue
			}
			seen[value] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return false, nil
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return true, out
}

func (s *ApplyService) mapTLSInputs(configs []tlsconfigs.TLSConfig, materials []revisionsnapshots.CertificateMaterialSnapshot, profiles []easysiteprofiles.EasySiteProfile) ([]pipeline.TLSConfigInput, []pipeline.CertificateInput, []pipeline.ArtifactOutput, error) {
	sortedConfigs := append([]tlsconfigs.TLSConfig(nil), configs...)
	sort.Slice(sortedConfigs, func(i, j int) bool { return sortedConfigs[i].SiteID < sortedConfigs[j].SiteID })

	materialByCertificateID := make(map[string]revisionsnapshots.CertificateMaterialSnapshot, len(materials))
	for _, item := range materials {
		materialByCertificateID[item.CertificateID] = item
	}

	tlsInputs := make([]pipeline.TLSConfigInput, 0, len(sortedConfigs))
	certInputs := make([]pipeline.CertificateInput, 0, len(sortedConfigs))
	tlsMaterialArtifacts := make([]pipeline.ArtifactOutput, 0, len(sortedConfigs)*2)
	seenMaterialArtifacts := map[string]struct{}{}
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
		seenMaterialArtifacts[config.CertificateID] = struct{}{}
		tlsMaterialArtifacts = append(tlsMaterialArtifacts,
			pipeline.NewArtifact(fmt.Sprintf("tls/materials/%s/certificate.pem", config.CertificateID), pipeline.ArtifactKindTLSRef, certificatePEM),
			pipeline.NewArtifact(fmt.Sprintf("tls/materials/%s/private.key", config.CertificateID), pipeline.ArtifactKindTLSRef, privateKeyPEM),
		)
	}

	for _, profile := range profiles {
		if !profile.FrontService.MTLSEnabled {
			continue
		}
		certificateID := certificateMaterialIDFromRef(profile.FrontService.MTLSClientCARef)
		if certificateID == "" {
			continue
		}
		if _, ok := seenMaterialArtifacts[certificateID]; ok {
			continue
		}
		material, ok := materialByCertificateID[certificateID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("mTLS CA material %s not found in revision snapshot", certificateID)
		}
		certificatePEM, err := s.snapshots.ReadMaterial(material.CertificateRef)
		if err != nil {
			return nil, nil, nil, err
		}
		seenMaterialArtifacts[certificateID] = struct{}{}
		tlsMaterialArtifacts = append(tlsMaterialArtifacts,
			pipeline.NewArtifact(fmt.Sprintf("tls/materials/%s/certificate.pem", certificateID), pipeline.ArtifactKindTLSRef, certificatePEM),
		)
	}

	return tlsInputs, certInputs, tlsMaterialArtifacts, nil
}

func certificateMaterialIDFromRef(ref string) string {
	ref = strings.TrimSpace(strings.ReplaceAll(ref, "\\", "/"))
	if ref == "" {
		return ""
	}
	parts := strings.Split(ref, "/")
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "files" && parts[i+2] == "certificate.pem" {
			return parts[i+1]
		}
	}
	return ""
}

func mapSiteUpstreamInputs(siteItems []sites.Site, upstreamItems []upstreams.Upstream, tlsItems []tlsconfigs.TLSConfig, easyItems []easysiteprofiles.EasySiteProfile, managementHosts []string, managementConfigured bool) ([]pipeline.SiteInput, []pipeline.UpstreamInput) {
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
	mtlsBySite := make(map[string]pipeline.MTLSInput, len(easyItems))
	for _, item := range easyItems {
		hostHeaderEnabledBySite[item.SiteID] = !item.UpstreamRouting.DisableHostHeader
		mtlsBySite[item.SiteID] = pipeline.MTLSInput{
			MTLSEnabled:     item.FrontService.MTLSEnabled,
			MTLSOptional:    item.FrontService.MTLSOptional,
			MTLSVerifyDepth: item.FrontService.MTLSVerifyDepth,
			MTLSClientCARef: runtimeCertificateMaterialPath(item.FrontService.MTLSClientCARef),
			MTLSPassHeaders: item.FrontService.MTLSPassHeaders,
		}
	}

	easySites := make(map[string]struct{}, len(easyItems))
	customErrorPagesBySite := make(map[string]bool, len(easyItems))
	managementHostSet := make(map[string]struct{}, len(managementHosts))
	for _, host := range managementHosts {
		managementHostSet[strings.ToLower(strings.TrimSpace(host))] = struct{}{}
	}
	for _, item := range easyItems {
		easySites[item.SiteID] = struct{}{}
		customErrorPagesBySite[item.SiteID] = item.UseCustomErrorPages
	}

	siteInputs := make([]pipeline.SiteInput, 0, len(sortedSites))
	for _, item := range sortedSites {
		_, hasTLS := tlsSites[item.ID]
		_, hasEasy := easySites[item.ID]
		mtls := mtlsBySite[item.ID]
		_, management := managementHostSet[strings.ToLower(strings.TrimSpace(item.PrimaryHost))]
		siteInputs = append(siteInputs, pipeline.SiteInput{
			ID:                   item.ID,
			Name:                 item.ID,
			Enabled:              item.Enabled,
			PrimaryHost:          item.PrimaryHost,
			ListenHTTP:           !hasTLS,
			ListenHTTPS:          hasTLS,
			DefaultUpstreamID:    defaultUpstreamBySite[item.ID],
			UseEasyConfig:        hasEasy,
			UseCustomErrorPages:  customErrorPagesBySite[item.ID],
			MTLS:                 mtls,
			Management:           management,
			ManagementConfigured: managementConfigured,
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

func runtimeCertificateMaterialPath(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "/") {
		return ref
	}
	if certificateID := certificateMaterialIDFromRef(ref); certificateID != "" {
		return "/etc/waf/tls/materials/" + certificateID + "/certificate.pem"
	}
	return "/var/lib/waf/control-plane/" + strings.TrimLeft(ref, "/")
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

func mapAccessInputs(items []accesspolicies.AccessPolicy, profiles []easysiteprofiles.EasySiteProfile) []pipeline.AccessPolicyInput {
	// build site→security_mode lookup from easy profiles
	modeBysite := make(map[string]string, len(profiles))
	for _, p := range profiles {
		modeBysite[p.SiteID] = strings.ToLower(strings.TrimSpace(p.FrontService.SecurityMode))
	}
	sorted := append([]accesspolicies.AccessPolicy(nil), items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out := make([]pipeline.AccessPolicyInput, 0, len(sorted))
	for _, item := range sorted {
		defaultAction := "allow"
		if len(item.AllowList) > 0 {
			defaultAction = "deny"
		}
		out = append(out, pipeline.AccessPolicyInput{
			ID:                item.ID,
			SiteID:            item.SiteID,
			DefaultAction:     defaultAction,
			AllowCIDRs:        append([]string(nil), item.AllowList...),
			DenyCIDRs:         append([]string(nil), item.DenyList...),
			TrustedProxyCIDRs: append([]string(nil), item.TrustedProxyCIDRs...),
			SecurityMode:      modeBysite[item.SiteID],
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

func mapAPIPositiveEndpointPolicies(items []easysiteprofiles.APIPositiveEndpointPolicy) []pipeline.APIPositiveEndpointPolicyInput {
	out := make([]pipeline.APIPositiveEndpointPolicyInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.APIPositiveEndpointPolicyInput{
			Path:         item.Path,
			Methods:      append([]string(nil), item.Methods...),
			TokenIDs:     append([]string(nil), item.TokenIDs...),
			ContentTypes: append([]string(nil), item.ContentTypes...),
			Mode:         item.Mode,
		})
	}
	return out
}

func mapAntibotChallengeRules(items []easysiteprofiles.AntibotChallengeRule) []pipeline.AntibotChallengeRuleInput {
	out := make([]pipeline.AntibotChallengeRuleInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.AntibotChallengeRuleInput{
			Path:      item.Path,
			Challenge: item.Challenge,
		})
	}
	return out
}

func mapAntibotExclusionRules(items []easysiteprofiles.AntibotExclusionRule) []pipeline.AntibotExclusionRuleInput {
	out := make([]pipeline.AntibotExclusionRuleInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.AntibotExclusionRuleInput{
			Path:    item.Path,
			Methods: append([]string(nil), item.Methods...),
		})
	}
	return out
}

func mapAuthUsers(items []easysiteprofiles.SecurityAuthUser) []pipeline.ServiceAuthUserInput {
	out := make([]pipeline.ServiceAuthUserInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.ServiceAuthUserInput{
			Username:    item.Username,
			Password:    item.Password,
			Enabled:     item.Enabled,
			LastLoginAt: item.LastLoginAt,
		})
	}
	return out
}

func mapAuthExclusionRules(items []easysiteprofiles.SecurityAuthExclusionRule) []pipeline.AuthExclusionRuleInput {
	out := make([]pipeline.AuthExclusionRuleInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.AuthExclusionRuleInput{
			Path:    item.Path,
			Methods: append([]string(nil), item.Methods...),
		})
	}
	return out
}

func mapAuthServiceTokens(items []easysiteprofiles.SecurityAuthServiceToken) []pipeline.ServiceAuthTokenInput {
	out := make([]pipeline.ServiceAuthTokenInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.ServiceAuthTokenInput{
			ServiceName: item.ServiceName,
			Token:       item.Token,
			Enabled:     item.Enabled,
			LastUsedAt:  item.LastUsedAt,
		})
	}
	return out
}

func mapEasyInputs(items []easysiteprofiles.EasySiteProfile, virtualPatches []virtualpatches.VirtualPatch) []pipeline.EasyProfileInput {
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
			HSTSEnabled:           item.HTTPHeaders.HSTSEnabled,
			HSTSMaxAgeSeconds:     item.HTTPHeaders.HSTSMaxAgeSeconds,
			HSTSIncludeSubdomains: item.HTTPHeaders.HSTSIncludeSubdomains,
			HSTSPreload:           item.HTTPHeaders.HSTSPreload,
			UseCORS:               item.HTTPHeaders.UseCORS,
			CORSAllowedOrigins:    item.HTTPHeaders.CORSAllowedOrigins,
			CookieFlags:           item.HTTPHeaders.CookieFlags,
			KeepUpstreamHeaders:   item.HTTPHeaders.KeepUpstreamHeaders,

			ReverseProxyCustomHost: item.UpstreamRouting.ReverseProxyCustomHost,
			ReverseProxySSLSNI:     item.UpstreamRouting.ReverseProxySSLSNI,
			ReverseProxySSLSNIName: item.UpstreamRouting.ReverseProxySSLSNIName,
			ReverseProxyWebsocket:  item.UpstreamRouting.ReverseProxyWebsocket,
			ReverseProxyKeepalive:  item.UpstreamRouting.ReverseProxyKeepalive,
			PassHostHeader:         !item.UpstreamRouting.DisableHostHeader,
			SendXForwardedFor:      !item.UpstreamRouting.DisableXForwardedFor,
			SendXForwardedProto:    !item.UpstreamRouting.DisableXForwardedProto,
			SendXRealIP:            item.UpstreamRouting.EnableXRealIP,

			HealthCheckEnabled:         item.UpstreamRouting.HealthCheckEnabled,
			HealthCheckPath:            item.UpstreamRouting.HealthCheckPath,
			HealthCheckIntervalSeconds: item.UpstreamRouting.HealthCheckIntervalSeconds,
			HealthCheckFailThreshold:   item.UpstreamRouting.HealthCheckFailThreshold,

			UseAuthBasic:               item.SecurityAuthBasic.UseAuthBasic,
			AuthMode:                   item.SecurityAuthBasic.AuthMode,
			AuthOrder:                  item.SecurityAuthBasic.AuthOrder,
			AuthBasicUser:              item.SecurityAuthBasic.AuthBasicUser,
			AuthBasicPassword:          item.SecurityAuthBasic.AuthBasicPassword,
			AuthBasicText:              item.SecurityAuthBasic.AuthBasicText,
			AuthUsers:                  mapAuthUsers(item.SecurityAuthBasic.Users),
			AuthServiceTokens:          mapAuthServiceTokens(item.SecurityAuthBasic.ServiceTokens),
			AuthExclusionRules:         mapAuthExclusionRules(item.SecurityAuthBasic.ExclusionRules),
			AuthSessionTTLMin:          item.SecurityAuthBasic.SessionInactivityMinutes,
			AntibotChallenge:           item.SecurityAntibot.AntibotChallenge,
			AntibotChallengeTemplate:   item.SecurityAntibot.AntibotChallengeTemplate,
			AntibotURI:                 item.SecurityAntibot.AntibotURI,
			AntibotScannerAutoBan:      item.SecurityAntibot.ScannerAutoBanEnabled,
			AntibotRecaptchaScore:      item.SecurityAntibot.AntibotRecaptchaScore,
			AntibotRecaptchaKey:        item.SecurityAntibot.AntibotRecaptchaSitekey,
			AntibotHcaptchaKey:         item.SecurityAntibot.AntibotHcaptchaSitekey,
			AntibotTurnstileKey:        item.SecurityAntibot.AntibotTurnstileSitekey,
			AntibotExclusionRules:      mapAntibotExclusionRules(item.SecurityAntibot.ExclusionRules),
			ChallengeEscalationEnabled: item.SecurityAntibot.ChallengeEscalationEnabled,
			ChallengeEscalationMode:    item.SecurityAntibot.ChallengeEscalationMode,
			AntibotChallengeRules:      mapAntibotChallengeRules(item.SecurityAntibot.ChallengeRules),

			UseLimitConn:              item.SecurityBehaviorAndLimits.UseLimitConn,
			LimitConnMaxHTTP1:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP1,
			LimitConnMaxHTTP2:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP2,
			LimitConnMaxHTTP3:         item.SecurityBehaviorAndLimits.LimitConnMaxHTTP3,
			UseLimitReq:               item.SecurityBehaviorAndLimits.UseLimitReq,
			LimitReqURL:               item.SecurityBehaviorAndLimits.LimitReqURL,
			LimitReqRate:              item.SecurityBehaviorAndLimits.LimitReqRate,
			CustomLimitRules:          mapCustomLimitRules(item.SecurityBehaviorAndLimits.CustomLimitRules),
			UseBadBehavior:            item.SecurityBehaviorAndLimits.UseBadBehavior,
			BadBehaviorStatusCodes:    append([]int(nil), item.SecurityBehaviorAndLimits.BadBehaviorStatusCodes...),
			BadBehaviorBanTimeSeconds: item.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds,

			BlacklistIP:        item.SecurityBehaviorAndLimits.BlacklistIP,
			BlacklistUserAgent: item.SecurityBehaviorAndLimits.BlacklistUserAgent,
			BlacklistURI:       item.SecurityBehaviorAndLimits.BlacklistURI,

			ExceptionsURI: append([]string(nil), item.SecurityBehaviorAndLimits.ExceptionsURI...),

			BlacklistCountry: item.SecurityCountryPolicy.BlacklistCountry,
			WhitelistCountry: item.SecurityCountryPolicy.WhitelistCountry,
			ShowGeoBlockPage: item.SecurityCountryPolicy.ShowGeoBlockPage,
			GeoTimeWindows:   mapGeoTimeWindows(item.SecurityCountryPolicy.GeoTimeWindows),

			WSInspection: pipeline.WSInspectionInput{
				UseWSInspection:   item.SecurityWebSocket.UseWSInspection,
				WSBlockPatterns:   append([]string(nil), item.SecurityWebSocket.WSBlockPatterns...),
				WSMaxMessageBytes: item.SecurityWebSocket.WSMaxMessageBytes,
				WSRateMsgPerSec:   item.SecurityWebSocket.WSRateMsgPerSec,
			},

			MTLS: pipeline.MTLSInput{
				MTLSEnabled:     item.FrontService.MTLSEnabled,
				MTLSOptional:    item.FrontService.MTLSOptional,
				MTLSVerifyDepth: item.FrontService.MTLSVerifyDepth,
				MTLSClientCARef: item.FrontService.MTLSClientCARef,
				MTLSPassHeaders: item.FrontService.MTLSPassHeaders,
			},

			UpstreamMTLS: pipeline.UpstreamMTLSInput{
				UpstreamMTLSEnabled: item.UpstreamRouting.UpstreamMTLSEnabled,
				UpstreamMTLSCertRef: item.UpstreamRouting.UpstreamMTLSCertRef,
				UpstreamMTLSKeyRef:  item.UpstreamRouting.UpstreamMTLSKeyRef,
				UpstreamMTLSCARef:   item.UpstreamRouting.UpstreamMTLSCARef,
			},

			UseModSecurity:                    item.SecurityModSecurity.UseModSecurity,
			UseModSecurityCRSPlugins:          item.SecurityModSecurity.UseModSecurityCRSPlugins,
			UseModSecurityCustomConfiguration: item.SecurityModSecurity.UseCustomConfiguration,
			ModSecurityCRSVersion:             item.SecurityModSecurity.ModSecurityCRSVersion,
			ModSecurityCRSPlugins:             item.SecurityModSecurity.ModSecurityCRSPlugins,
			ModSecurityExclusionRules:         mapModSecurityExclusionRules(item.SecurityModSecurity.ExclusionRules),
			ModSecurityCustomPath:             item.SecurityModSecurity.CustomConfiguration.Path,
			ModSecurityCustomContent:          item.SecurityModSecurity.CustomConfiguration.Content,

			UseAPIPositiveSecurity: item.SecurityAPIPositive.UseAPIPositiveSecurity,
			OpenAPISchemaRef:       item.SecurityAPIPositive.OpenAPISchemaRef,
			APIEnforcementMode:     item.SecurityAPIPositive.EnforcementMode,
			APIDefaultAction:       item.SecurityAPIPositive.DefaultAction,
			APIEndpointPolicies:    mapAPIPositiveEndpointPolicies(item.SecurityAPIPositive.EndpointPolicies),

			HttpStrictParsing: item.HTTPBehavior.HttpStrictParsing,

			VirtualPatches: append(mapProfileVirtualPatches(item.VirtualPatches), mapVirtualPatches(virtualPatches, item.SiteID)...),

			UseCustomErrorPages: item.UseCustomErrorPages,
			DisabledErrorPages:  item.DisabledErrorPages,
		})
	}
	return out
}

func mapModSecurityExclusionRules(items []easysiteprofiles.ModSecurityExclusionRule) []pipeline.ModSecurityExclusionRuleInput {
	out := make([]pipeline.ModSecurityExclusionRuleInput, 0, len(items))
	for _, item := range items {
		out = append(out, pipeline.ModSecurityExclusionRuleInput{
			Path: item.Path, PathPattern: item.PathPattern, Methods: append([]string(nil), item.Methods...),
			Mode: item.MatchMode, RuleIDs: append([]int(nil), item.RuleIDs...), Targets: append([]string(nil), item.Targets...), Comment: item.Comment,
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
	URL   string
	Token string
}

func (h HTTPHealthChecker) Check(active *pipeline.ActivePointer) error {
	candidates := runtimeEndpointCandidates(strings.TrimSpace(h.URL), "http://127.0.0.1:8081/readyz")
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error
	for _, candidate := range candidates {
		req, err := http.NewRequest(http.MethodGet, candidate, nil)
		if err != nil {
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, strings.TrimSpace(h.Token))
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				lastErr = nil
				return
			}
			lastErr = fmt.Errorf("health endpoint returned %d", resp.StatusCode)
		}()
		if lastErr == nil {
			return nil
		}
		return lastErr
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("runtime health endpoint is not reachable")
}

// mapVirtualPatches filters virtual patches by siteID and converts to pipeline input.
func mapVirtualPatches(patches []virtualpatches.VirtualPatch, siteID string) []pipeline.VirtualPatchInput {
	out := make([]pipeline.VirtualPatchInput, 0)
	for _, p := range patches {
		if p.SiteID != siteID {
			continue
		}
		out = append(out, pipeline.VirtualPatchInput{
			ID:      p.ID,
			Pattern: p.Pattern,
			Target:  p.Target,
			Action:  p.Action,
		})
	}
	return out
}

func mapProfileVirtualPatches(patches []easysiteprofiles.VirtualPatchSettings) []pipeline.VirtualPatchInput {
	out := make([]pipeline.VirtualPatchInput, 0, len(patches))
	for _, p := range patches {
		out = append(out, pipeline.VirtualPatchInput{
			ID:      p.ID,
			Pattern: p.Pattern,
			Target:  p.Target,
			Action:  p.Action,
		})
	}
	return out
}

// mapGeoTimeWindows converts easysiteprofiles.GeoTimeWindow to pipeline.GeoTimeWindowInput.
func mapGeoTimeWindows(windows []easysiteprofiles.GeoTimeWindow) []pipeline.GeoTimeWindowInput {
	if len(windows) == 0 {
		return nil
	}
	out := make([]pipeline.GeoTimeWindowInput, 0, len(windows))
	for _, w := range windows {
		out = append(out, pipeline.GeoTimeWindowInput{
			Countries:  append([]string(nil), w.Countries...),
			Action:     w.Action,
			DaysOfWeek: append([]int(nil), w.DaysOfWeek...),
			HoursStart: w.HoursStart,
			HoursEnd:   w.HoursEnd,
		})
	}
	return out
}
