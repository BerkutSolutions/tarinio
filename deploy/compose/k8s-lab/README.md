# TARINIO Compose Profile: `k8s-lab`

`k8s-lab` is an isolated integration profile to validate WAF routing against Kubernetes Ingress.

This profile is intentionally not part of `default`:

- `default` remains standalone runtime-only.
- `k8s-lab` adds a dedicated `k3s` container for Kubernetes workloads.

## Main guarantees

- No impact on existing local/preprod stacks.
- Random project and random resource names per smoke run.
- Automatic cleanup (`down -v --remove-orphans`) by default.

## Included artifacts

- Overlay compose: `deploy/compose/k8s-lab/docker-compose.yml`
- K8s manifest: `deploy/compose/k8s-lab/manifests/lab-apps.yaml`
- Smoke script: `scripts/smoke-k8s-ingress-waf.ps1`

## Integration model

1. WAF stack starts from `deploy/compose/default/docker-compose.yml`.
2. `k3s` is added via `deploy/compose/k8s-lab/docker-compose.yml`.
3. Test apps and ingress are applied from `lab-apps.yaml`.
4. WAF routes requests to `k3s` ingress on `http://k3s:80`.
5. Traffic verification uses explicit `Host` header for 3 virtual hosts.

## Operational note

Local `kubectl` is not required. All Kubernetes commands are executed inside the `k3s` container, and the smoke script auto-detects whether to use `kubectl` or `k3s kubectl`.
