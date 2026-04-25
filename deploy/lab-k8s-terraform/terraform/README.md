# Terraform Baseline (Lab Orchestrator)

Этот раздел закрывает базовый слой задачи `Terraform-управление lab-контуром`.

## Что делает Terraform baseline

1. опционально bootstrap-ит local kind cluster;
2. подключает kube-context;
3. применяет Kubernetes manifests (`kubectl apply -k`);
4. поддерживает teardown через `terraform destroy` (`kubectl delete -k`).

## Важно

- это `experimental/lab` подход через `local-exec` и `kubectl`;
- перед `apply` секреты должны быть подготовлены:
  - `powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1`

## Быстрый запуск

```powershell
cd deploy/lab-k8s-terraform/terraform
terraform init
terraform plan -var-file=terraform.tfvars.example
terraform apply -var-file=terraform.tfvars.example
```

## Удаление lab-контура

```powershell
terraform destroy -var-file=terraform.tfvars.example
```

## Проверка без следов в системе

Если нужен только тест конфигурации без сохранения локальных terraform-артефактов:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/terraform/scripts/test-no-trace.ps1
```

Скрипт запускает `init/validate/plan`, а затем удаляет:

- `.terraform/`
- `.terraform.lock.hcl`
- временный файл плана

