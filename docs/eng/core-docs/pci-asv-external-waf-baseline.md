# PCI/ASV External WAF Baseline

This page defines the hardening baseline for a standalone external TARINIO WAF that protects an upstream production website.

## Scope

In scope:

- internet-facing TARINIO runtime edge;
- control-plane management boundary;
- upstream production site reachable only through WAF routing.

Out of scope:

- internal Kubernetes ingress topology;
- non-edge internal service scans not routed through the external WAF path.

## Finding Matrix

| Scanner finding class | Typical example | Owner | Required control | Verification |
| --- | --- | --- | --- | --- |
| Network stack disclosure | `TCP timestamps enabled` | Host OS / platform ops | Disable timestamps or document accepted exception with risk rationale | Host sysctl/netstack checks + external scan |
| Edge attack surface | Extra public ports | Deployment profile owner | Publish only required edge ports (`80/443`) and keep management endpoints non-public | `docker compose ps`, host firewall review, port scan |
| TLS protocol/cipher posture | Protocol/cipher reporting findings | Runtime TLS policy owner | Keep approved TLS versions/ciphers and remove weak/deprecated options | TLS scan + revision artifact review |
| HSTS posture | `preload` missing in HSTS | Runtime policy owner | Support preload-ready mode where policy allows it | Header checks on representative responses |
| Deprecated legacy controls | HPKP missing | Compliance owner | Record explicit "not applicable/deprecated" policy statement | Compliance evidence pack |
| Informational discovery | Host alive / service detection / ALPN reporting | Security operations | Keep expected and controlled; no unnecessary exposure | Periodic external scans + inventory |

## Acceptance Criteria (Definition of Done)

A release passes this baseline when all are true:

1. External edge exposes only intended services and ports for production traffic.
2. Management UI/API are not publicly exposed by default deployment profile.
3. Runtime TLS and security-header posture is stable across revisions.
4. Deprecated scanner checks are handled by explicit compliance policy, not ignored ad hoc.
5. A repeatable preflight check can be executed before official ASV scans.
6. The latest ASV cycle reports no problem vulnerabilities requiring remediation.

## Evidence Bundle Requirements

For each scan cycle, archive:

- external scan report set (executive + detailed + attestation if available);
- effective compose/deploy profile and env overrides used in production;
- preflight output for ports/TLS/headers;
- revision id and apply status proving deployed policy state;
- approved exceptions with owner and review timestamp.

## Preflight Procedure Before ASV Window

Run preflight from the external WAF edge host (or a controlled probe host with equivalent visibility):

```bash
TARGET_HOST=<public-waf-hostname> \
TARGET_HTTP_PORT=80 \
TARGET_HTTPS_PORT=443 \
EXPECTED_OPEN_PORTS=80,443 \
PORT_PROBE_SET=22,80,443,8080,8443,9200 \
COMPLIANCE_POLICY_FILE=security/compliance/deprecated-controls-policy.json \
./scripts/pci-preflight-perimeter.sh
```

Archive `summary.json`, `summary.txt`, and all generated check artifacts for the evidence pack.

## ASV Result Interpretation Rules

Use this interpretation order during triage:

1. Failing controls affecting exposure, protocol posture, or exploitability must be remediated and rescanned.
2. Deprecated legacy controls (for example HPKP / Expect-CT) must be documented against `security/compliance/deprecated-controls-policy.json` and treated as policy-backed exceptions, not hidden failures.
3. Informational findings remain tracked in inventory, but are not treated as release blockers unless they indicate unintended exposure.

## Safe Rollback Rules For Scan Regressions

If preflight or ASV regression appears after a change:

1. Roll back to the last known-good revision in `Revisions`.
2. Re-run preflight and confirm `"overall": "pass"`.
3. Record rollback reason and attach artifacts to the incident or change record.
4. Re-introduce fixes as a narrower follow-up revision.
