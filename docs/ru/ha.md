# Высокая доступность и multi-node

Эта страница относится к текущей ветке документации.

## Что именно появилось

В `2.0.2` TARINIO получил практическую multi-node схему для control-plane:

- общее состояние в PostgreSQL;
- распределённую координацию через Redis;
- несколько `control-plane` нод за API load balancer;
- leader election для фоновых задач;
- глобальную сериализацию `compile/apply`.

То есть прежнее ограничение `single-node only` для control-plane больше не является актуальным описанием продукта.

## Поддерживаемая топология

```text
ui -> api-lb -> control-plane-a / control-plane-b
                    |             |
                    +---- Redis ---+
                    +-- PostgreSQL-+
                    +-- shared runtime root
                              |
                           runtime
                              |
                        защищаемые сайты
```

В этой схеме:

- любой control-plane может обслуживать API;
- только одна нода одновременно выполняет leader-only фоновые задачи;
- `compile/apply` защищены распределёнными lock;
- все ноды видят одно и то же состояние в PostgreSQL.

## Что реализовано в коде

### PostgreSQL как реальный backend

Основное состояние control-plane хранится в PostgreSQL, а legacy file-state переносится автоматически при первом запуске.

### Redis coordination

Через Redis реализованы:

- distributed lock для `revision compile`;
- distributed lock для `revision apply`;
- leader election для:
  - `auto-apply`;
  - `dev fast start`;
  - `tls auto-renew`.

### Видимость активной ноды

`GET /api/app/meta` теперь возвращает:

- `ha_enabled`
- `ha_node_id`

Это позволяет быстро проверять, какая нода отвечает через балансировщик.

## Обязательные условия для HA

Все control-plane ноды должны разделять:

- одинаковый `POSTGRES_DSN`;
- одинаковый `CONTROL_PLANE_REDIS_ADDR`;
- общий runtime API token;
- общий runtime root или эквивалентную общую поверхность артефактов.

У каждой ноды должен быть свой:

- `CONTROL_PLANE_HA_NODE_ID`

Ключевые переменные:

- `CONTROL_PLANE_HA_ENABLED=true`
- `CONTROL_PLANE_HA_OPERATION_LOCK_TTL_SECONDS`
- `CONTROL_PLANE_HA_LEADER_LOCK_TTL_SECONDS`

## Поведение кластера

### Compile / apply

В любой момент только одна нода может держать глобальный operation lock. Это защищает от:

- гонки версий ревизий;
- параллельной записи в activation path;
- split-brain при apply.

### Auto-apply

Изменения могут прийти на любую ноду, но фактический auto-apply исполняет только текущий лидер.

### Dev fast start

Автобутстрап может быть включён на всех нодах, но реально выполняется только лидером.

### TLS auto-renew

Автообновление сертификатов также идёт через leader election, чтобы не было дублей renewal job.

## Готовая HA-лаборатория

Используй профиль:

- `deploy/compose/ha-lab`

В него входят:

- `api-lb`
- `control-plane-a`
- `control-plane-b`
- `postgres`
- `redis`
- `ui`
- `runtime`
- `ddos-model`
- `demo-app`
- `toolbox`

Профиль специально ограничен по ресурсам и подходит для рабочей станции.

## Сценарий проверки

1. Запуск:

```powershell
cd deploy/compose/ha-lab
docker compose --profile tools up -d --build
```

2. Открыть `http://localhost:18080/login`
3. Развернуть 20 demo-сервисов:

```powershell
docker compose --profile tools exec toolbox /tools/provision-20-services.sh
```

4. Проверить распределение по нодам:

```powershell
docker compose --profile tools exec toolbox /tools/cluster-status.sh
```

5. Запустить мини-нагрузку:

```powershell
docker compose --profile tools exec toolbox /tools/mini-ddos.sh
```

6. Остановить одну control-plane ноду и убедиться, что API продолжает работать через оставшуюся.

## Честные границы текущей реализации

Что уже закрыто:

- HA для control-plane;
- общее persistence state;
- реальная multi-node координация;
- локальная лаборатория для failover и anti-DDoS проверки.

Что ещё не входит в этот релиз:

- multi-region orchestration;
- fleet management;
- полноценный cross-host storage abstraction без внешнего shared storage;
- SIEM / SSO / SCIM и прочие enterprise-интеграции следующего уровня.
