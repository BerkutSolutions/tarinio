import { normalizeArray } from "./sites.routing-merge.js";
import { normalizeAntibotChallengeRules, normalizeAntibotExclusionRules, normalizeCustomLimitRules, normalizeStringArray } from "./sites.normalize.js";
import { normalizeAuthBasicUsers } from "./sites.auth-geo.js";
import { BAN_SCOPE_VALUES, normalizeBanEscalationStages } from "./sites.traffic-helpers.js";

export function validateDraft(draft, ctx) {
  if (!draft.id.trim()) return ctx.t("sites.validation.siteIdRequired");
  if (!draft.primary_host.trim()) return ctx.t("sites.validation.primaryHostRequired");
  if (!draft.upstream_id.trim()) return ctx.t("sites.validation.upstreamIdRequired");
  if (!draft.upstream_host.trim()) return ctx.t("sites.validation.upstreamHostRequired");
  if (!Number.isInteger(draft.upstream_port) || draft.upstream_port < 1 || draft.upstream_port > 65535) return ctx.t("sites.validation.portRange");
  if (!normalizeStringArray(draft.allowed_methods).length) return ctx.t("sites.validation.allowedMethodsRequired");
  if (draft.use_bad_behavior && !normalizeArray(draft.bad_behavior_status_codes).length) return ctx.t("sites.validation.badBehaviorStatusCodesRequired");
  if (draft.use_bad_behavior && (!Number.isFinite(draft.bad_behavior_ban_time_seconds) || draft.bad_behavior_ban_time_seconds < 0)) return ctx.t("sites.validation.badBehaviorBanDuration");
  if (draft.ban_escalation_enabled) {
    if (!BAN_SCOPE_VALUES.includes(String(draft.ban_escalation_scope || "").trim().toLowerCase())) return ctx.t("sites.validation.banEscalationScope");
    const stages = normalizeBanEscalationStages(draft.ban_escalation_stages_seconds, draft.bad_behavior_ban_time_seconds);
    if (!stages.length) return ctx.t("sites.validation.banEscalationStagesRequired");
    if (stages.length > 12) return ctx.t("sites.validation.banEscalationStagesLimit");
    for (let i = 0; i < stages.length; i += 1) {
      const value = stages[i];
      if (!Number.isFinite(value) || value < 0) return ctx.t("sites.validation.banEscalationStageValue");
      if (value === 0 && i !== stages.length - 1) return ctx.t("sites.validation.banEscalationPermanentLast");
    }
  }
  if (draft.use_limit_req && !String(draft.limit_req_rate || "").trim()) return ctx.t("sites.validation.limitReqRateRequired");
  if (draft.use_limit_req && !/^\d+r\/s$/i.test(String(draft.limit_req_rate || "").trim().replace(/\s+/g, ""))) return ctx.t("sites.validation.limitReqRateFormat");
  if (normalizeCustomLimitRules(draft.custom_limit_rules).length > 32) return ctx.t("sites.validation.customLimitRulesLimit");
  if (draft.antibot_challenge === "no" && normalizeAntibotExclusionRules(draft.antibot_exclusion_rules).length) return ctx.t("sites.validation.antibotExclusionsRequireEnabled");
  if (draft.hsts_enabled && (!Number.isFinite(draft.hsts_max_age_seconds) || Number(draft.hsts_max_age_seconds) <= 0)) return ctx.t("sites.validation.hstsMaxAgePositive");
  if (draft.hsts_preload && !draft.hsts_enabled) return ctx.t("sites.validation.hstsPreloadNeedsEnabled");
  if (draft.hsts_include_subdomains && !draft.hsts_enabled) return ctx.t("sites.validation.hstsPreloadNeedsEnabled");
  if (draft.hsts_preload && !draft.hsts_include_subdomains) return ctx.t("sites.validation.hstsPreloadNeedsIncludeSubdomains");
  if (draft.hsts_preload && Number(draft.hsts_max_age_seconds || 0) < 31536000) return ctx.t("sites.validation.hstsPreloadNeedsMaxAge");
  for (const rule of normalizeCustomLimitRules(draft.custom_limit_rules)) {
    if (!rule.path.startsWith("/")) return ctx.t("sites.validation.customLimitPathFormat");
    if (!/^\d+r\/s$/i.test(rule.rate)) return ctx.t("sites.validation.customLimitRateFormat");
  }
  if (normalizeAntibotExclusionRules(draft.antibot_exclusion_rules).length > 32) return ctx.t("sites.validation.antibotExclusionRulesLimit");
  for (const rule of normalizeAntibotExclusionRules(draft.antibot_exclusion_rules)) {
    if (!rule.path.startsWith("/")) return ctx.t("sites.validation.antibotExclusionPathFormat");
    const methods = Array.isArray(rule.methods) ? rule.methods : [];
    if (!methods.length) return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
    if (methods.includes("*") && methods.length !== 1) return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
    for (const method of methods) {
      if (!["*", "GET", "POST", "HEAD", "OPTIONS", "PUT", "DELETE", "PATCH"].includes(method)) return ctx.t("sites.validation.antibotExclusionMethodsInvalid");
    }
  }
  if (normalizeAntibotChallengeRules(draft.antibot_challenge_rules).length > 32) return "Too many antibot challenge rules (max 32).";
  for (const rule of normalizeAntibotChallengeRules(draft.antibot_challenge_rules)) {
    if (!rule.path.startsWith("/")) return "Antibot challenge rule path must start with /.";
    if (!["cookie", "javascript", "captcha", "recaptcha", "hcaptcha", "turnstile", "mcaptcha"].includes(rule.challenge)) return "Antibot challenge rule mode is invalid.";
  }
  if (draft.use_auth_basic) {
    const users = normalizeAuthBasicUsers(draft.auth_basic_users);
    const enabledUsers = users.filter((item) => item.enabled);
    if (!enabledUsers.length) return ctx.t("sites.validation.authBasicUserRequired");
    for (const user of enabledUsers) {
      if (!String(user.password || "").trim()) return ctx.t("sites.validation.authBasicPasswordRequired");
    }
  }
  if (draft.use_modsecurity_custom_configuration && !String(draft.modsecurity_custom_path || "").trim()) return ctx.t("sites.validation.modsecCustomPathRequired");
  return "";
}
