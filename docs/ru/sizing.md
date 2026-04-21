# Руководство по планированию ресурсов

Эта страница относится к текущей ветке документации.

## Назначение

Этот документ помогает прикинуть CPU, память и storage для развёртываний TARINIO.

## Небольшое развёртывание

Типичный сценарий:

- несколько защищаемых сервисов;
- умеренный трафик;
- single-node control-plane;
- bundled PostgreSQL.

Стартовая точка:

- `4 vCPU`
- `8 GB RAM`
- быстрый SSD-backed storage

## Среднее развёртывание

Типичный сценарий:

- несколько сайтов и upstreams;
- постоянный production traffic;
- отдельный observability stack;
- повышенные эксплуатационные требования.

Стартовая точка:

- `8 vCPU`
- `16 GB RAM`
- отдельный PostgreSQL host или хорошо выделенный volume

## HA / enterprise-style развёртывание

Типичный сценарий:

- две control-plane ноды;
- общий PostgreSQL и Redis;
- включённая observability;
- rolling upgrades и DR planning.

Стартовая точка:

- `2 x control-plane nodes`
- отдельный PostgreSQL
- отдельный Redis
- отдельная ёмкость под Prometheus / Grafana

## Что сильнее всего влияет на ресурсы

- число защищаемых сервисов;
- объём трафика;
- retention логов и запросов;
- сложность CRS и policy;
- observability retention;
- HA и benchmark-нагрузки.
