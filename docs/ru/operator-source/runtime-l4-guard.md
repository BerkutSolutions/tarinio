# Runtime L4 Guard

Документ фиксирует базовый L4 anti-DDoS слой runtime-контейнера.

## Назначение

L4 guard защищает runtime до того, как трафик дойдёт до:
- NGINX;
- TLS handshake;
- ModSecurity;
- HTTP parser и пользовательских политик.

Это инфраструктурный слой. Он не заменяет WAF-правила и не зависит от содержимого запросов.

## Что делает guard

При bootstrap runtime применяет детерминированные `iptables` правила, которые:
- ограничивают число одновременных TCP-соединений на один source IP через `connlimit`;
- ограничивают скорость новых TCP-соединений через `hashlimit`;
- выполняют `DROP` или `REJECT`, если лимиты превышены.

Дополнительно guard читает adaptive output и применяет временные per-IP правила:
- `throttle`;
- `drop`.

## Порядок запуска

Штатный bootstrap flow:

1. `waf-runtime-l4-guard bootstrap`
2. создание или обновление firewall chain
3. запуск `waf-runtime-launcher`
4. launcher читает `active/current.json` и стартует NGINX

Отдельный ручной шаг после старта runtime не требуется.

## Родительские цепочки

Guard умеет размещать jump rule в:
- `DOCKER-USER`;
- `INPUT`.

Режим `auto` является дефолтным:
- если в namespace есть `DOCKER-USER`, guard предпочитает её;
- иначе используется `INPUT`.

`DOCKER-USER` предпочтительнее в docker-aware сценариях, где трафик приходит к runtime через DNAT.

## Docker, DNAT и destination IP

В docker bridge режиме фильтрация в `DOCKER-USER` должна учитывать destination IP контейнера после DNAT.

Для этого используется:
- `WAF_L4_GUARD_DESTINATION_IP=<runtime-container-ip>`

Если значение не задано:
- guard пытается определить первый non-loopback IPv4 в текущем namespace.

## Защищаемые порты

Порты задаются через:
- `WAF_L4_GUARD_PORTS`

Значение по умолчанию:
- `80,443`

Список должен совпадать с реальным runtime exposure, иначе часть трафика останется вне защиты.

## Конфигурационные параметры

- `WAF_L4_GUARD_ENABLED`
  По умолчанию: `true`
- `WAF_L4_GUARD_CHAIN_MODE`
  Допустимые значения: `auto`, `docker-user`, `input`
- `WAF_L4_GUARD_CONN_LIMIT`
  Базовый лимит одновременных TCP-соединений
- `WAF_L4_GUARD_RATE_PER_SECOND`
  Базовый лимит новых TCP-соединений в секунду
- `WAF_L4_GUARD_RATE_BURST`
  Разрешённый burst для новых TCP-соединений
- `WAF_L4_GUARD_TARGET`
  `DROP` или `REJECT`
- `WAF_L4_GUARD_DESTINATION_IP`
  Явный IPv4-адрес runtime container для `DOCKER-USER`
- `WAF_L4_GUARD_ADAPTIVE_PATH`
  Путь к `adaptive.json` с динамическими throttle/drop-правилами

## Adaptive integration

Guard читает:
- базовый конфиг из `l4guard/config.json`;
- adaptive output из `l4guard-adaptive/adaptive.json`.

Если в adaptive output есть запись:
- `drop` — создаётся прямое правило на IP;
- `throttle` — создаётся отдельное hashlimit-правило для конкретного IP.

Таким образом:
- ревизия задаёт baseline;
- adaptive-модель добавляет временные меры;
- enforcement выполняет именно L4 guard.

## Контракт runtime

Guard не должен читать:
- control-plane storage;
- доменные объекты;
- revision metadata вне runtime bundle.

Он зависит только от:
- env/runtime config;
- доступности `iptables`;
- активной ревизии и её артефактов;
- adaptive output файла.

## Требуемые привилегии

Runtime должен иметь возможность управлять firewall rules в выбранном namespace.

Типичное требование:
- `NET_ADMIN`

Если `iptables` недоступен:
- bootstrap должен завершиться ошибкой;
- runtime не должен продолжать запуск так, будто защита была успешно активирована.
