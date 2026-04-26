# Runbook Anti-DDoS

Этот runbook фиксирует практический порядок действий при всплесках трафика и атаках в TARINIO `3.0.6`.

## 1. Быстрый triage

1. Проверить `Dashboard`, `Events`, `Security Events`.
2. Определить профиль нагрузки: normal spike, scanner, brute-force, flood.
3. Убедиться, что применена актуальная ревизия и корректный профиль сервиса.
4. Проверить, активен ли `tarinio-sentinel` для нужных сайтов (`model_enabled` + per-site opt-in).

## 2. Базовые проверки runtime

- статус контейнера `tarinio-sentinel`;
- актуальность `adaptive.json` и `model-state.json`;
- наличие записей в `l7-suggestions.json` при scanner-активности;
- корректность `l4guard` (цепочки и счетчики не деградируют).

## 3. Режимы эксплуатации

Поддерживаются три режима в smoke/enterprise-пакете:

- `classic-only`: работает только baseline Anti-DDoS, adaptive-модель отключена;
- `hybrid`: baseline и Sentinel работают совместно;
- `adaptive-only`: эскалация опирается на Sentinel при активном baseline-профиле L4.

## 4. Как читать решения Sentinel

Лестница действий:

- `watch` -> `throttle` -> `drop` -> `temp_ban`

Важно:

- агрессивные действия требуют multi-signal подтверждения или emergency-сигнала;
- anti-flapping защищает от дерганых переходов;
- публикация ограничена budget-параметрами и top-N batching.

## 5. Работа с L7 suggestions

Статусы:

- `suggested`: кандидат без блокировки;
- `shadow`: считаются would-block/FP метрики;
- `temporary`: временное применение с TTL;
- `permanent`: кандидат на постоянное правило после review.

Безусловного auto-apply permanent нет: финальная фиксация должна проходить операторский контроль.

## 6. Перманентные баны

Sentinel не должен писать напрямую в blacklist-файлы.

Промоут в постоянный бан выполняется через control-plane контур синхронизации (`SentinelBanSyncService`), что сохраняет аудит, idempotency и единые policy-priority правила.

## 7. Рекомендуемые лимиты (CPU-only)

- профиль `default/enterprise/ha-lab`: `0.75 vCPU`, `512 MB` для Sentinel;
- practical range: `0.25-0.75 vCPU`, `128-512 MB`;
- обязательно держать guardrails:
  - `MODEL_MAX_ACTIVE_IPS`
  - `MODEL_MAX_UNIQUE_PATHS_PER_IP`
  - `MODEL_MAX_PUBLISHED_ENTRIES`
  - `MODEL_MAX_ACTIONS_PER_MINUTE`

## 8. Когда эскалировать инцидент

Эскалируйте вручную, если:

- `adaptive.json` не обновляется при явной атаке;
- резко выросли false positives на нормальном трафике;
- `l4guard` показывает rule churn выше операционного порога;
- Sentinel стабильно упирается в cardinality/budget лимиты.

## 9. Что фиксировать в постмортеме

- профиль (`default`/`ha-lab`) и режим (`classic-only`/`hybrid`/`adaptive-only`);
- входные метрики атаки (RPS, unique IP, типы path);
- решения Sentinel и длительность эскалации;
- FP/FN и влияние на легитимный трафик;
- какие параметры были изменены и почему.

## Связанные материалы

- [Модель Anti-DDoS](anti-ddos-model.md)
- [TARINIO Sentinel](tarinio-sentinel.md)
- [Тюнинг WAF](../operators/waf-tuning-guide.md)
