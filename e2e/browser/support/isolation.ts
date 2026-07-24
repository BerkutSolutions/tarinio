import { expect, type TestInfo } from "@playwright/test";
import { createHash, randomUUID } from "node:crypto";

let sequence = 0;
const processRunID = (process.env.WAF_E2E_RUN_ID || randomUUID()).replace(/[^a-z0-9]+/gi, "").toLowerCase().slice(-10);

function shortHash(value: string) {
  return createHash("sha256").update(value).digest("hex").slice(0, 8);
}

export function e2eID(testInfo: TestInfo, prefix: string) {
  sequence += 1;
  const project = shortHash(testInfo.project.name);
  const test = shortHash(testInfo.testId || testInfo.titlePath.join("/"));
  const worker = Math.max(0, testInfo.workerIndex).toString(36);
  const retry = Math.max(0, testInfo.retry).toString(36);
  return `${prefix}-${processRunID}-${project}-${test}-${worker}${retry}-${sequence.toString(36)}`.slice(0, 63);
}

type CleanupEntry = { label: string; remove: () => Promise<unknown>; absent: () => Promise<boolean> };

export class CleanupLedger {
  private entries: CleanupEntry[] = [];

  add(label: string, remove: () => Promise<unknown>, absent: () => Promise<boolean>) {
    this.entries.push({ label, remove, absent });
  }

  async run() {
    const failures: string[] = [];
    for (const entry of [...this.entries].reverse()) {
      try {
        await entry.remove();
        await expect.poll(entry.absent, { timeout: 30_000, message: `cleanup left ${entry.label}` }).toBe(true);
      } catch (error) {
        failures.push(`${entry.label}: ${error instanceof Error ? error.message : String(error)}`);
      }
    }
    if (failures.length) throw new Error("E2E cleanup failed:\n" + failures.join("\n"));
  }
}
