#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const repoRoot = path.resolve(__dirname, "..");
const lockfilePath = path.join(repoRoot, "package-lock.json");

const lock = JSON.parse(fs.readFileSync(lockfilePath, "utf8"));
const problems = [];

for (const [pkgPath, meta] of Object.entries(lock.packages || {})) {
  if (!pkgPath.startsWith("node_modules/")) {
    continue;
  }
  if (!meta || typeof meta.version !== "string" || typeof meta.resolved !== "string") {
    continue;
  }
  if (!meta.resolved.startsWith("https://registry.npmjs.org/")) {
    continue;
  }

  const packageName = pkgPath.slice("node_modules/".length);
  const leaf = packageName.slice(packageName.lastIndexOf("/") + 1);
  const expectedSuffix = `/-/${leaf}-${meta.version}.tgz`;
  if (!meta.resolved.endsWith(expectedSuffix)) {
    problems.push(`${pkgPath}: version=${meta.version}, resolved=${meta.resolved}`);
  }
}

if (problems.length > 0) {
  console.error("package-lock.json contains suspicious npm tarball references:");
  for (const problem of problems.slice(0, 20)) {
    console.error(`- ${problem}`);
  }
  process.exit(1);
}

console.log("package-lock.json tarball references look consistent");
