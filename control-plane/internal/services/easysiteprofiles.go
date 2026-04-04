package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/wafpolicies"
)

type EasySiteProfileStore interface {
	Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error)
	Create(profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error)
	Update(profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error)
}

type EasySiteProfileService struct {
	store             EasySiteProfileStore
	sites             SiteReader
	wafPolicies       easyWAFPolicyStore
	accessPolicies    easyAccessPolicyStore
	rateLimitPolicies easyRateLimitPolicyStore
	compile           easyRevisionCompileService
	apply             easyRevisionApplyService
	audits            *AuditService
}

type easyRevisionCompileService interface {
	Create(ctx context.Context) (CompileRequestResult, error)
}

type easyRevisionApplyService interface {
	Apply(ctx context.Context, revisionID string) (jobs.Job, error)
}

type easyWAFPolicyStore interface {
	List() ([]wafpolicies.WAFPolicy, error)
	Create(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error)
	Update(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error)
}

type easyAccessPolicyStore interface {
	List() ([]accesspolicies.AccessPolicy, error)
	Create(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error)
	Update(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error)
}

type easyRateLimitPolicyStore interface {
	List() ([]ratelimitpolicies.RateLimitPolicy, error)
	Create(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
	Update(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
}

const maskedSecretValue = "********"

func NewEasySiteProfileService(
	store EasySiteProfileStore,
	sites SiteReader,
	wafPolicies easyWAFPolicyStore,
	accessPolicies easyAccessPolicyStore,
	rateLimitPolicies easyRateLimitPolicyStore,
	compile easyRevisionCompileService,
	apply easyRevisionApplyService,
	audits *AuditService,
) *EasySiteProfileService {
	return &EasySiteProfileService{
		store:             store,
		sites:             sites,
		wafPolicies:       wafPolicies,
		accessPolicies:    accessPolicies,
		rateLimitPolicies: rateLimitPolicies,
		compile:           compile,
		apply:             apply,
		audits:            audits,
	}
}

func (s *EasySiteProfileService) Get(siteID string) (easysiteprofiles.EasySiteProfile, error) {
	site, err := s.ensureSiteExists(siteID)
	if err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	item, ok, err := s.store.Get(site.ID)
	if err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	if ok {
		return maskEasySiteProfileSecrets(item), nil
	}
	out := easysiteprofiles.DefaultProfile(site.ID)
	if site.PrimaryHost != "" {
		out.FrontService.ServerName = site.PrimaryHost
	}
	if err := s.applyLegacyToEasy(site.ID, &out); err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	return maskEasySiteProfileSecrets(out), nil
}

func (s *EasySiteProfileService) Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (updated easysiteprofiles.EasySiteProfile, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "easysiteprofile.upsert",
			ResourceType: "easysiteprofile",
			ResourceID:   profile.SiteID,
			SiteID:       profile.SiteID,
			Status:       auditStatus(err),
			Summary:      "easy site profile upsert",
		})
	}()

	site, err := s.ensureSiteExists(profile.SiteID)
	if err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	profile.SiteID = site.ID

	existing, ok, err := s.store.Get(site.ID)
	if err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	if ok {
		profile = mergeMaskedSecrets(profile, existing)
	}
	if ok {
		updated, err = s.store.Update(profile)
	} else {
		updated, err = s.store.Create(profile)
	}
	if err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	if err := s.syncEasyToLegacy(updated); err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	if err := s.compileAndApply(ctx); err != nil {
		return easysiteprofiles.EasySiteProfile{}, err
	}
	return maskEasySiteProfileSecrets(updated), nil
}

func (s *EasySiteProfileService) compileAndApply(ctx context.Context) error {
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

func (s *EasySiteProfileService) ensureSiteExists(siteID string) (siteIDHost, error) {
	id := normalizeSiteID(siteID)
	if id == "" {
		return siteIDHost{}, fmt.Errorf("site %s not found", id)
	}
	items, err := s.sites.List()
	if err != nil {
		return siteIDHost{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return siteIDHost{ID: item.ID, PrimaryHost: item.PrimaryHost}, nil
		}
	}
	return siteIDHost{}, fmt.Errorf("site %s not found", id)
}

type siteIDHost struct {
	ID          string
	PrimaryHost string
}

func normalizeSiteID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *EasySiteProfileService) applyLegacyToEasy(siteID string, out *easysiteprofiles.EasySiteProfile) error {
	if s.wafPolicies != nil {
		items, err := s.wafPolicies.List()
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.SiteID != siteID {
				continue
			}
			easyPrefix := "easy-" + siteID + "-"
			if strings.HasPrefix(item.ID, easyPrefix) {
				continue
			}
			out.SecurityModSecurity.UseModSecurity = item.Enabled
			out.SecurityModSecurity.UseModSecurityCRSPlugins = item.CRSEnabled
			out.SecurityModSecurity.ModSecurityCRSPlugins = append([]string(nil), item.CustomRuleIncludes...)
			break
		}
	}
	if s.accessPolicies != nil {
		items, err := s.accessPolicies.List()
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.SiteID != siteID {
				continue
			}
			out.SecurityBehaviorAndLimits.BlacklistIP = append([]string(nil), item.DenyList...)
			break
		}
	}
	if s.rateLimitPolicies != nil {
		items, err := s.rateLimitPolicies.List()
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.SiteID != siteID {
				continue
			}
			out.SecurityBehaviorAndLimits.UseLimitReq = item.Enabled
			if item.Limits.RequestsPerSecond > 0 {
				out.SecurityBehaviorAndLimits.LimitReqRate = fmt.Sprintf("%dr/s", item.Limits.RequestsPerSecond)
			}
			break
		}
	}
	return nil
}

