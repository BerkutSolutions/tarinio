# Руководство по устранению неполадок

Эта страница относится к текущей ветке документации.

## Назначение

Этот документ связывает типовые симптомы с вероятными причинами и первыми действиями по восстановлению.

## `/healthz` не healthy

Проверьте:

- состояние контейнеров;
- логи control-plane;
- подключение к PostgreSQL;
- подключение к Redis в HA режиме.

Первые действия:

- проверить environment variables;
- убедиться, что PostgreSQL и Redis доступны;
- посмотреть, не было ли недавних upgrade / migration.

## Не работает вход

Проверьте:

- bootstrap credentials или состояние пользователя;
- session storage;
- cookie и HTTPS boundary;
- browser console и сетевые ответы.

Первые действия:

- убедиться, что admin user существует;
- проверить session persistence;
- проверить trusted proxy и host configuration.

## Не проходит compile или apply

Проверьте:

- `Revisions`;
- `Events`;
- состояние runtime;
- HA lock contention;
- логи control-plane.

Первые действия:

- убедиться, что одновременно не идёт другой apply;
- посмотреть текст ошибки;
- повторить после исправления конфигурации или runtime readiness.

## Runtime healthy, но трафик сломан

Проверьте:

- `Sites`;
- `Upstreams`;
- TLS bindings;
- runtime logs;
- `Requests`.

Первые действия:

- убедиться, что routing указывает на нужный upstream;
- проверить certificate bindings;
- прогнать запрос на простой known-good page.

## Слишком много `403` или `429`

Проверьте:

- CRS mode и exclusions;
- rate-limit policies;
- настройки Anti-DDoS;
- `Events`, `Requests` и `Bans`.

Первые действия:

- определить, кто именно блокирует: WAF, rate-limit или Anti-DDoS;
- временно ослабить самый узкий policy;
- сохранить пример проблемного запроса до больших изменений.
