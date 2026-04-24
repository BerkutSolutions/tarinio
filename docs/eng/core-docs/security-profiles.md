# Security Profiles

TARINIO 3.0.2 ships 5 built-in service security profiles:

1. `strict`
2. `balanced`
3. `compat`
4. `api`
5. `public-edge`

Profiles are baseline presets. Operators can keep the same profile and add custom overrides per service.

## Profile Intents

- `strict`: maximum protective posture for internet-facing services.
- `balanced`: default profile for mixed production traffic.
- `compat`: reduced-friction profile for legacy applications with higher false-positive risk.
- `api`: API-first posture with CORS and API Positive Security baseline enabled.
- `public-edge`: edge profile for publicly exposed entry points with stronger anti-bot and blacklist defaults.

## Operational Model

- Profile selection pre-fills recommended values.
- Manual edits are preserved on next saves.
- The profile label remains visible in service list and service settings.
- RAW editor includes `WAF_SITE_SERVICE_PROFILE` and can be used to set or keep profile explicitly.