func (s *EasySiteProfileService) syncEasyToLegacy(profile easysiteprofiles.EasySiteProfile) error {
	if s.wafPolicies != nil {
		policies, err := s.wafPolicies.List()
		if err != nil {
			return err
		}
		wafID := "easy-" + profile.SiteID + "-waf"
		wafExists := false
		updatedAny := false
		for _, item := range policies {
			if item.SiteID != profile.SiteID {
				continue
			}
			target := wafpolicies.WAFPolicy{
				ID:                 item.ID,
				SiteID:             profile.SiteID,
				Enabled:            false,
				Mode:               wafpolicies.ModeDetection,
				CRSEnabled:         false,
				CustomRuleIncludes: nil,
			}
			if _, err := s.wafPolicies.Update(target); err != nil {
				return err
			}
			updatedAny = true
			if item.ID == wafID {
				wafExists = true
			}
		}
		if !updatedAny {
			target := wafpolicies.WAFPolicy{
				ID:                 wafID,
				SiteID:             profile.SiteID,
				Enabled:            false,
				Mode:               wafpolicies.ModeDetection,
				CRSEnabled:         false,
				CustomRuleIncludes: nil,
			}
			if !wafExists {
				if _, err := s.wafPolicies.Create(target); err != nil {
					return err
				}
			} else {
				if _, err := s.wafPolicies.Update(target); err != nil {
					return err
				}
			}
		}
	}

	if s.accessPolicies != nil {
		policies, err := s.accessPolicies.List()
		if err != nil {
			return err
		}
		accessID := "easy-" + profile.SiteID + "-access"
		accessExists := false
		for _, item := range policies {
			if item.SiteID == profile.SiteID {
				accessID = item.ID
				accessExists = true
				break
			}
		}
		target := accesspolicies.AccessPolicy{
			ID:       accessID,
			SiteID:   profile.SiteID,
			Enabled:  profile.SecurityBehaviorAndLimits.UseBlacklist,
			DenyList: append([]string(nil), profile.SecurityBehaviorAndLimits.BlacklistIP...),
		}
		if !accessExists {
			if _, err := s.accessPolicies.Create(target); err != nil {
				return err
			}
		} else {
			if _, err := s.accessPolicies.Update(target); err != nil {
				return err
			}
		}
	}

	if s.rateLimitPolicies != nil {
		policies, err := s.rateLimitPolicies.List()
		if err != nil {
			return err
		}
		rateID := "easy-" + profile.SiteID + "-rate"
		rateExists := false
		for _, item := range policies {
			if item.SiteID == profile.SiteID {
				rateID = item.ID
				rateExists = true
				break
			}
		}
		rps := parseRPS(profile.SecurityBehaviorAndLimits.LimitReqRate)
		if rps <= 0 {
			rps = 100
		}
		target := ratelimitpolicies.RateLimitPolicy{
			ID:      rateID,
			SiteID:  profile.SiteID,
			Enabled: profile.SecurityBehaviorAndLimits.UseLimitReq,
			Limits: ratelimitpolicies.Limits{
				RequestsPerSecond: rps,
				Burst:             rps,
			},
		}
		if !rateExists {
			if _, err := s.rateLimitPolicies.Create(target); err != nil {
				return err
			}
		} else {
			if _, err := s.rateLimitPolicies.Update(target); err != nil {
				return err
			}
		}
	}

	return nil
}

func parseRPS(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimSuffix(value, "r/s")
	v, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return v
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return maskedSecretValue
}

func maskEasySiteProfileSecrets(profile easysiteprofiles.EasySiteProfile) easysiteprofiles.EasySiteProfile {
	profile.SecurityAntibot.AntibotRecaptchaSecret = maskSecret(profile.SecurityAntibot.AntibotRecaptchaSecret)
	profile.SecurityAntibot.AntibotHcaptchaSecret = maskSecret(profile.SecurityAntibot.AntibotHcaptchaSecret)
	profile.SecurityAntibot.AntibotTurnstileSecret = maskSecret(profile.SecurityAntibot.AntibotTurnstileSecret)
	profile.SecurityAuthBasic.AuthBasicPassword = maskSecret(profile.SecurityAuthBasic.AuthBasicPassword)
	return profile
}

func mergeMaskedSecrets(incoming, existing easysiteprofiles.EasySiteProfile) easysiteprofiles.EasySiteProfile {
	if strings.TrimSpace(incoming.SecurityAntibot.AntibotRecaptchaSecret) == maskedSecretValue {
		incoming.SecurityAntibot.AntibotRecaptchaSecret = existing.SecurityAntibot.AntibotRecaptchaSecret
	}
	if strings.TrimSpace(incoming.SecurityAntibot.AntibotHcaptchaSecret) == maskedSecretValue {
		incoming.SecurityAntibot.AntibotHcaptchaSecret = existing.SecurityAntibot.AntibotHcaptchaSecret
	}
	if strings.TrimSpace(incoming.SecurityAntibot.AntibotTurnstileSecret) == maskedSecretValue {
		incoming.SecurityAntibot.AntibotTurnstileSecret = existing.SecurityAntibot.AntibotTurnstileSecret
	}
	if strings.TrimSpace(incoming.SecurityAuthBasic.AuthBasicPassword) == maskedSecretValue {
		incoming.SecurityAuthBasic.AuthBasicPassword = existing.SecurityAuthBasic.AuthBasicPassword
	}
	return incoming
}
