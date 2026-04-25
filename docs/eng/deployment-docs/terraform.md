---
sidebar_position: 4
---

# Terraform Deployment

Production cheatsheet for provisioning and rolling out TARINIO with Terraform.

## What The Terraform Layer Should Cover

- infrastructure provider resources (cloud/on-prem);
- Kubernetes cluster and network dependencies;
- environment secrets/config wiring;
- Kubernetes/Helm resource deployment;
- lifecycle operations: `plan`/`apply`/`destroy`.

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
- use short-lived CI credentials.

## Post-Apply Verification

- `/healthz` and `/login` are reachable;
- `/core-docs/api/app/meta` reports expected version;
- compile/apply succeeds in UI/API;
- telemetry/events are visible after traffic.
