# Toolchain Bootstrap (Beginner)

Этот документ закрывает базовую часть задачи по подготовке среды для новичка.

## Минимальные требования

- Docker Desktop (или Docker Engine)
- `kubectl`
- `helm`
- `terraform`
- один локальный Kubernetes runtime:
  - `kind` (рекомендуется для первого старта), или
  - `k3d`

## Быстрый preflight

PowerShell:

```powershell
pwsh -File deploy/lab-k8s-terraform/scripts/preflight.ps1
```

Если `pwsh` недоступен:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/preflight.ps1
```

## Быстрый старт с kind

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1
```

Скрипт:

1. проверяет наличие `kind`;
2. создает кластер `tarinio-lab` (если его еще нет);
3. создает namespace `tarinio-lab`.

## Что дальше

После bootstrap:

1. собрать и загрузить образы в кластер (см. `deploy/lab-k8s-terraform/k8s/README.md`);
2. применить манифесты;
3. запустить базовый smoke.

