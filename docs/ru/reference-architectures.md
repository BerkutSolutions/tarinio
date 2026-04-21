# Референсные архитектуры

Эта страница относится к текущей ветке документации.

## Назначение

Эта страница даёт типовые deployment shapes, чтобы оператору не приходилось собирать topology с нуля.

## 1. Single-Node Starter

Использовать, когда:

- идёт знакомство с TARINIO;
- защищается небольшое число сервисов;
- HA пока не требуется.

Состав:

- один control-plane;
- один runtime;
- bundled PostgreSQL;
- при желании локальная observability.

## 2. HA Control-Plane

Использовать, когда:

- важна доступность administrative plane;
- нужны rolling upgrades без простоя API;
- нескольким операторам нужен стабильный доступ к control-plane.

Состав:

- `api-lb`
- `control-plane-a`
- `control-plane-b`
- shared PostgreSQL
- shared Redis
- runtime

## 3. Enterprise-Style Segmented Deployment

Использовать, когда:

- administrative access должен быть отделён;
- data services должны оставаться внутренними;
- observability выносится в отдельный monitoring contour.

## 4. Lab / Validation Architecture

Использовать, когда:

- проверяется failover;
- снимаются benchmark results;
- тестируются upgrade и migration workflows.

Состав:

- `deploy/compose/ha-lab`
