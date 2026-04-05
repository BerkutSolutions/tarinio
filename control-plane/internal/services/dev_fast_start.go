package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/config"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
)

type devFastStartSiteService interface {
	List() ([]sites.Site, error)
	Create(ctx context.Context, site sites.Site) (sites.Site, error)
	Update(ctx context.Context, site sites.Site) (sites.Site, error)
}

type devFastStartUpstreamService interface {
	List() ([]upstreams.Upstream, error)
	Create(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error)
	Update(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error)
}

type devFastStartCertificateService interface {
	List() ([]certificates.Certificate, error)
	Update(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error)
}

type devFastStartMaterialReader interface {
	Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error)
}

type devFastStartTLSConfigService interface {
	List() ([]tlsconfigs.TLSConfig, error)
	Create(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
	Update(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
}

type devFastStartRevisionReader interface {
	CurrentActive() (revisions.Revision, bool, error)
}

type devFastStartCompileService interface {
	Create(ctx context.Context) (CompileRequestResult, error)
}

type devFastStartApplyService interface {
	Apply(ctx context.Context, revisionID string) (jobs.Job, error)
}

type devFastStartCertificateIssuer interface {
	Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *ACMEIssueOptions) (jobs.Job, error)
}

type devFastStartEasySiteProfileStore interface {
	Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error)
}

type devFastStartEasySiteProfileService interface {
	Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error)
}

type devFastStartRateLimitPolicyStore interface {
	List() ([]ratelimitpolicies.RateLimitPolicy, error)
}

type DevFastStartBootstrapper struct {
	cfg          config.DevFastStartConfig
	sites        devFastStartSiteService
	upstreams    devFastStartUpstreamService
	certificates devFastStartCertificateService
	materials    devFastStartMaterialReader
	tlsConfigs   devFastStartTLSConfigService
	revisions    devFastStartRevisionReader
	compile      devFastStartCompileService
	apply        devFastStartApplyService
	issuer       devFastStartCertificateIssuer
	easyStore    devFastStartEasySiteProfileStore
	easyProfiles devFastStartEasySiteProfileService
	ratePolicies devFastStartRateLimitPolicyStore
}

func NewDevFastStartBootstrapper(
	cfg config.DevFastStartConfig,
	sites devFastStartSiteService,
	upstreams devFastStartUpstreamService,
	certificates devFastStartCertificateService,
	materials devFastStartMaterialReader,
	tlsConfigs devFastStartTLSConfigService,
	revisions devFastStartRevisionReader,
	compile devFastStartCompileService,
	apply devFastStartApplyService,
	issuer devFastStartCertificateIssuer,
	easyStore devFastStartEasySiteProfileStore,
	easyProfiles devFastStartEasySiteProfileService,
	ratePolicies devFastStartRateLimitPolicyStore,
) *DevFastStartBootstrapper {
	return &DevFastStartBootstrapper{
		cfg:          cfg,
		sites:        sites,
		upstreams:    upstreams,
		certificates: certificates,
		materials:    materials,
		tlsConfigs:   tlsConfigs,
		revisions:    revisions,
		compile:      compile,
		apply:        apply,
		issuer:       issuer,
		easyStore:    easyStore,
		easyProfiles: easyProfiles,
		ratePolicies: ratePolicies,
	}
}

