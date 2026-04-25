# Kubernetes Secrets And Config (Lab Baseline)

Этот документ закрывает базовый слой задачи по секретам и конфигурации в Kubernetes.

## Разделение ответственности

В lab-контуре:

- несекретные параметры хранятся в `ConfigMap`:
  - `deploy/lab-k8s-terraform/k8s/manifests/01-configmap.yaml`
- секреты хранятся в `Secret`:
  - `deploy/lab-k8s-terraform/k8s/manifests/02-secrets.yaml`

## Безопасные дефолты для lab

- не использовать `CHANGE_ME` значения;
- генерировать случайные значения для:
  - `POSTGRES_PASSWORD`
  - `CONTROL_PLANE_SECURITY_PEPPER`
  - `WAF_RUNTIME_API_TOKEN`
  - `OPENSEARCH_PASSWORD`
- Vault отключен в текущем baseline (`VAULT_ENABLED=false`) ради упрощения первого старта.

## Инициализация секретов

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1
```

Скрипт генерирует `02-secrets.yaml` с случайными значениями и корректным `POSTGRES_DSN`.

## Ротация секретов

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/rotate-secrets.ps1
```

Скрипт:

1. пере-генерирует `02-secrets.yaml`;
2. применяет обновленный Secret;
3. перезапускает `control-plane` и `runtime`;
4. дожидается `rollout status`.

## Проверка перед apply

`apply-lab.ps1` проверяет, что в `02-secrets.yaml` нет `CHANGE_ME`.
Если шаблонные значения остались, deployment будет остановлен до исправления.

