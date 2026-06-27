---
sidebar_position: 4
---

# Terraform Deployment

Production cheatsheet for provisioning and rolling out TARINIO v1.3.5+ with Terraform.

## What The Terraform Layer Should Cover

- infrastructure provider resources (cloud/on-prem);
- Kubernetes cluster and network dependencies;
- environment secrets/config wiring;
- Kubernetes/Helm resource deployment;
- lifecycle operations: `plan`/`apply`/`destroy`;
- Vault resources (policies, mount points) for mTLS certificate management.

## Components Managed By Terraform

| Component | Notes |
|---|---|
| `control-plane` | Deployment + Service |
| `runtime` | DaemonSet / Deployment, NET_ADMIN capabilities |
| `tarinio-sentinel` | Deployment, reads nginx access.log |
| `postgres` | Managed DB or StatefulSet |
| `opensearch` | Managed or StatefulSet |
| `vault` | Managed Vault or StatefulSet, required for mTLS |

## Recommended Pipeline

1. `terraform fmt` and `terraform validate`.
2. `terraform plan` and save the plan artifact.
3. Plan review and approval.
4. `terraform apply` from approved plan.
5. Post-deploy smoke validation.

## Commands (sh)

```bash
cd deploy/terraform
terraform init
terraform fmt -check -recursive
terraform validate
terraform plan -out=tfplan.bin
terraform apply tfplan.bin
```

## Commands (PowerShell)

```powershell
Set-Location deploy/terraform
terraform init
terraform fmt -check -recursive
terraform validate
terraform plan -out=tfplan.bin
terraform apply tfplan.bin
```

## Security And Maturity

- use remote backend with locking for state;
- isolate environments (`dev/stage/prod`) via vars/workspaces;
- do not keep plaintext secrets in `tfvars`;
- use short-lived CI credentials;
- manage mTLS certificates and CA via Vault Terraform provider — do not store them in state.

## Post-Apply Verification

- `/healthz` and `/login` are reachable;
- `/core-docs/api/app/meta` reports expected version;
- compile/apply succeeds in UI/API;
- telemetry/events are visible after traffic;
- sentinel published `adaptive.json` (non-empty);
- Vault is unsealed and policies are applied.
