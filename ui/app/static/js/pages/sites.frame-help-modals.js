import { renderHelpModalShell, buildHelpRows } from "./sites.help-modal-shell.js";

function makeFrameRenderer({ chapter, frame, idSlug, rowKeys }) {
  const slug = idSlug || frame.toLowerCase();
  return function renderFrameHelp(ctx, escapeHtml) {
    const modalID = `service-${chapter}-${slug}-help-modal`;
    const titleID = `service-${chapter}-${slug}-help-title`;
    const prefix = `sites.help.${chapter}.${frame}`;
    return renderHelpModalShell({
      modalID,
      titleID,
      titleKey: `${prefix}.title`,
      subtitleKey: `${prefix}.subtitle`,
      rows: buildHelpRows(prefix, rowKeys),
      ctx,
      escapeHtml,
    });
  };
}

export const renderTrafficBadBehaviorHelpModal = makeFrameRenderer({
  chapter: "traffic",
  frame: "badBehavior",
  idSlug: "badbehavior",
  rowKeys: ["activate", "statusCodes", "banDuration", "threshold", "period"],
});

export const renderTrafficLimitsHelpModal = makeFrameRenderer({
  chapter: "traffic",
  frame: "limits",
  rowKeys: [
    "connEnable",
    "maxHttp1",
    "maxHttp2",
    "maxHttp3",
    "reqEnable",
    "reqUrl",
    "reqRate",
    "custom",
  ],
});

export const renderTrafficDnsblHelpModal = makeFrameRenderer({
  chapter: "traffic",
  frame: "dnsbl",
  rowKeys: [
    "blacklistToggle",
    "dnsblToggle",
    "allowlistToggle",
    "exceptionsToggle",
    "lists",
    "blacklistByType",
    "ja3Blacklist",
    "sources",
  ],
});


export const renderTrafficAllowlistHelpModal = makeFrameRenderer({
  chapter: "traffic",
  frame: "allowlist",
  rowKeys: [
    "activate",
    "allowlistIp",
    "exceptions",
    "exceptionsUri",
  ],
});

export const renderTrafficBlacklistHelpModal = makeFrameRenderer({
  chapter: "traffic",
  frame: "blacklist",
  rowKeys: [
    "activate",
    "ip",
    "rdns",
    "asn",
    "userAgent",
    "uri",
    "ja3",
    "ipUrls",
    "dnsblActivate",
    "dnsblProviders",
  ],
});

export const renderUpstreamHeadersHelpModal = makeFrameRenderer({
  chapter: "upstream",
  frame: "headers",
  rowKeys: ["passHost", "xff", "xfp", "xri"],
});
