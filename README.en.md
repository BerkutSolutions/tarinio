# Berkut Solutions - TARINIO

<p align="center">
  <img src="ui/app/static/logo500x300.png" alt="Berkut TARINIO logo" width="240">
</p>

[Russian version](README.md)

Berkut Solutions - TARINIO is a self-hosted web traffic protection and control platform (WAF/CRS + L4/L7 Anti-DDoS) with centralized change management through revisions.

Current version: `1.1.2`

## Product Overview

TARINIO intercepts inbound traffic before business applications, inspects requests with security policies, applies rate and anti-DDoS controls, and ships configuration updates through a controlled revision cycle:

- `compile` -> `validate` -> `apply`
- rollback to the last stable revision when operational risk appears

This provides reproducible changes, auditable operations, and one control loop for runtime and control-plane.

## Who It Is For

- Teams that need a self-hosted WAF perimeter without external SaaS dependencies.
- Infrastructure and security teams that need governed rollout of rules and traffic policies.
- Organizations that require traceable changes and predictable incident recovery.

## Business Value

- Reduces incident risk by filtering unwanted traffic before it reaches applications.
- Enables controlled release flow through revisions and centralized audit trail.
- Unifies operations across WAF, TLS, access policies, rate-limits, and Anti-DDoS.

## Functional Capabilities

- WAF/CRS request inspection and access policy enforcement.
- L4/L7 Anti-DDoS with per-site controls and global settings.
- Management of sites, upstreams, certificates, and TLS configurations.
- Events, request logs, audit stream, and revision reports.
- UI + API + CLI (`waf-cli`) for operator workflows.

## Security by Design

- Server-side zero-trust authorization on every endpoint.
- Session auth with support for 2FA (TOTP) and passkeys (WebAuthn).
- Self-hosted data boundary for runtime and control artifacts.
- Production hardening baseline: use non-default secrets, HTTPS, and restricted trusted proxies.

## Technical Profile

- Backend: Go.
- Runtime: NGINX + ModSecurity + OWASP CRS.
- Storage: PostgreSQL.
- Deployment: Docker / Docker Compose.

## Documentation

- Docs index: [`docs/README.md`](docs/README.md)
- Russian docs: [`docs/ru/README.md`](docs/ru/README.md)
- English docs: [`docs/eng/README.md`](docs/eng/README.md)
- CLI commands: [`docs/CLI_COMMANDS.md`](docs/CLI_COMMANDS.md)

## Quick Start

- AIO one-command install:
  - `curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh`
- Docker image:
  - `docker pull ghcr.io/berkutsolutions/tarinio:latest`
- Deploy: [`docs/eng/deploy.md`](docs/eng/deploy.md)
- Runbook: [`docs/eng/runbook.md`](docs/eng/runbook.md)
- Upgrade/rollback: [`docs/eng/upgrade.md`](docs/eng/upgrade.md)
- Compose profiles: [`deploy/compose/README.md`](deploy/compose/README.md)

## Screenshots

![Screenshot 1](ui/app/static/screen1.png)

![Screenshot 2](ui/app/static/screen2.png)

![Screenshot 3](ui/app/static/screen3.png)



