import { renderHelpModalShell, buildHelpRows } from "./sites.help-modal-shell.js";

// Helper: builds a chapter modal renderer with a stable id pattern.
function makeChapterRenderer({ chapter, rowKeys }) {
  return function renderChapterHelp(ctx, escapeHtml) {
    const modalID = `service-${chapter}-chapter-help-modal`;
    const titleID = `service-${chapter}-chapter-help-title`;
    return renderHelpModalShell({
      modalID,
      titleID,
      titleKey: `sites.help.${chapter}.title`,
      subtitleKey: `sites.help.${chapter}.subtitle`,
      rows: buildHelpRows(`sites.help.${chapter}`, rowKeys),
      ctx,
      escapeHtml,
    });
  };
}

export const renderFrontChapterHelpModal = makeChapterRenderer({
  chapter: "front",
  rowKeys: [
    "serverName",
    "serviceId",
    "securityMode",
    "serviceProfile",
    "caServer",
    "serviceEnabled",
    "adaptiveModel",
    "autoLetsEncrypt",
    "letsEncryptStaging",
    "wildcard",
    "tlsEnabled",
    "tlsSelfSigned",
    "certificateId",
  ],
});

export const renderUpstreamChapterHelpModal = makeChapterRenderer({
  chapter: "upstream",
  rowKeys: [
    "useReverseProxy",
    "keepalive",
    "websocket",
    "customHost",
    "reverseProxyUrl",
    "target",
    "headerForwarding",
    "sslSni",
  ],
});

export const renderHttpChapterHelpModal = makeChapterRenderer({
  chapter: "http",
  rowKeys: [
    "allowedMethods",
    "maxBodySize",
    "sslProtocols",
    "http2",
    "http3",
    "strictParsing",
  ],
});

export const renderHeadersChapterHelpModal = makeChapterRenderer({
  chapter: "headers",
  rowKeys: [
    "cookieFlags",
    "referrerPolicy",
    "hsts",
    "hstsMaxAge",
    "hstsIncludeSubdomains",
    "hstsPreload",
    "csp",
    "permissionsPolicy",
    "keepUpstreamHeaders",
    "useCors",
    "corsAllowedOrigins",
  ],
});

export const renderBlockingChapterHelpModal = makeChapterRenderer({
  chapter: "blocking",
  rowKeys: [
    "enabled",
    "scope",
    "stages",
    "stageFormat",
  ],
});

export const renderAntibotChapterHelpModal = makeChapterRenderer({
  chapter: "antibot",
  rowKeys: [
    "frameAntibot",
    "frameAuth",
    "order",
    "exclusions",
    "escalation",
  ],
});

export const renderGeoChapterHelpModal = makeChapterRenderer({
  chapter: "geo",
  rowKeys: [
    "blacklist",
    "whitelist",
    "groups",
    "timeWindows",
  ],
});

export const renderModsecChapterHelpModal = makeChapterRenderer({
  chapter: "modsec",
  rowKeys: [
    "useModsec",
    "useCrsPlugins",
    "useCustom",
    "crsVersion",
    "plugins",
    "customPath",
    "customContent",
  ],
});

export const renderWebSocketChapterHelpModal = makeChapterRenderer({
  chapter: "websocket",
  rowKeys: [
    "useInspection",
    "maxMessageBytes",
    "rateMsgPerSec",
    "blockPatterns",
  ],
});

export const renderVirtualPatchesChapterHelpModal = makeChapterRenderer({
  chapter: "virtualpatches",
  rowKeys: [
    "overview",
    "pattern",
    "target",
    "action",
  ],
});

export const renderUpstreamMtlsChapterHelpModal = makeChapterRenderer({
  chapter: "upstreamMtls",
  rowKeys: [
    "enabled",
    "certRef",
    "keyRef",
    "caRef",
  ],
});

export function renderFrontMainHelpModal(ctx, escapeHtml) {
  return renderHelpModalShell({
    modalID: "service-front-main-help-modal",
    titleID: "service-front-main-help-title",
    titleKey: "sites.help.front.main.title",
    subtitleKey: "sites.help.front.main.subtitle",
    rows: buildHelpRows("sites.help.front.main", [
      "serverName", "serviceId", "securityMode", "serviceProfile",
      "caServer", "serviceEnabled", "adaptiveModel", "autoLetsEncrypt",
      "letsEncryptStaging", "wildcard", "tlsEnabled", "tlsSelfSigned", "certificateId",
    ]),
    ctx,
    escapeHtml,
  });
}

export function renderFrontMtlsHelpModal(ctx, escapeHtml) {
  return renderHelpModalShell({
    modalID: "service-front-mtls-help-modal",
    titleID: "service-front-mtls-help-title",
    titleKey: "sites.help.front.mtls.title",
    subtitleKey: "sites.help.front.mtls.subtitle",
    rows: buildHelpRows("sites.help.front.mtls", [
      "enabled", "optional", "passHeaders", "verifyDepth", "clientCaRef",
    ]),
    ctx,
    escapeHtml,
  });
}

