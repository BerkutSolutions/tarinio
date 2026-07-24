import assert from "node:assert/strict";
import test from "node:test";
import { validateStrictResults } from "./strict-results-lib.mjs";

test("strict result gate accepts only clean reports", () => {
  assert.deepEqual(validateStrictResults({ stats: { expected: 3, unexpected: 0, flaky: 0, skipped: 0 }, errors: [], suites: [] }), []);
});

test("strict result gate rejects skip, flaky, failure and retry evidence", () => {
  const failures = validateStrictResults({
    stats: { expected: 1, unexpected: 1, flaky: 1, skipped: 1 },
    errors: [{}],
    suites: [{ specs: [{ tests: [{ results: [{ retry: 1 }] }] }] }],
  });
  assert.deepEqual(failures, ["unexpected=1", "flaky=1", "skipped=1", "top-level-errors=1", "retried-results=1"]);
});
