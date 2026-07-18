import fs from "node:fs";

function findingKey(target, id) {
  return `${target}\u0000${id}`;
}

function securityFindings(report) {
  const findings = [];
  for (const result of report.Results ?? []) {
    const target = result.Target ?? "unknown";
    for (const item of result.Vulnerabilities ?? []) {
      if (["HIGH", "CRITICAL"].includes(item.Severity)) {
        findings.push({ key: findingKey(target, item.VulnerabilityID), kind: "vulnerability" });
      }
    }
    for (const item of result.Secrets ?? []) {
      if (["HIGH", "CRITICAL"].includes(item.Severity)) {
        findings.push({ key: findingKey(target, item.RuleID), kind: "secret" });
      }
    }
    for (const item of result.Misconfigurations ?? []) {
      if (["HIGH", "CRITICAL"].includes(item.Severity)) {
        findings.push({ key: findingKey(target, item.ID), kind: "misconfiguration" });
      }
    }
  }
  return findings;
}

function countByKey(findings) {
  const counts = new Map();
  for (const finding of findings) {
    counts.set(finding.key, (counts.get(finding.key) ?? 0) + 1);
  }
  return counts;
}

const [reportPath, baselinePath] = process.argv.slice(2);
if (!reportPath || !baselinePath) {
  throw new Error("usage: node scripts/verify-trivy-baseline.mjs REPORT BASELINE");
}

const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
const allowed = new Map(
  baseline.misconfigurations.map((entry) => [findingKey(entry.target, entry.id), entry.maxCount]),
);
const observed = countByKey(securityFindings(report));
const unexpected = [];

for (const [key, count] of observed) {
  const [target, id] = key.split("\u0000");
  const allowedCount = allowed.get(key);
  if (allowedCount === undefined || count > allowedCount) {
    unexpected.push(`${target}: ${id} (${count}; allowed ${allowedCount ?? 0})`);
  }
}

if (unexpected.length > 0) {
  throw new Error(`new or expanded Trivy high/critical finding(s):\n${unexpected.join("\n")}`);
}

console.log(`[trivy-baseline] accepted ${observed.size} reviewed finding key(s); new high/critical findings: 0`);
