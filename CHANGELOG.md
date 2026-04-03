# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [1.0.0] - 2026-04-03

### Release highlights
- TARINIO 1.0.0 release: standalone self-hosted WAF with revision-driven configuration lifecycle.
- Unified product branding as `Berkut Solutions - TARINIO`.
- Established RU/EN release-synced documentation baseline.

### Added
- Dedicated `Anti-DDoS` UI section and `/anti-ddos` route shell page.
- Persistent global Anti-DDoS settings model and API (`GET/PUT /api/anti-ddos/settings`) with audit trail.
- Operator runbook for Anti-DDoS tuning and rollback: `docs/operators/anti-ddos-runbook.md`.

### Changed
- Runtime L4 guard now applies limits from revision artifact `l4guard/config.json` generated from persisted Easy profile settings.
- Apply pipeline now consumes Anti-DDoS snapshot settings to override L4 guard artifact and optional global L7 rate-limit compile output.
- Easy site validation flow switched to i18n keys (limiter/auth/method/status-code checks) with RU/EN dictionary alignment.
- Product/about copy updated to reflect TARINIO scope (WAF/CRS + L4/L7 + revision workflow).
- Version source centralized in `control-plane/internal/appmeta/meta.go` and exposed via `/api/app/meta`.

### Compatibility and fallback behavior
- Environment variables for L4 guard limits remain supported as fallback overrides when revision file config is absent.

### Documentation
- Added SCC-style RU/EN documentation structure (`docs/ru`, `docs/eng`) with baseline `1.0.0`.
- Updated root README files to reflect TARINIO product positioning and screenshot gallery.
- Added bilingual quick start guides: `QUICKSTART.md` and `QUICKSTART.en.md`.
