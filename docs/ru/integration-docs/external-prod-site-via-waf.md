# Подключение внешнего production-сайта через TARINIO WAF

Этот runbook задаёт эталонную схему интеграции internet-facing сайта, который защищается standalone-развёрнутым TARINIO WAF.

## Эталонная топология

```text
Клиенты из Интернета
  -> runtime edge TARINIO (80/443)
    -> upstream production-сервис (http/https)
```

Control-plane остаётся management-контуром и не должен использоваться как публичная edge-точка.

## Требования к интеграции

1. Edge-экспозиция
   - Публикуются только runtime-порты production-трафика (`80/443`).
   - Management UI/API держатся в приватной сети или на loopback listener.
2. Маршрутизация в upstream
   - Настроить `Site` + `Upstream` с явными `scheme/host/port`.
   - Использовать reverse proxy режим (`WAF_SITE_USE_REVERSE_PROXY=true`).
3. Цепочка доверия client IP
   - Корректно настроить trusted proxies.
   - Сохранять семантику `X-Forwarded-For`, `X-Real-IP`, `X-Forwarded-Proto`.
4. TLS-граница
   - Явно выбрать модель termination:
     - TLS завершается на WAF, в upstream идёт HTTP.
     - TLS re-encryption до upstream по HTTPS с SNI при необходимости.
5. Health и fail-safe
   - Мониторить `/healthz` и `/healthcheck`.
   - Для отката использовать revision rollback, без ручной правки runtime.

## Рекомендуемые deployment-defaults

- Публичная сеть:
  - `runtime`: `80/443`
- Приватная сеть или loopback-only:
  - `ui`
  - `control-plane`
  - backend-сервисы (`PostgreSQL`, `Redis`, `OpenSearch`, `ClickHouse`, `Vault`)

## Чек-лист проверки

1. `docker compose ps` (или аналог) подтверждает, что снаружи опубликованы только ожидаемые edge-порты.
2. Внешний запрос доходит до WAF и маршрутизируется в ожидаемый upstream.
3. Forwarded-заголовки с IP клиента корректны в `Requests/Events`.
4. TLS/header-политика на публичной точке соответствует утверждённому baseline.
5. Откат на последнюю стабильную ревизию работает без ручной правки runtime.

## Проверка перед ASV-сканированием

Перед внешним ASV-сканом запускайте perimeter preflight и архивируйте артефакты:

```bash
TARGET_HOST=<public-waf-hostname> \
TARGET_HTTP_PORT=80 \
TARGET_HTTPS_PORT=443 \
EXPECTED_OPEN_PORTS=80,443 \
PORT_PROBE_SET=22,80,443,8080,8443,9200 \
./scripts/pci-preflight-perimeter.sh
```

Если preflight завершился с fail, сначала выполняется rollback на last known-good ревизию и повторная проверка, и только потом запрашивается новое scan-окно.
