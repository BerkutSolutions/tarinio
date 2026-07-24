export function validateRegistry(registryIds, baselineIds) {
  const duplicateIds = registryIds.filter((id, index) => registryIds.indexOf(id) !== index);
  const missingIds = baselineIds.filter((id) => !registryIds.includes(id));
  const unexpectedIds = registryIds.filter((id) => !baselineIds.includes(id));
  return { duplicateIds: [...new Set(duplicateIds)], missingIds, unexpectedIds };
}

export function coverageByTab(registry, passedTitles) {
  const browserRegistry = registry.filter((item) => item.layer === "browser-ui");
  return [...new Set(browserRegistry.map((item) => item.tab))].map((tab) => {
    const rows = browserRegistry.filter((item) => item.tab === tab);
    const passed = rows.filter((item) => {
      const expected = item.testTitles?.length ? item.testTitles : [item.id];
      return expected.some((title) => passedTitles.has(title));
    }).length;
    return { tab, passed, total: rows.length, percent: rows.length ? Math.round((passed / rows.length) * 100) : 0 };
  });
}

export function coverageRegressions(rows, minimumByTab) {
  return rows.filter((row) => row.percent < Number(minimumByTab[row.tab] ?? 100));
}
