import fs from "node:fs";

const [reportPath, baselinePath] = process.argv.slice(2);
if (!reportPath || !baselinePath) {
  throw new Error("usage: node scripts/verify-trivy-image-baseline.mjs REPORT BASELINE");
}

const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
const reviewed = new Set((baseline.reviewed_unfixed ?? []).map(([pkg, id]) => `${pkg}\u0000${id}`));
const blocking = [];
const accepted = [];

for (const result of report.Results ?? []) {
  for (const finding of result.Vulnerabilities ?? []) {
    if (!['HIGH', 'CRITICAL'].includes(finding.Severity)) continue;
    const key = `${finding.PkgName}\u0000${finding.VulnerabilityID}`;
    if (finding.FixedVersion) {
      blocking.push(`${finding.PkgName}: ${finding.VulnerabilityID} has fix ${finding.FixedVersion}`);
    } else if (reviewed.has(key)) {
      accepted.push(key);
    } else {
      blocking.push(`${finding.PkgName}: ${finding.VulnerabilityID} is new and has no reviewed exception`);
    }
  }
}

if (blocking.length) {
  throw new Error(`blocking Trivy image finding(s):\n${blocking.join('\n')}`);
}

console.log(`[trivy-image-baseline] accepted ${accepted.length} reviewed upstream-unfixed finding(s); blocking findings: 0`);
