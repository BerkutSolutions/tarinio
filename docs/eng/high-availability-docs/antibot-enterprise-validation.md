# Anti-Bot Enterprise Validation

Last full profile run: 2026-04-24.

This document captures the dedicated enterprise validation slice for anti-bot challenge behavior in TARINIO `3.0.1`.

## Validation Goal

Confirm that the anti-bot challenge loop:

1. Triggers on requests without verification cookie.
2. Correctly issues and reuses anti-bot cookie state.
3. Separates human-like and bot-like traffic by actual client behavior.

## Traffic Profiles (5 Actors)

- `human-regular`
- `human-hacker`
- `bot-curl`
- `bot-python-requests`
- `bot-scanner-direct-verify-no-jar`

## Acceptance Criteria

1. First request without anti-bot cookie is redirected to challenge.
2. Human-like profile that completes `/challenge/verify` with cookie persistence bypasses repeated challenge.
3. Bot-like profile without cookie persistence remains challenged.
4. Verification cookie stays stable across sequential human requests.

## What The Test Verifies

The test `ui/tests/e2e_antibot_profiles_test.go` validates:

- challenge redirect for a new client;
- challenge page availability;
- `/challenge/verify` redirect flow;
- post-verify behavior on the originally requested URL;
- presence of `waf_antibot_` cookie in persistent jar scenarios.

## Final Result

The anti-bot layer passes enterprise-style validation for human/bot separation:

- human traffic returns to normal navigation after verification;
- bot traffic without cookie persistence does not get a stable bypass;
- challenge contract and cookie persistence remain predictable.

This supports production-like use of anti-bot controls as part of TARINIO `3.0.1`.
