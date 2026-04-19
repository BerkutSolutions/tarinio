# Сквозная Валидация Stage 1

Дата: `2026-04-01`

Документ фиксирует результаты operator-проверки Stage 1 для single-node развертывания.

## Область проверки

- чистый `docker compose` старт;
- onboarding bootstrap;
- compile/apply первой ревизии;
- активация runtime;
- HTTPS handoff;
- вход оператора и audit trail;
- корректная UI routing surface без `.html`;
- bootstrap L4 guard без шумных warning.

## Среда

- Windows host;
- compose stack из `deploy/compose/`;
- проверка выполнялась после fresh reset:
  `docker compose down --remove-orphans`

## Матрица проверки

| Сценарий | Ожидаемый результат | Фактический результат | Статус |
| --- | --- | --- | --- |
| Первый вход | `http://localhost` открывает onboarding | onboarding page открывается корректно | PASS |
| Bootstrap первого администратора | первый user создаётся только через bootstrap flow | `POST /api/auth/bootstrap` создаёт `admin` | PASS |
| Site + Upstream + TLS | onboarding создаёт базовую конфигурацию сайта | persisted state содержит сайт, upstream, certificate и TLS binding | PASS |
| Compile + apply | первая ревизия успешно собирается и применяется | `rev-000001` и `apply-rev-000001` завершаются успешно | PASS |
| Active runtime | runtime видит активную ревизию | `current_active_revision_id=rev-000001` | PASS |
| HTTPS runtime | runtime отвечает по `443` development-сертификатом | `HTTP/1.1 200 OK` подтверждён | PASS |
| Login after setup | оператор логинится стандартным путём | `POST /api/auth/login` успешен | PASS |
| Audit trail | ключевые действия отражены в audit | bootstrap, site create, compile, apply и login записаны | PASS |
| Clean routes | `/login` и `/dashboard` работают без `.html` | чистые маршруты доступны, legacy URLs редиректят | PASS |
| L4 guard bootstrap | runtime стартует с установленными guard rules | warning noise отсутствует, правила на месте | PASS |

## Итог

Stage 1 в проверяемой среде проходит базовый product flow:
- пустой стек -> onboarding;
- onboarding создаёт первого администратора;
- onboarding готовит базовый сайт и TLS;
- compile/apply активирует первую ревизию;
- runtime становится рабочим;
- HTTPS и операторский login подтверждены.

## Ограничения Stage 1

- используется single-node модель;
- development certificate flow не равен публичному production issuance;
- anti-DDoS слой остаётся baseline-first, без distributed control plane.
