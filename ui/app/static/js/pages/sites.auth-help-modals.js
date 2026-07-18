import { renderHelpModalShell } from "./sites.help-modal-shell.js";

export function renderAuthHelpModal(ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const rows = [
    { labelKey: "sites.easy.antibot.useAuthBasic", helpKey: "sites.easy.auth.help.enable" },
    { labelKey: "sites.easy.auth.mode", helpKey: "sites.easy.auth.help.mode" },
    { labelKey: "sites.easy.auth.order", helpKey: "sites.easy.auth.help.order" },
    { labelKey: "sites.easy.antibot.authBasicLocation", helpKey: "sites.easy.auth.help.location" },
    { labelKey: "sites.easy.antibot.authText", helpKey: "sites.easy.auth.help.text" },
    { labelKey: "sites.easy.auth.template", helpKey: "sites.easy.auth.help.template" },
    { labelKey: "sites.easy.antibot.authSessionTtl", helpKey: "sites.easy.auth.help.sessionTtl" },
    { labelKey: "sites.easy.auth.exclusionRules", helpKey: "sites.easy.auth.help.exclusions" },
    { labelKey: "sites.easy.antibot.authUsers", helpKey: "sites.easy.auth.help.users" },
    { labelKey: "sites.easy.auth.serviceTokens", helpKey: "sites.easy.auth.help.tokens" }
  ];
  return renderHelpModalShell({
    modalID: "service-auth-help-modal",
    titleID: "service-auth-help-title",
    titleKey: "sites.easy.auth.help.title",
    subtitleKey: "sites.easy.auth.help.subtitle",
    rows,
    ctx,
    escapeHtml,
  });
}

export function renderAntibotHelpModal(ctx, deps = {}) {
  const escapeHtml = deps.escapeHtml;
  const rows = [
    { labelKey: "sites.easy.antibot.challenge", helpKey: "sites.easy.antibot.help.challenge" },
    { labelKey: "sites.easy.antibot.url", helpKey: "sites.easy.antibot.help.url" },
    { labelKey: "sites.easy.antibot.scannerAutoBanEnabled", helpKey: "sites.easy.antibot.help.scanner" },
    { labelKey: "sites.easy.antibot.recaptchaScore", helpKey: "sites.easy.antibot.help.recaptchaScore" },
    { labelKey: "sites.easy.antibot.recaptchaSitekey", helpKey: "sites.easy.antibot.help.providerKeys" },
    { labelKey: "sites.easy.antibot.exclusionRules", helpKey: "sites.easy.antibot.help.exclusions" },
    { labelKey: "sites.easy.antibot.twoLayerEscalation", helpKey: "sites.easy.antibot.help.escalation" },
    { labelKey: "sites.easy.antibot.escalationMode", helpKey: "sites.easy.antibot.help.escalationMode" },
    { labelKey: "sites.easy.antibot.challengeRulesByUrl", helpKey: "sites.easy.antibot.help.challengeRules" }
  ];
  return renderHelpModalShell({
    modalID: "service-antibot-help-modal",
    titleID: "service-antibot-help-title",
    titleKey: "sites.easy.antibot.help.title",
    subtitleKey: "sites.easy.antibot.help.subtitle",
    rows,
    ctx,
    escapeHtml,
  });
}
