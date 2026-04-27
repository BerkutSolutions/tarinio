# External Production Site Behind TARINIO WAF

This runbook defines the reference integration path for an internet-facing production website protected by a standalone TARINIO WAF.

## Reference Topology

```text
Internet clients
  -> TARINIO runtime edge (80/443)
    -> upstream production service (http/https)
```

Control-plane is a management plane and must not be exposed as a public edge endpoint.

## Integration Requirements

1. Edge exposure
   - Publish only runtime edge ports required for production traffic (`80/443`).
   - Keep management UI/API on private network or loopback-bound listener.
2. Upstream routing
   - Configure `Site` + `Upstream` with explicit scheme/host/port.
   - Use reverse proxy mode (`WAF_SITE_USE_REVERSE_PROXY=true`).
3. Client IP trust chain
   - Configure trusted proxies correctly.
   - Preserve `X-Forwarded-For`, `X-Real-IP`, and `X-Forwarded-Proto` semantics.
4. TLS boundary
   - Choose termination model explicitly:
     - TLS termination at WAF and upstream over HTTP.
     - TLS re-encryption from WAF to upstream over HTTPS with SNI when required.
5. Health and fail-safe
   - Monitor `/healthz` and `/healthcheck`.
   - Use revision rollback as the default recovery mechanism for unsafe policy changes.

## Recommended Deployment Defaults

- Public network:
  - `runtime`: `80/443`
- Private or loopback-only:
  - `ui`
  - `control-plane`
  - data backends (`PostgreSQL`, `Redis`, `OpenSearch`, `ClickHouse`, `Vault`)

## Validation Checklist

1. `docker compose ps` (or equivalent) confirms only expected public edge ports are mapped.
2. External request reaches WAF and is routed to the expected upstream.
3. Forwarded client IP headers are correct in requests/events.
4. TLS/header policy on the public endpoint matches approved baseline.
5. Rollback to last known-good revision works without manual runtime edits.

## Pre-ASV Validation

Before external ASV scanning, execute perimeter preflight and archive artifacts:

```bash
TARGET_HOST=<public-waf-hostname> \
TARGET_HTTP_PORT=80 \
TARGET_HTTPS_PORT=443 \
EXPECTED_OPEN_PORTS=80,443 \
PORT_PROBE_SET=22,80,443,8080,8443,9200 \
./scripts/pci-preflight-perimeter.sh
```

If preflight fails, roll back to the last known-good revision first and re-run checks before requesting a new scan window.
