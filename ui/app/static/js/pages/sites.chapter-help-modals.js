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
    "reverseProxyHost",
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
