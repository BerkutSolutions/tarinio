---
sidebar_position: 3
---

# Развертывание в Kubernetes

Шпаргалка для production-развертывания TARINIO v1.3.5+ в Kubernetes-кластере.

## Минимальные требования

- Kubernetes `1.28+`;
- `kubectl` с доступом к production context;
- выделенный namespace (например, `tarinio`);
- отдельные `Secret` для production;
- внешний PostgreSQL или устойчивый stateful-контур данных;
- Vault (или Kubernetes Secrets) для хранения mTLS-сертификатов.

## Компоненты для деплоя

| Компонент | Тип | Примечание |
|---|---|---|
| `control-plane` | Deployment | API + компилятор |
| `runtime` | DaemonSet / Deployment | nginx/OpenResty, требует NET_ADMIN |
| `tarinio-sentinel` | Deployment | DDoS-модель, читает nginx access.log |
| `postgres` | StatefulSet / внешний | Основная БД |
| `opensearch` | StatefulSet / внешний | Индекс запросов |
| `vault` | StatefulSet / внешний | Обязателен для mTLS |

## Базовый порядок работ

1. Подготовить namespace и секреты.
2. Применить манифесты компонентов.
3. Дождаться готовности pod/service/ingress.
4. Проверить health и первый вход.
5. Зафиксировать smoke-проверки в release-процессе.

## Команды (sh)

```bash
kubectl create namespace tarinio
kubectl -n tarinio apply -f secrets.yaml
kubectl -n tarinio apply -f manifests/
kubectl -n tarinio rollout status deploy/control-plane --timeout=180s
kubectl -n tarinio rollout status deploy/runtime --timeout=180s
kubectl -n tarinio get pods,svc,ingress
```

## Команды (PowerShell)

```powershell
kubectl create namespace tarinio
kubectl -n tarinio apply -f secrets.yaml
kubectl -n tarinio apply -f manifests/
kubectl -n tarinio rollout status deploy/control-plane --timeout=180s
kubectl -n tarinio rollout status deploy/runtime --timeout=180s
kubectl -n tarinio get pods,svc,ingress
```

## Post-deploy проверки

- `GET /healthz` возвращает стабильный `200`;
- доступен `/login`;
- `GET /core-docs/api/app/meta` возвращает ожидаемую версию;
- compile/apply проходит без ошибок;
- в UI появляются events/requests/audit сигналы;
- sentinel pod запущен и публикует `/out/adaptive.json`.

## Production рекомендации

- не использовать значения `CHANGE_ME` и дефолтные секреты;
- ограничить доступ к control-plane сетевыми политиками;
- вынести TLS-терминацию и сертификаты в управляемый контур;
- mTLS-сертификаты хранить в Vault или Kubernetes Secrets с ограниченным RBAC;
- runtime требует `NET_ADMIN` + `NET_RAW` capabilities — учитывать в PodSecurityPolicy/PodSecurityAdmission;
- регулярно проверять backup/restore сценарий.
