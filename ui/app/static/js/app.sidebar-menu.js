function renderMenuLink(section, active, icons, translate, sectionPath, labelKey) {
  const label = translate(labelKey || section.labelKey);
  return `
    <a class="sidebar-link ${section.id === active.id ? "active" : ""}" href="${sectionPath(section)}" data-path="${section.id}" title="${label}">
      <span class="sidebar-link-icon">${icons[section.id] || ""}</span>
      <span class="sidebar-link-label">${label}</span>
    </a>
  `;
}

function renderPendingMenuLink(item, icons, translate) {
  return `
    <span class="sidebar-link sidebar-link-pending" aria-disabled="true" title="${translate(item.labelKey)}">
      <span class="sidebar-link-icon">${icons[item.id] || ""}</span>
      <span class="sidebar-link-label">${translate(item.labelKey)}</span>
    </span>
  `;
}

const sidebarGroups = [
  { labelKey: "app.sidebarGroup.monitoring", sections: ["dashboard"], pending: [{ id: "incidents", labelKey: "app.incidents" }] },
  { labelKey: "app.sidebarGroup.management", sections: ["sites", "requests", "bans", "revisions"] },
  { labelKey: "app.sidebarGroup.protection", sections: ["antiddos", "owaspcrs", "tls"], labels: { tls: "app.sidebarCertificates" } },
  {
    labelKey: "app.sidebarGroup.system",
    sections: ["administration", "events", "activity", "settings"],
    labels: { events: "app.sidebarJournal", activity: "app.sidebarAudit" },
  },
];

export function renderSidebarMenu({ menu, sections, active, icons, translate, sectionPath, canAccessSection }) {
  const available = new Map(
    sections
      .filter((section) => !section.hiddenInMenu && canAccessSection(section.id))
      .map((section) => [section.id, section]),
  );

  menu.innerHTML = sidebarGroups.map((group) => {
    const links = group.sections
      .map((sectionID) => available.get(sectionID))
      .filter(Boolean)
      .map((section) => renderMenuLink(section, active, icons, translate, sectionPath, group.labels?.[section.id]))
      .join("") + (group.pending || []).map((item) => renderPendingMenuLink(item, icons, translate)).join("");
    if (!links) {
      return "";
    }
    return `<section class="sidebar-nav-group"><h2 class="sidebar-group-title">${translate(group.labelKey)}</h2>${links}</section>`;
  }).join("");
}
