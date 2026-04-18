# Журнал изменений

Все значимые изменения проекта фиксируются в этом файле.

## [1.1.20] - 2026-04-18

### Healthcheck / Probes
- Исправлены healthcheck-пробы `API дашборда`, `API запросов` и `API событий`: они переведены на lightweight probe-режим через `/api/dashboard/stats?probe=...`, без запроса тяжёлых payload и без пересечения со special-case роутом UI nginx для `/api/requests`.
- Медленные ответы (`needs_attention`) теперь показываются warning-статусом, а не серым neutral-state.

### Runtime / Nginx
- Дополнительно увеличены `variables_hash_max_size` и `variables_hash_bucket_size`, чтобы убрать warning `could not build optimal variables_hash` на инстансах с большим числом переменных.

### Live Logs
- Для live logs контейнеров добавлена дедупликация строк по `timestamp + message`, чтобы при polling не появлялись повторяющиеся записи одной и той же строки.

### Установка / AIO
- Усилена очистка терминала в `install-aio.sh`: перед выводом теперь выполняется полный terminal reset (`ESC c`) с дополнительной очисткой экрана и scrollback.