# Журнал изменений

Все значимые изменения проекта фиксируются в этом файле.

## [Не выпущено]

## [1.0.1] - 2026-04-04

### Изменено
- Compose-профили разделены на два сценария:
  - `deploy/compose/default` — production baseline.
  - `deploy/compose/auto-start` — localhost auto bootstrap/dev fast start.
- Версия приложения обновлена до `1.0.1` в `control-plane/internal/appmeta/meta.go`.

### Документация
- Удален быстрый старт из корневых README и основного индекса документации.
- Обновлена документация по compose-профилям:
  - `deploy/compose/README.md`
  - `deploy/compose/default/README.md`
  - `deploy/compose/auto-start/README.md`
- Базовые версии в индексах документации и корневых README обновлены до `1.0.1`.

## [1.0.0] - 2026-04-03

### Первый публичный релиз

Первая релизная версия продукта `Berkut Solutions - TARINIO`.

Базовая версия: `1.0.0`.
