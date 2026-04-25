# Deploy Lab Runbook (EN)

## Scope

This runbook covers the `experimental/lab` Kubernetes + Terraform deployment cycle for TARINIO.

For transition requirements from `experimental/lab` to `supported`, see:

- `.work/SUPPORTED_PROFILE_CRITERIA.md`

## Quick Start

1. Preflight:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/preflight.ps1`
2. Bootstrap cluster:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1`
3. Initialize secrets:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1`
4. Terraform apply:
   - `cd deploy/lab-k8s-terraform/terraform`
   - `.\terraform.exe init`
   - `.\terraform.exe apply -auto-approve -var-file=terraform.tfvars.example`
5. Smoke:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1`

## HA Profile

1. Apply HA overlay:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-ha-control-plane.ps1`
2. HA smoke:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-ha-control-plane.ps1`

## OpenSearch/ClickHouse Profile

1. Apply profile:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-opensearch-clickhouse.ps1`
2. Configure runtime logging:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/configure-opensearch-clickhouse.ps1`

## Teardown

- `cd deploy/lab-k8s-terraform/terraform`
- `.\terraform.exe destroy -auto-approve -var-file=terraform.tfvars.example`

## Troubleshooting

- `terraform init` fails:
  - ensure `terraform.exe` exists in `deploy/lab-k8s-terraform/terraform`
  - run from the terraform directory
- pods not ready:
  - `kubectl -n tarinio-lab get pods`
  - `kubectl -n tarinio-lab logs deploy/control-plane --tail=200`
  - `kubectl -n tarinio-lab logs deploy/runtime --tail=200`
- smoke failure on API:
  - verify services: `kubectl -n tarinio-lab get svc`
  - verify `/healthz` and `/readyz` endpoints


