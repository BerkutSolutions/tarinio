---
sidebar_position: 3
---

# Kubernetes Deployment

Production cheatsheet for deploying TARINIO on Kubernetes.

## Minimum Requirements

- Kubernetes `1.28+`;
- `kubectl` access to a production context;
- dedicated namespace (for example `tarinio`);
- production-grade secrets in `Secret` objects;
- external PostgreSQL or resilient stateful data layer.

## Baseline Flow

1. Create namespace and apply secrets.
2. Apply component manifests.
3. Wait for deploy/service/ingress readiness.
4. Verify health and first login.
5. Keep smoke checks in release gates.

## Commands (sh)

```bash
kubectl create namespace tarinio
kubectl -n tarinio apply -f secrets.yaml
kubectl -n tarinio apply -f manifests/
kubectl -n tarinio rollout status deploy/control-plane --timeout=180s
kubectl -n tarinio rollout status deploy/runtime --timeout=180s
kubectl -n tarinio get pods,svc,ingress
```

## Commands (PowerShell)

```powershell
kubectl create namespace tarinio
kubectl -n tarinio apply -f secrets.yaml
kubectl -n tarinio apply -f manifests/
kubectl -n tarinio rollout status deploy/control-plane --timeout=180s
kubectl -n tarinio rollout status deploy/runtime --timeout=180s
kubectl -n tarinio get pods,svc,ingress
```

## Post-Deploy Validation

- `GET /healthz` is consistently `200`;
- `/login` is available;
- `GET /core-docs/api/app/meta` returns expected version;
- compile/apply succeeds;
- events/requests/audit data appears in UI.

## Production Guidance

- never keep default secrets;
- restrict control-plane access with network policy/security groups;
- use managed TLS termination/certificate rotation;
- regularly test backup/restore.
