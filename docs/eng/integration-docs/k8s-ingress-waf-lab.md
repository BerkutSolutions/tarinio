# K8s Lab: WAF + Ingress + 3 Virtual Hosts

This lab validates a realistic integration path:

- TARINIO WAF runtime terminates client traffic.
- Kubernetes (`k3s`) runs inside a dedicated Docker container.
- Kubernetes Ingress routes by host to 3 different apps.
- WAF routes by site/upstream to Kubernetes Ingress.

The setup is isolated and does not require local Kubernetes installation.

## Why this profile exists

- It is separate from `default` because `default` is standalone runtime only.
- It uses random per-run project/resource names to avoid collisions with preprod/local stacks.
- It destroys all temporary containers, networks, and volumes by default.

## What is deployed

1. Base WAF stack from `deploy/compose/default/docker-compose.yml`
2. K8s overlay from `deploy/compose/k8s-lab/docker-compose.yml`
3. K8s test apps and ingress from `deploy/compose/k8s-lab/manifests/lab-apps.yaml`

Kubernetes hosts used in the lab:

- `example.com` -> `site-a`
- `domen.example.com` -> `site-b`
- `ex.example.com` -> `site-c`

Each app returns a unique marker string so routing is easy to verify.

## Important implementation details

- No local `kubectl` is required.
- All cluster operations run inside the `k3s` container via `docker exec ...`.
- The script auto-detects the working command prefix (`kubectl` or `k3s kubectl`) and uses it for all cluster operations.
- Upstreams are configured as `http://k3s:80` for lab stability.
- Revisions are applied sequentially (3 revisions, one per site).

## Smoke script

Script: `scripts/smoke-k8s-ingress-waf.ps1`

It performs:

1. Random smoke id generation.
2. Random free host ports for WAF runtime HTTP/HTTPS.
3. Per-run `.env` and compose override generation with random names.
4. `docker compose up -d --build` for base + `k8s-lab` overlay.
5. Control-plane readiness wait.
6. Optional bootstrap admin creation.
7. `k3s` readiness wait (via auto-detected kubectl prefix).
8. Manifest apply (via auto-detected kubectl prefix).
9. Rollout wait for `site-a`, `site-b`, `site-c`.
10. WAF site/upstream upsert for each host.
11. Compile + apply revision after each site (3 total).
12. Traffic checks through runtime with explicit `Host` header.
13. JSON report save.
14. Full cleanup (unless `-NoCleanup`).

## Prerequisites

- Docker Desktop is running.
- Shell can call `docker`.
- Sufficient local resources.

Recommended minimum:

- 6 vCPU
- 8 GB RAM
- 20 GB free disk

## Run

From repository root:

```powershell
./scripts/smoke-k8s-ingress-waf.ps1
```

Keep lab alive for manual troubleshooting:

```powershell
./scripts/smoke-k8s-ingress-waf.ps1 -NoCleanup
```

## Expected result

The script prints JSON and stores:

- `.work/k8s-ingress-waf-smoke/result-<smoke-id>.json`

Success criteria:

- `passed: true`
- 3 successful checks with expected markers
- 3 revision ids in `revisions`

Example markers:

- `K8S-LAB SITE A: example.com`
- `K8S-LAB SITE B: domen.example.com`
- `K8S-LAB SITE C: ex.example.com`

## Manual validation commands

Use runtime HTTP port from the result file:

```powershell
curl.exe -H "Host: example.com" http://127.0.0.1:<runtime_http_port>/
curl.exe -H "Host: domen.example.com" http://127.0.0.1:<runtime_http_port>/
curl.exe -H "Host: ex.example.com" http://127.0.0.1:<runtime_http_port>/
```

Each command must return its own marker string.

## Troubleshooting

1. Check Docker:
   - `docker version`
2. Check compose state:
   - `docker compose -p <project> ... ps`
3. Check k3s from container:
   - `docker exec <k3s-container> sh -lc "kubectl get nodes"` (or `k3s kubectl get nodes` if your image uses that form)
4. Check ingress objects:
   - `docker exec <k3s-container> sh -lc "kubectl -n waf-k8s-lab get pod,svc,ingress"`
5. Check WAF entities:
   - `waf-cli --json sites list`
   - `waf-cli --json upstreams list`
6. Check revision state:
   - compile returns `revision.id`
   - apply status is not `failed`
7. Check result file as source of truth for pass/fail.

## Cleanup

- Default behavior: automatic cleanup at the end.
- Manual cleanup for `-NoCleanup` runs:

```powershell
docker compose -p <project> `
  -f deploy/compose/default/docker-compose.yml `
  -f deploy/compose/k8s-lab/docker-compose.yml `
  down -v --remove-orphans
```

## Files in this scenario

- Compose overlay: `deploy/compose/k8s-lab/docker-compose.yml`
- K8s manifest: `deploy/compose/k8s-lab/manifests/lab-apps.yaml`
- Smoke script: `scripts/smoke-k8s-ingress-waf.ps1`
