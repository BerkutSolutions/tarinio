# Kubernetes Baseline (Single-Node Lab)

Этот раздел закрывает базовый слой задачи `Базовый Kubernetes deployment TARINIO (single-node lab)`.

## Что включено

- namespace `tarinio-lab`;
- базовые манифесты для:
  - `postgres`;
  - `opensearch`;
  - `control-plane`;
  - `runtime`;
  - `ui`;
- общий `ConfigMap` и пример `Secret` (`secrets.example.yaml`);
- разделение секретов/конфигурации и базовый процесс ротации:
  - `deploy/lab-k8s-terraform/k8s/SECRETS_AND_CONFIG.md`
- `kustomization.yaml` для пакетного применения;
- скрипты применения и двух уровней проверки:
  - базовый smoke;
  - расширенный e2e smoke (health, login, compile/apply, traffic, events/requests/bans checks).

## Важно

- это `experimental/lab`, не production профиль;
- для простоты в baseline отключен Vault (`VAULT_ENABLED=false`);
- значения секретов в `02-secrets.yaml` должны быть реальными (без `CHANGE_ME`);
- можно автоматически сгенерировать секреты через `init-secrets.ps1`.

## Быстрый запуск

1. Создать/подготовить кластер:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/scripts/bootstrap-kind.ps1
```

2. Собрать образы:

```powershell
docker build -f control-plane/Dockerfile -t tarinio-control-plane:lab .
docker build -f runtime/image/Dockerfile -t tarinio-runtime:lab .
docker build -f ui/Dockerfile -t tarinio-ui:lab .
```

3. Если используется `kind`, загрузить образы:

```powershell
kind load docker-image tarinio-control-plane:lab --name tarinio-lab
kind load docker-image tarinio-runtime:lab --name tarinio-lab
kind load docker-image tarinio-ui:lab --name tarinio-lab
```

4. Подготовить секрет:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/init-secrets.ps1
```

5. Применить манифесты:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-lab.ps1
```

6. Базовая проверка:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-basic.ps1
```

7. Расширенная e2e проверка:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/smoke-k8s-lab.ps1
```

## Ротация секретов

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/rotate-secrets.ps1
```

## Профиль OpenSearch + ClickHouse

Для lab-профиля с cold-tier:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/lab-k8s-terraform/k8s/scripts/apply-profile-opensearch-clickhouse.ps1
```

Что делает профиль:

1. добавляет `clickhouse` deployment/service/PVC;
2. применяет runtime logging profile через API:
   - hot backend: `opensearch`
   - cold backend: `clickhouse`
3. проверяет, что профиль реально активирован в `settings/runtime`.

