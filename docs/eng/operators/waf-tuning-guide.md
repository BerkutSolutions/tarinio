# Practical WAF Tuning Guide

This document is intended for a Stage 1 operator who already has:

- a running stack;
- completed onboarding;
- a working single-node WAF deployment.

It is not a theoretical ModSecurity reference. It is a practical guide for changing policies safely.

## How WAF Fits Into The Architecture

In TARINIO, WAF is not tuned by editing runtime files directly.

The flow is always:

```text
control-plane -> policy changes -> compile -> revision bundle -> apply -> runtime
```

This means:

- the operator changes policies in UI or API;
- the control-plane creates a new revision;
- the compiler generates a deterministic bundle;
- runtime uses only the active compiled bundle.

Manual editing of:

- NGINX config;
- ModSecurity config;
- active bundle files

is an architectural violation and will be overwritten by later revisions.

## WAF Modes

### Detection

Use it when:

- onboarding WAF for a new site;
- testing a new policy;
- investigating false positives.

### Prevention

Use it when:

- the site' traffic profile is understood;
- the baseline policy has been validated;
- false positives are under control.

Recommended rollout:

1. `Detection`
2. observe
3. `Prevention`

## Safe Start

For a new site:

- enable the WAF policy;
- keep CRS enabled;
- start in `Detection`.

Then:

1. compile a revision;
2. apply the revision;
3. inspect activity, events, and requests;
4. move to `Prevention` only after that.

## Typical Problems

### False `403`

Recommended sequence:

1. open `Events` or related observability screens;
2. locate the relevant event near the affected timestamp;
3. identify the site, revision, and policy change;
4. add a narrow exception;
5. compile a new revision;
6. apply and verify again.

Rule of thumb:

- prefer narrow exceptions over globally disabling CRS.

### Site Breaks After WAF Enablement

Safe recovery:

1. temporarily disable WAF for the affected site;
2. compile and apply the revision;
3. confirm recovery;
4. re-enable WAF in `Detection`;
5. move back toward enforcement gradually.

### Legitimate Users Receive `429`

Check:

- the current `RateLimitPolicy`;
- the Anti-DDoS L7 override;
- trusted proxies and real client IP visibility.

Not every `429` is a WAF issue. Many come from rate-limit or Anti-DDoS controls.

### Operator Self-Lockout

Check:

- `AccessPolicy`;
- denylist and allowlist rules;
- office, VPN, and management source ranges.

Before applying a manual ban, always confirm:

- the exact IP;
- the CIDR width;
- whether the rule affects management access.

## Recommended Change Flow

For WAF, access, and rate-limit policies, use the same operational order:

1. change the policy in UI or API;
2. compile a new revision;
3. apply the revision;
4. verify site behavior;
5. roll back to the last known-good revision if degradation appears.

Helpful discipline:

- make one meaningful change at a time;
- avoid packing unrelated changes into one revision;
- validate immediately after apply.

## Observability Without External SIEM

Use:

- UI `Activity` and related observability screens;
- `GET /core-docs/api/audit`
- `GET /core-docs/api/events`
- `GET /core-docs/api/requests`
- `GET /core-docs/api/reports/revisions`

These sources help answer:

- who changed a policy;
- which revision was applied;
- whether blocks started after the change;
- how system behavior evolved over time.

## When To Roll Back Immediately

Roll back to the last stable revision when:

- the site starts returning widespread false `403`;
- a critical user flow degrades after apply;
- Anti-DDoS or rate-limit controls block legitimate traffic;
- `/healthz` or `/healthcheck` show degradation after rollout.

After rollback:

- record the reason;
- make the next change narrower;
- release the fix as a separate revision.

