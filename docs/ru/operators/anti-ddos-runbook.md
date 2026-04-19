# Руководство Оператора: Anti-DDoS

Этот документ описывает безопасную эксплуатацию раздела `Anti-DDoS` в TARINIO.

Раздел управляет тремя уровнями защиты:
- базовой L4-защитой на уровне `iptables`;
- глобальным L7 rate-limit override, который компилируется в ревизию;
- адаптивной anti-DDoS моделью, которая анализирует runtime access log и публикует временные throttle/drop-правила.

Источник истины для настроек:
- control-plane storage: `anti_ddos_settings.json`;
- snapshot ревизии: поле `anti_ddos_settings`;
- runtime: артефакты текущей активной ревизии после `compile -> validate -> apply`.

## Что меняет оператор

На странице `/anti-ddos` оператор настраивает:
- `use_l4_guard`, `chain_mode`, `conn_limit`, `rate_per_second`, `rate_burst`, `ports`, `target`;
- `enforce_l7_rate_limit`, `l7_requests_per_second`, `l7_burst`, `l7_status_code`;
- adaptive-модель: интервалы опроса, пороги throttle/drop, время удержания, веса сигналов, emergency thresholds.

Любое изменение этих параметров считается конфигурационным изменением и должно попадать в новую ревизию.

## Безопасный рабочий порядок

1. Изменить параметры на странице `Anti-DDoS`.
2. Собрать новую ревизию в разделе `Ревизии`.
3. Применить ревизию.
4. Проверить:
   - `GET /healthz` и страницу `/healthcheck`;
   - статус ревизии и последние rollout events;
   - отсутствие ложных блокировок на боевом трафике;
   - события `security_rate_limit`, `security_access`, `security_waf`.

## Рекомендуемые стартовые значения

Для первой боевой активации используйте консервативный профиль:
- `conn_limit`: `200`;
- `rate_per_second`: `100`;
- `rate_burst`: `200`;
- `ports`: `80,443`;
- `target`: `DROP`;
- `enforce_l7_rate_limit`: `true`;
- `l7_requests_per_second`: `100`;
- `l7_burst`: `200`;
- `l7_status_code`: `429`.

Для адаптивной модели типовой baseline:
- `model_poll_interval_seconds`: `2`;
- `model_decay_lambda`: `0.08`;
- `model_throttle_threshold`: `2.5`;
- `model_drop_threshold`: `6.0`;
- `model_hold_seconds`: `60`;
- `model_throttle_rate_per_second`: `3`;
- `model_throttle_burst`: `6`;
- `model_throttle_target`: `REJECT`.

## Практика безопасного тюнинга

- Ужесточайте защиту по одному блоку за раз.
- Не уменьшайте одновременно и `conn_limit`, и `rate_per_second`, и `l7_requests_per_second`.
- Сначала соберите baseline нормального трафика, затем включайте более строгие emergency thresholds.
- Для management site (`control-plane-access`) проверяйте, что не создаёте условий для само-блокировки операторов.

## Симптомы и действия

Если легитимный трафик получает `429`:
- проверить L7 override;
- увеличить `l7_burst`;
- проверить trusted proxies и реальный client IP;
- убедиться, что это не локальный per-site rate-limit, а именно anti-DDoS слой.

Если падают новые соединения до NGINX:
- проверить L4 guard chain placement;
- проверить `DOCKER-USER` / `INPUT`;
- проверить `destination_ip` для docker bridge / DNAT сценария;
- сверить `ports` с реальным runtime exposure.

Если adaptive-модель начала агрессивно throttle/drop:
- посмотреть свежие записи access log и security events;
- проверить emergency burst detector;
- увеличить `model_drop_threshold` и/или уменьшить веса сигналов `403/444`;
- переприменить известную стабильную ревизию при сильной деградации.

## Откат

Если настройка оказалась слишком агрессивной:
1. Открыть раздел `Ревизии`.
2. Найти предыдущую known-good revision.
3. Выполнить `apply`.
4. Дождаться успешного применения и восстановления `/healthz`.
5. Вернуться в `Anti-DDoS`, смягчить параметры и повторить compile/apply.

Откат всегда выполняется через ревизии. Ручное редактирование runtime filesystem или `iptables` в обход ревизии не считается штатным сценарием.

## Связанные документы

- `docs/ru/operators/anti-ddos-model.md`
- `docs/ru/operators/runtime-l4-guard.md`
- `docs/ru/runbook.md`
