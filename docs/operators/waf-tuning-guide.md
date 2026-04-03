# MVP WAF Tuning Guide

This guide is for a Stage 1 operator who already:

- started the stack through `S1-46`
- completed onboarding through `S1-47`
- has a working single-node WAF

This is not a ModSecurity theory document.

It is a practical guide for real operator tasks:
- safely enabling WAF
- diagnosing block decisions
- reducing false positives
- adjusting rate limiting
- avoiding self-inflicted lockouts

---

## 1. How WAF Works In This Architecture

In this system, WAF behavior is not edited directly in runtime files.

The flow is:

```text
control-plane -> policies -> compile -> bundle -> runtime
```

That means:

- operators change `WAFPolicy`, `AccessPolicy`, and `RateLimitPolicy`
- control-plane creates a new `Revision`
- compiler generates a deterministic bundle
- runtime loads only the active compiled bundle

Important rule:

Do not manually edit:
- NGINX config inside runtime
- ModSecurity config inside runtime
- files under the active bundle path

Those edits will be overwritten by the next compile/apply cycle and break the architecture.

All changes must go through:
- UI
- control-plane API

---

## 2. WAF Modes

Stage 1 uses two practical WAF modes:

### Detection mode

Use detection mode when:
- introducing WAF on a new site
- testing a new policy
- diagnosing false positives

Behavior:
- rules are evaluated
- suspicious requests are observed
- enforcement is reduced compared to blocking mode

This is the safest starting point.

### Prevention mode

Use prevention mode when:
- site traffic is already understood
- the policy has been tested
- false positives are under control

Behavior:
- WAF actively blocks matching traffic
- operator impact is immediate if tuning is wrong

Recommended usage:

- staging or initial rollout: `Detection`
- production after validation: `Prevention`

---

## 3. Safe Default Starting Point

For a new site, start with:

- `WAFPolicy.enabled = true`
- `CRS enabled = true`
- `mode = detection`

Recommended sequence:

1. enable WAF in detection mode
2. compile a new revision
3. apply the revision
4. observe activity and reports
5. only then move to prevention mode

Do not start with:

- custom rule includes before baseline works
- aggressive rate limiting before normal traffic is understood
- manual denylist entries without checking your own operator IP

---

## 4. Common Real-World Problems

### False positive

Symptom:

- user receives `403`
- site is up
- apply succeeded

What to do:

1. Open UI `Logs / Activity`
2. Find the event around the time of the block
3. Identify:
   - site
   - action context
   - related revision or rollout change
4. Check which WAF change was recently applied
5. Adjust the policy through control-plane:
   - add or refine a `rule override`
   - add a `custom_rule_include` when a targeted exception is required
6. Compile a new revision
7. Apply it
8. Re-test the request

Safe rule:

Prefer narrow exceptions over broad disabling.

Do not:

- disable CRS globally just because one request path breaks
- edit runtime files manually to “quick fix” a false positive

### Site breaks after enabling WAF

Symptom:

- site worked before WAF
- after apply, users see blocks or broken flows

What to do:

1. Set `WAFPolicy.enabled = false`
2. Compile a new revision
3. Apply it
4. Confirm the site recovers
5. Re-enable WAF in `Detection` mode
6. Reintroduce policy gradually

Recommended recovery order:

1. disabled
2. enabled + detection
3. enabled + prevention

This is safer than trying to guess multiple overrides under pressure.

### Rate limit blocks legitimate users

Symptom:

- users see `429`
- site is otherwise healthy
- WAF may not be the real cause

What to do:

1. Review the current `RateLimitPolicy`
2. Reduce aggressiveness:
   - lower enforcement pressure by increasing `burst`
   - adjust `requests_per_second` to a realistic level
3. Check whether requests are arriving through proxies
4. Verify trusted proxy handling is correct

If proxy topology is wrong, many users may appear as one client IP.

Do not treat every `429` as a WAF issue.

In Stage 1, rate limiting is a separate policy concern.

### Manual IP ban causes operator lockout

Symptom:

- access suddenly disappears for an operator
- a deny entry was recently added

What to do:

1. Check the site `AccessPolicy`
2. Review `denylist`
3. Confirm your own current IP is not blocked
4. Remove the deny entry if needed
5. Compile and apply again

Safe habit:

Before adding a manual ban:
- confirm the target IP exactly
- avoid broad CIDR bans unless necessary
- double-check office/VPN/admin source ranges

---

## 5. Safe Policy Change Flow

Use the same sequence for WAF, access, and rate-limit changes:

1. change the policy in UI or API
2. create a new revision
3. apply the revision
4. verify:
   - reports
   - activity
   - site behavior
5. if the change causes problems:
   - roll back to the last known good revision

Recommended operator habit:

- make one focused change at a time
- apply one revision at a time
- verify immediately after apply

Avoid batching unrelated policy changes into one revision when diagnosing a problem.

---

## 6. How To Read The System Without SIEM

Stage 1 does not include raw-log search or SIEM pipelines.

Use these sources instead:

### UI `Logs / Activity`

Use this as the main operator view for:

- recent audit activity
- rollout activity
- action history around a change

### `/api/audit`

Use this when you need to answer:

- who changed a policy
- who triggered compile/apply
- when a login or admin action happened

This is the source of truth for operator actions.

### `/api/reports/revisions`

Use this when you need to answer:

- did apply succeed
- which revisions failed
- was rollback performed
- which sites are affected most often

This is the main apply/reporting view in Stage 1.

Important limitation:

Stage 1 visibility is good enough for operator workflows, but not for deep forensic analysis.

---

## 7. L4 Anti-DDoS And WAF Interaction

These are different layers.

### L4 guard

The L4 guard:

- runs before HTTP and TLS parsing
- uses kernel/network filtering
- drops excessive traffic early

Its job is to protect:

- NGINX CPU
- ModSecurity CPU
- the runtime container itself

### WAF

The WAF:

- inspects requests that already reached the runtime
- applies compiled HTTP-layer security policy
- is not a replacement for transport/network filtering

Critical rule:

`WAF does not protect the WAF from load.`

More directly:

`firewall protects WAF`

If CPU is high under abusive traffic:

- first inspect L4 guard behavior
- then inspect WAF/rate-limit policy

Do not try to solve transport-layer floods by only tuning WAF rules.

---

## 8. Stage 1 Limits

Be explicit about current limits:

- no raw access-log archive
- no full-text search
- no SIEM integration
- limited rule-level visibility
- no distributed protection
- no HA or multi-node setup
- only baseline L4 anti-DDoS protection

This means:

- diagnosis is based on audit, reports, and UI activity
- advanced incident analysis may require Stage 2+ improvements
- operators should prefer incremental, low-risk policy changes

---

## 9. Practical Working Pattern

For a new protected site:

1. enable WAF with CRS in detection mode
2. compile and apply
3. browse the site normally
4. test critical user flows
5. watch activity and revision reports
6. fix false positives narrowly
7. move to prevention mode only after validation

For an incident:

1. identify whether the issue is:
   - WAF block
   - rate limit
   - manual deny
   - failed apply
2. confirm recent changes through audit and revision reports
3. either:
   - narrow the policy
   - disable the policy temporarily
   - rollback the revision

The safest emergency action is usually:

- revert to the last known good revision

---

## 10. Done Means

This guide is successful if an operator can:

- enable WAF safely
- diagnose why requests are blocked
- reduce false positives
- recover from a bad policy change

without reading the source code first.
