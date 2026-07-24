export function validateStrictResults(report) {
  const stats = report?.stats || {};
  const failures = [];
  if (!Number.isInteger(stats.expected) || stats.expected < 1) failures.push("no passed tests were reported");
  for (const field of ["unexpected", "flaky", "skipped"]) {
    const value = Number(stats[field] || 0);
    if (value !== 0) failures.push(`${field}=${value}`);
  }
  if (Array.isArray(report?.errors) && report.errors.length > 0) failures.push(`top-level-errors=${report.errors.length}`);

  const retries = [];
  const visit = (value) => {
    if (!value || typeof value !== "object") return;
    if (Array.isArray(value)) {
      for (const item of value) visit(item);
      return;
    }
    if (Number(value.retry || 0) > 0) retries.push(Number(value.retry));
    for (const nested of Object.values(value)) visit(nested);
  };
  visit(report?.suites || []);
  if (retries.length > 0) failures.push(`retried-results=${retries.length}`);
  return failures;
}
