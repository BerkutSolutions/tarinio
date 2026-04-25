# Deploy Lab Runbook (RU)

## Область применения

Этот runbook описывает `experimental/lab` цикл деплоя TARINIO в Kubernetes + Terraform.

Критерии перехода из `experimental/lab` в `supported`:

- `.work/SUPPORTED_PROFILE_CRITERIA.md`

## Быстрый старт

1. Preflight:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/preflight.ps1`
2. Bootstrap кластера:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1`
3. Инициализация секретов:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1`
4. Terraform apply:
   - `cd deploy/lab-k8s-terraform/terraform`
   - `.\terraform.exe init`
   - `.\terraform.exe apply -auto-approve -var-file=terraform.tfvars.example`
5. Smoke:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1`

## HA-профиль

1. Применить HA overlay:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-ha-control-plane.ps1`
2. Прогнать HA smoke:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-ha-control-plane.ps1`

## Профиль OpenSearch/ClickHouse

1. Применить профиль:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-opensearch-clickhouse.ps1`
2. Настроить runtime logging:
   - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/configure-opensearch-clickhouse.ps1`

## Teardown

- `cd deploy/lab-k8s-terraform/terraform`
- `.\terraform.exe destroy -auto-approve -var-file=terraform.tfvars.example`

## Troubleshooting

- `terraform init` не проходит:
  - проверьте, что `terraform.exe` лежит в `deploy/lab-k8s-terraform/terraform`
  - запускайте команды из каталога terraform
- pod не готов:
  - `kubectl -n tarinio-lab get pods`
  - `kubectl -n tarinio-lab logs deploy/control-plane --tail=200`
  - `kubectl -n tarinio-lab logs deploy/runtime --tail=200`
- smoke падает на API:
  - проверьте сервисы: `kubectl -n tarinio-lab get svc`
  - проверьте доступность `/healthz` и `/readyz`