func (b *DevFastStartBootstrapper) Run(ctx context.Context) error {
	if b == nil || !b.cfg.Enabled {
		return nil
	}

	changed, err := b.ensureManagementResources(ctx)
	if err != nil {
		return err
	}

	_, hasActiveRevision, err := b.revisions.CurrentActive()
	if err != nil {
		return err
	}
	if hasActiveRevision && !changed {
		return nil
	}

	compileResult, err := b.compile.Create(ctx)
	if err != nil {
		return fmt.Errorf("dev fast start compile failed: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= b.cfg.MaxAttempts; attempt++ {
		applyJob, applyErr := b.apply.Apply(ctx, compileResult.Revision.ID)
		if applyErr == nil && applyJob.Status == jobs.StatusSucceeded {
			return nil
		}
		if applyErr != nil {
			err = applyErr
		} else {
			err = fmt.Errorf("apply job %s finished with %s: %s", applyJob.ID, applyJob.Status, strings.TrimSpace(applyJob.Result))
		}
		lastErr = err
		if attempt >= b.cfg.MaxAttempts {
			break
		}
		time.Sleep(time.Duration(b.cfg.RetryDelaySeconds) * time.Second)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("dev fast start failed without a reported error")
	}
	return fmt.Errorf("dev fast start failed after %d attempts: %w", b.cfg.MaxAttempts, lastErr)
}

func (b *DevFastStartBootstrapper) ensureManagementResources(ctx context.Context) (bool, error) {
	ctx = withAutoApplyDisabled(ctx)
	siteID := normalizeDevFastStartID(b.cfg.ManagementSiteID)
	host := strings.ToLower(strings.TrimSpace(b.cfg.Host))
	upstreamID := siteID + "-upstream"
	certificateID := normalizeDevFastStartID(b.cfg.CertificateID)
	changed := false

	siteItems, err := b.sites.List()
	if err != nil {
		return false, err
	}
	var existingSite *sites.Site
	for i := range siteItems {
		if siteItems[i].ID == siteID {
			existingSite = &siteItems[i]
			break
		}
	}
	desiredSite := sites.Site{
		ID:          siteID,
		PrimaryHost: host,
		Enabled:     true,
	}
	if existingSite == nil {
		if _, err := b.sites.Create(ctx, desiredSite); err != nil {
			return false, err
		}
		changed = true
	} else if existingSite.PrimaryHost != desiredSite.PrimaryHost || existingSite.Enabled != desiredSite.Enabled {
		desiredSite.CreatedAt = existingSite.CreatedAt
		desiredSite.UpdatedAt = existingSite.UpdatedAt
		if _, err := b.sites.Update(ctx, desiredSite); err != nil {
			return false, err
		}
		changed = true
	}

	upstreamItems, err := b.upstreams.List()
	if err != nil {
		return false, err
	}
	var existingUpstream *upstreams.Upstream
	for i := range upstreamItems {
		if upstreamItems[i].ID == upstreamID {
			existingUpstream = &upstreamItems[i]
			break
		}
	}
	desiredUpstream := upstreams.Upstream{
		ID:     upstreamID,
		SiteID: siteID,
		Host:   strings.TrimSpace(b.cfg.UpstreamHost),
		Port:   b.cfg.UpstreamPort,
		Scheme: "http",
	}
	if existingUpstream == nil {
		if _, err := b.upstreams.Create(ctx, desiredUpstream); err != nil {
			return false, err
		}
		changed = true
	} else if existingUpstream.SiteID != desiredUpstream.SiteID || existingUpstream.Host != desiredUpstream.Host || existingUpstream.Port != desiredUpstream.Port || existingUpstream.Scheme != desiredUpstream.Scheme {
		desiredUpstream.CreatedAt = existingUpstream.CreatedAt
		desiredUpstream.UpdatedAt = existingUpstream.UpdatedAt
		if _, err := b.upstreams.Update(ctx, desiredUpstream); err != nil {
			return false, err
		}
		changed = true
	}

	certificateItems, err := b.certificates.List()
	if err != nil {
		return false, err
	}
	var existingCertificate *certificates.Certificate
	for i := range certificateItems {
		if certificateItems[i].ID == certificateID {
			existingCertificate = &certificateItems[i]
			break
		}
	}
	needsCertificateIssue := false
	if existingCertificate != nil && (existingCertificate.CommonName != host || existingCertificate.Status != "active") {
		updated := *existingCertificate
		updated.CommonName = host
		updated.Status = "active"
		if _, err := b.certificates.Update(ctx, updated); err != nil {
			return false, err
		}
		changed = true
		needsCertificateIssue = true
	}
	if _, _, _, err := b.materials.Read(certificateID); err != nil {
		needsCertificateIssue = true
	}
	if existingCertificate == nil {
		needsCertificateIssue = true
	}
	if needsCertificateIssue {
		job, err := b.issuer.Issue(ctx, certificateID, host, nil, nil)
		if err != nil {
			return false, err
		}
		if job.Status != jobs.StatusSucceeded {
			return false, fmt.Errorf("certificate issue job %s finished with %s: %s", job.ID, job.Status, strings.TrimSpace(job.Result))
		}
		changed = true
	}

	tlsItems, err := b.tlsConfigs.List()
	if err != nil {
		return false, err
	}
	var existingTLS *tlsconfigs.TLSConfig
	for i := range tlsItems {
		if tlsItems[i].SiteID == siteID {
			existingTLS = &tlsItems[i]
			break
		}
	}
	desiredTLS := tlsconfigs.TLSConfig{
		SiteID:        siteID,
		CertificateID: certificateID,
	}
	if existingTLS == nil {
		if _, err := b.tlsConfigs.Create(ctx, desiredTLS); err != nil {
			return false, err
		}
		changed = true
	} else if existingTLS.CertificateID != certificateID {
		desiredTLS.CreatedAt = existingTLS.CreatedAt
		desiredTLS.UpdatedAt = existingTLS.UpdatedAt
		if _, err := b.tlsConfigs.Update(ctx, desiredTLS); err != nil {
			return false, err
		}
		changed = true
	}

	if b.easyStore != nil && b.easyProfiles != nil {
		existingProfile, ok, err := b.easyStore.Get(siteID)
		if err != nil {
			return false, err
		}
		desiredProfile := defaultDevFastStartEasyProfile(siteID, host)
		if !ok {
			if _, err := b.easyProfiles.Upsert(withAutoApplyDisabled(ctx), desiredProfile); err != nil {
				return false, err
			}
			changed = true
		} else if needsDevFastStartEasyProfileUpdate(existingProfile, desiredProfile) {
			existingProfile.HTTPBehavior.AllowedMethods = append([]string(nil), desiredProfile.HTTPBehavior.AllowedMethods...)
			existingProfile.SecurityBehaviorAndLimits.UseLimitReq = desiredProfile.SecurityBehaviorAndLimits.UseLimitReq
			existingProfile.SecurityBehaviorAndLimits.UseLimitConn = desiredProfile.SecurityBehaviorAndLimits.UseLimitConn
			existingProfile.SecurityBehaviorAndLimits.UseBadBehavior = desiredProfile.SecurityBehaviorAndLimits.UseBadBehavior
			existingProfile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = append([]int(nil), desiredProfile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes...)
			existingProfile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = desiredProfile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds
			existingProfile.SecurityBehaviorAndLimits.BadBehaviorThreshold = desiredProfile.SecurityBehaviorAndLimits.BadBehaviorThreshold
			existingProfile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds = desiredProfile.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds
			existingProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = desiredProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1
			existingProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = desiredProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2
			existingProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = desiredProfile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3
			existingProfile.SecurityBehaviorAndLimits.LimitReqRate = desiredProfile.SecurityBehaviorAndLimits.LimitReqRate
			existingProfile.SecurityBehaviorAndLimits.LimitReqURL = desiredProfile.SecurityBehaviorAndLimits.LimitReqURL
			existingProfile.SecurityBehaviorAndLimits.CustomLimitRules = append([]easysiteprofiles.CustomLimitRule(nil), desiredProfile.SecurityBehaviorAndLimits.CustomLimitRules...)
			if _, err := b.easyProfiles.Upsert(withAutoApplyDisabled(ctx), existingProfile); err != nil {
				return false, err
			}
			changed = true
		}
	}

	return changed, nil
}

func normalizeDevFastStartID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func defaultDevFastStartEasyProfile(siteID, host string) easysiteprofiles.EasySiteProfile {
	profile := easysiteprofiles.DefaultProfile(siteID)
	profile.FrontService.ServerName = host
	profile.HTTPBehavior.AllowedMethods = []string{"GET", "POST", "HEAD", "OPTIONS", "PUT", "PATCH", "DELETE"}
	return profile
}

func needsDevFastStartEasyProfileUpdate(current, desired easysiteprofiles.EasySiteProfile) bool {
	if !sameStringSet(current.HTTPBehavior.AllowedMethods, desired.HTTPBehavior.AllowedMethods) {
		return true
	}
	if current.SecurityBehaviorAndLimits.UseLimitReq != desired.SecurityBehaviorAndLimits.UseLimitReq {
		return true
	}
	if current.SecurityBehaviorAndLimits.UseBadBehavior != desired.SecurityBehaviorAndLimits.UseBadBehavior {
		return true
	}
	if current.SecurityBehaviorAndLimits.UseLimitConn != desired.SecurityBehaviorAndLimits.UseLimitConn {
		return true
	}
	if current.SecurityBehaviorAndLimits.LimitReqRate != desired.SecurityBehaviorAndLimits.LimitReqRate {
		return true
	}
	if current.SecurityBehaviorAndLimits.LimitReqURL != desired.SecurityBehaviorAndLimits.LimitReqURL {
		return true
	}
	if !sameCustomLimitRules(current.SecurityBehaviorAndLimits.CustomLimitRules, desired.SecurityBehaviorAndLimits.CustomLimitRules) {
		return true
	}
	if current.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 != desired.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 ||
		current.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 != desired.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 ||
		current.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 != desired.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 {
		return true
	}
	if !sameIntSet(current.SecurityBehaviorAndLimits.BadBehaviorStatusCodes, desired.SecurityBehaviorAndLimits.BadBehaviorStatusCodes) {
		return true
	}
	if current.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds != desired.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds {
		return true
	}
	if current.SecurityBehaviorAndLimits.BadBehaviorThreshold != desired.SecurityBehaviorAndLimits.BadBehaviorThreshold {
		return true
	}
	if current.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds != desired.SecurityBehaviorAndLimits.BadBehaviorCountTimeSeconds {
		return true
	}
	return false
}

func (b *DevFastStartBootstrapper) managementRatePolicyNeedsDisable(siteID string) bool {
	if b.ratePolicies == nil || normalizeDevFastStartID(siteID) != "control-plane-access" {
		return false
	}
	policies, err := b.ratePolicies.List()
	if err != nil {
		return false
	}
	for _, item := range policies {
		if item.SiteID == siteID && item.Enabled {
			return true
		}
	}
	return false
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]int, len(left))
	for _, item := range left {
		seen[item]++
	}
	for _, item := range right {
		if seen[item] == 0 {
			return false
		}
		seen[item]--
	}
	return true
}

func sameCustomLimitRules(left, right []easysiteprofiles.CustomLimitRule) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Path != right[i].Path || left[i].Rate != right[i].Rate {
			return false
		}
	}
	return true
}

func sameIntSet(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[int]int, len(left))
	for _, item := range left {
		seen[item]++
	}
	for _, item := range right {
		if seen[item] == 0 {
			return false
		}
		seen[item]--
	}
	return true
}
