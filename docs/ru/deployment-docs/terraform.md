---
sidebar_position: 4
---

# Развертывание через Terraform

Шпаргалка по production-оркестрации TARINIO через Terraform.

## Что должно быть в Terraform-слое

- провайдер инфраструктуры (cloud/on-prem);
- кластер Kubernetes и сетевые зависимости;
- секреты и конфигурация окружения;
- применение Kubernetes/Helm ресурсов;
- управляемый lifecycle: `plan`/`apply`/`destroy`.

## Рекомендуемый pipeline

1. `terraform fmt` и `terraform validate`.
2. `terraform plan` с сохранением плана.
3. Ревью плана.
4. `terraform apply` из согласованного плана.
5. Post-deploy smoke.

## Команды (sh)

```bash
cd deploy/terraform
terraform init
terraform fmt -check -recursive
terraform validate
terraform plan -out=tfplan.bin
terraform apply tfplan.bin
```

## Команды (PowerShell)

```powershell
Set-Location deploy/terraform
terraform init
terraform fmt -check -recursive
terraform validate
terraform plan -out=tfplan.bin
terraform apply tfplan.bin
```

## Безопасность и зрелость

- хранить state в удаленном backend с блокировкой;
- использовать разделение окружений (`dev/stage/prod`) через переменные и workspace;
- не хранить секреты в открытом виде в `tfvars`;
- использовать короткоживущие учетные данные для CI/CD.

## Пост-проверка после apply

- доступность `/healthz` и `/login`;
- корректная версия в `/core-docs/api/app/meta`;
- успешный compile/apply в UI/API;
- наличие telemetry/events после подачи трафика.
