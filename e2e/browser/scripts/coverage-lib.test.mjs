import assert from "node:assert/strict";
import test from "node:test";
import { coverageByTab, coverageRegressions, validateRegistry } from "./coverage-lib.mjs";
import {
  apiCoverageByTab,
  mergeGoEvidence,
  parseGoEvidenceReport,
  parseGoTestJSONLines,
  validateApiEvidence,
  validateGoExecution,
} from "./api-evidence-lib.mjs";

test("registry guard detects duplicate, missing and unexpected IDs", () => {
  assert.deepEqual(validateRegistry(["a", "a", "c"], ["a", "b"]), {
    duplicateIds: ["a"], missingIds: ["b"], unexpectedIds: ["c"],
  });
});

test("coverage guard detects a per-tab regression", () => {
  const rows = coverageByTab([
    { id: "a.one", tab: "a", layer: "browser-ui" },
    { id: "a.two", tab: "a", layer: "browser-ui" },
  ], new Set(["a.one"]));
  assert.deepEqual(rows, [{ tab: "a", passed: 1, total: 2, percent: 50 }]);
  assert.deepEqual(coverageRegressions(rows, { a: 100 }), rows);
});

test("coverage requires exact registered test titles", () => {
  const registry = [{ id: "settings.save", tab: "settings", layer: "browser-ui", testTitles: ["settings.save"] }];
  assert.equal(coverageByTab(registry, new Set(["prefix settings.save suffix"]))[0].passed, 0);
  assert.equal(coverageByTab(registry, new Set(["settings.save"]))[0].passed, 1);
});

test("Go JSON parser records exact test and subtest terminal actions", () => {
  const result = parseGoTestJSONLines([
    JSON.stringify({ Action: "run", Test: "TestE2EAPI" }),
    JSON.stringify({ Action: "pass", Test: "TestE2EAPI/ReadContract" }),
    JSON.stringify({ Action: "skip", Test: "TestE2EAPI/Optional" }),
    JSON.stringify({ Action: "pass", Package: "waf/ui/tests" }),
    "not-json",
  ].join("\n"));
  assert.deepEqual([...result], [
    ["TestE2EAPI/ReadContract", "pass"],
    ["TestE2EAPI/Optional", "skip"],
  ]);
});

test("Go evidence merge never hides a failed or skipped execution with a pass", () => {
  const aggregate = parseGoEvidenceReport({ tests: [
    { name: "TestE2EAPI/Read", status: "pass" },
    { name: "TestE2EAPI/Write", status: "skip" },
  ] });
  mergeGoEvidence(aggregate, parseGoTestJSONLines([
    JSON.stringify({ Action: "fail", Test: "TestE2EAPI/Read" }),
    JSON.stringify({ Action: "pass", Test: "TestE2EAPI/Write" }),
  ].join("\n")));
  assert.deepEqual([...aggregate], [
    ["TestE2EAPI/Read", "fail"],
    ["TestE2EAPI/Write", "skip"],
  ]);
});

test("API coverage requires every exact Go test mapped by a workflow", () => {
  const registry = [
    { id: "api.read", tab: "api", layer: "api-runtime", goTests: ["TestE2EAPI/Read"] },
    { id: "api.write", tab: "api", layer: "api-runtime", goTests: ["TestE2EAPI/Write", "TestE2EAPI/Audit"] },
    { id: "api.unmapped", tab: "api", layer: "api-runtime" },
  ];
  const evidence = new Map([
    ["TestE2EAPI/Read", "pass"],
    ["TestE2EAPI/Write", "pass"],
    ["TestE2EAPI/Audit", "fail"],
  ]);
  assert.deepEqual(apiCoverageByTab(registry, evidence), [{ tab: "api", passed: 1, total: 3, percent: 33 }]);
  assert.deepEqual(validateApiEvidence(registry, evidence), {
    missingMappings: ["api.unmapped"],
    missingTests: [],
    failedTests: ["api.write:TestE2EAPI/Audit"],
    skippedTests: [],
  });
});

test("API evidence reports missing and skipped mapped tests", () => {
  const registry = [
    { id: "api.read", tab: "api", layer: "api-runtime", goTests: ["TestE2EAPI/Read"] },
    { id: "api.write", tab: "api", layer: "api-runtime", goTests: ["TestE2EAPI/Write"] },
  ];
  assert.deepEqual(validateApiEvidence(registry, new Map([["TestE2EAPI/Read", "skip"]])), {
    missingMappings: [],
    missingTests: ["api.write:TestE2EAPI/Write"],
    failedTests: [],
    skippedTests: ["api.read:TestE2EAPI/Read"],
  });
});

test("Go execution gate rejects every failed or skipped test", () => {
  assert.deepEqual(validateGoExecution(new Map([
    ["TestE2EAPI/Pass", "pass"],
    ["TestE2EAPI/Fail", "fail"],
    ["TestE2EAPI/Skip", "skip"],
  ])), {
    failed: ["TestE2EAPI/Fail"],
    skipped: ["TestE2EAPI/Skip"],
  });
});
