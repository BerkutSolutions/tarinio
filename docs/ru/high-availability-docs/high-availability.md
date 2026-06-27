# Высокая доступность

Документ описывает многонодовый режим TARINIO `1.3.5`: схему ролей, сетевые требования, порядок запуска и базовые проверки после переключения.

## Область применения

TARINIO `1.3.5` поддерживает практическую многонодовую топологию control-plane на одном Docker-хосте:

- общее состояние PostgreSQL;
- общая Redis-координация;
- два и более узлов `control-plane` за API load balancer;
- leader election для фоновых задач;
- глобальная сериализация compile/apply через distributed lock.

## Поддерживаемая схема HA

```text
ui -> api-lb -> control-plane-a / control-plane-b
                    |             |
                    +---- Redis ---+
                    +-- PostgreSQL-+
                    +-- shared runtime root
                              |
                           runtime
                              |
                         protected sites
```

В этой модели:

- любой узел control-plane обслуживает API-трафик;
- только один узел выполняет leader-only фоновые задачи;
- compile/apply защищены distributed lock;
- все узлы видят одно и то же состояние в PostgreSQL.

## Ключевые env-переменные

- `CONTROL_PLANE_HA_ENABLED=true`
- `CONTROL_PLANE_HA_NODE_ID` — уникален для каждого узла
- `CONTROL_PLANE_HA_OPERATION_LOCK_TTL_SECONDS`
- `CONTROL_PLANE_HA_LEADER_LOCK_TTL_SECONDS`
- `POSTGRES_DSN` — одинаков для всех узлов
- `CONTROL_PLANE_REDIS_ADDR` — одинаков для всех узлов

## HA-лаборатория

Используйте: `deploy/compose/ha-lab`

Профиль включает: `api-lb`, `control-plane-a`, `control-plane-b`, `postgres`, `redis`, `ui`, `runtime`, `tarinio-sentinel`, `demo-app`, `toolbox`, опционально `prometheus`/`grafana`.

Подробная инструкция по запуску и проверке — в английской версии:

- [Открыть английскую версию](/en/high-availability-docs/high-availability/)

## Связанные документы

- [Развёртывание](../core-docs/deploy.md)
- [Обновление](../core-docs/upgrade.md)
- [Восстановление](../core-docs/disaster-recovery.md)
- [Enterprise-проверка Sentinel](./sentinel-enterprise-validation.md)
- [Enterprise-проверка Anti-Bot](./antibot-enterprise-validation.md)
- [Enterprise-проверка защиты сервисов](./service-protection-enterprise-validation.md)
