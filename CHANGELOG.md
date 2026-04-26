## [3.0.5] - 26.04.2026

### Версия и релизные метаданные
- Запущен новый релизный цикл `3.0.5`.
- Синхронизирована версия в ключевых продуктовых точках: `control-plane`, UI и метаданных docs-site.
- Обновлены базовые релизные метки в RU/EN документации для текущего цикла.

### Решения Sentinel и перманентные баны
- Добавлено enterprise-подтверждение антифлаппинга при эскалации действий через `MODEL_PROMOTION_CONSECUTIVE_TICKS`, с учётом переходного состояния (`candidate_action`, `candidate_count`) и emergency-bypass.
- Добавлена полная интеграция перманентных банов `tarinio-sentinel -> control-plane` через `SentinelBanSyncService` и `ManualBanService.Ban("__all__", ip)` (без прямой записи в файлы blacklist).
- Добавлены параметры конфигурации и окружения для синхронизации банов: `CONTROL_PLANE_SENTINEL_BAN_SYNC_*`.
- Обновлены compose-профили (`default`, `auto-start`, `enterprise`, `ha-lab`, `testpage`): adaptive-volume монтируется в control-plane, синхронизация банов включена по умолчанию.
- Добавлены тесты для ban sync: promotion, idempotency, лимиты и переходы решений.

### Cardinality и L7 suggestions pipeline
- Добавлены guardrails против rule explosion в Sentinel: `max_active_ips`, `max_unique_paths_per_ip`, `max_published_entries`, `max_actions_per_minute`, eviction по inactivity TTL, очистка устаревших suggestions.
- Сохранена обратная совместимость `adaptive.json`, снижена нагрузка на `l4guard` за счёт top-N batching и `SaveAdaptiveIfChanged` (diff + publish interval).
- Усилен жизненный цикл L7 suggestions: `suggested -> shadow -> temporary -> permanent` с безопасными барьерами (контроль FP в shadow, rollback при высоком FP, окно временного применения, без безусловного auto-apply в permanent).
- Расширены статусы и хранилище anti-DDoS suggestions для `temporary` и `permanent`, добавлены поля `temporary_until` и `promotion_reason`.
- Обновлены тесты для action budget, promotion/rollback/pruning и переходов статусов suggestion store.

### Sentinel (Smart WAF)
- Продолжена эволюция `ddos-model -> tarinio-sentinel` с сохранением совместимости (`MODEL_*`, `ddos-model/config.json`, `adaptive.json`).
- Унифицированы service/container-названия Sentinel в compose, README и runbook.
- Унифицировано имя процесса в compose entrypoint: `tarinio-sentinel`.
- Добавлен опциональный CPU-only ML-классификатор (logistic regression) в `internal/sentinel/ml.go` без размещения в request path.
- Добавлены параметры `MODEL_ML_ENABLED`, `MODEL_ML_ARTIFACT_PATH`, `MODEL_ML_MIN_PROBABILITY`, `MODEL_ML_MAX_WEIGHT` с безопасным fallback на deterministic scoring.
- Добавлено версионирование model artifact в runtime state/adaptive output через `model_version`.

### Compose и эксплуатация
- Обновлены инструкции для `enterprise`, `ha-lab`, `testpage` на использование `tarinio-sentinel`.
- В профиле `testpage` сервисы Anti-DDoS переименованы в `tarinio-sentinel-mgmt` и `tarinio-sentinel-app`.
- Dockerfile-пайплайны Sentinel переведены на бинарь `/usr/local/bin/tarinio-sentinel`.

### Документация
- Актуализированы RU/EN страницы под цикл `3.0.5`, включая core, architecture, high-availability и operator guides.
- Обновлены enterprise smoke-страницы для Sentinel и service-protection.
- Исправлены несогласованности в обозначениях Sentinel в runtime/logging описаниях.
- Полностью обновлены RU model-docs по Sentinel/Anti-DDoS (`tarinio-sentinel`, `anti-ddos-model`, `anti-ddos-runbook`) с восстановлением корректного UTF-8 текста и синхронизацией с фактической логикой `3.0.5`.
- Синхронизированы EN model-docs: lifecycle L7 suggestions (`suggested -> shadow -> temporary -> permanent`), консервативное permanent-enforcement и актуальный resource envelope (`0.75 vCPU / 512MB` для default/enterprise/ha-lab).
- Устранены placeholder-версии вида `??????? ?????` в RU core/architecture-документации, заменены на фактическую версию `3.0.5` (и исторические метки `3.0.3`/`3.0.4` в release-секции RU README).

### Scoring и репутация
- Расширен signal-контур репутации: добавлен `signal_cross_site_spread`.
- Интегрирован ML-сигнал `signal_ml_risk` в explainability и decision pipeline.
- Усилено покрытие тестами по ML и расширенным сигналам в `internal/sentinel/ml_test.go`.

### Режимы источника, контейнерный профиль и enterprise smoke pack
- Добавлена явная поддержка режимов источника Sentinel для standalone и HA-ready сценариев с fallback backend в `internal/sentinel/sentinel.go` и тестами в `internal/sentinel/sentinel_test.go`.
- Обновлены compose-профили: явная проводка `MODEL_ENABLED` для Sentinel и CPU-only defaults (`0.75 vCPU`, `512 MB`) для `default`, `enterprise`, `ha-lab`, `testpage`.
- Расширен `scripts/smoke-sentinel.ps1`: режимы `classic-only`, `hybrid`, `adaptive-only`; fallback для baseline snapshot; retry-безопасный reset состояния runtime/sentinel при HA-флапах.
- Пересобран `scripts/smoke-sentinel-enterprise.ps1` для прогонов mode-pack и публикации RU/EN отчётов на основе артефактов.
- Зафиксированы результаты enterprise-проверки на 30 акторов для профилей `default` и `ha-lab` в:
  - `docs/eng/high-availability-docs/sentinel-enterprise-validation.md`
  - `docs/ru/high-availability-docs/sentinel-enterprise-validation.md`

