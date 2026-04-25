## [3.0.3] - 25.04.2026

### Приоритет блокировок и Anti-Bot
- Перестроен порядок проверок в `Easy NGINX`-конфиге:
  - проверки стран и черных списков выполняются раньше `Anti-Bot challenge`;
  - при совпадении с блокирующим правилом запрос получает `403` до редиректа на challenge.
- Добавлен флаг `scanner_auto_ban_enabled` (по умолчанию `true`):
  - при включении сканерные паттерны блокируются жестко (`403`) до challenge.

### Трафик: белые/черные списки
- Логика `allowlist` переведена в строгий режим:
  - если `allowlist` не пустой, `default_action` в `access policy` становится `deny`;
  - трафик вне `allowlist` блокируется (`403`).
- В UI синхронизирована запись `access policy`:
  - `access_allowlist` и `access_denylist` сохраняются как политика доступа;
  - `exceptions_ip` больше не подмешивается в `allowlist` (остается отдельным bypass-механизмом).

### UI и локализация
- Во вкладку `Antibot` добавлен чекбокс:
  - `Автобан сканеров (жесткая блокировка до challenge)`.
- Добавлены переводы ключа `sites.easy.antibot.scannerAutoBanEnabled` для локалей:
  - `en`, `ru`, `de`, `sr`, `zh`.
- Версия обновлена до `3.0.3` в UI и релизных метаданных.

### Тесты
- Дополнены тесты компилятора:
  - проверка, что hard-block проверки стоят раньше `Anti-Bot`;
  - проверка генерации scanner auto-ban guard.
- Дополнены тесты runtime apply:
  - проверка маппинга нового флага `scanner_auto_ban_enabled`;
  - проверка строгого поведения `allowlist`.
- Актуализированы UI-документные тесты:
  - контракт benchmark-документации переведен на режим удаленного контура.
- Добавлен интегральный smoke-пакет проверки защиты сервиса:
  - `scripts/smoke-service-protection.ps1`;
  - `scripts/smoke-service-protection-enterprise.ps1`;
  - сценарий из 10 клиентских ролей (нормальные, доверенные, scanner, payload, API burst, botnet, geo/deny-блок).

### Документация
- Удален контур benchmark-документации:
  - `docs/eng/security-benchmark-pack/README.md`
  - `docs/ru/security-benchmark-pack/README.md`
- Доработаны и синхронизированы документы:
  - расширен раздел CLI-команд;
  - обновлен и нормализован `docs/ru/core-docs/ui.md`;
  - дополнены описания security-профилей и связанные страницы навигации.
- Добавлен отдельный wiki-документ по enterprise-проверке полного контура защиты:
  - `docs/eng/high-availability-docs/service-protection-enterprise-validation.md`
  - `docs/ru/high-availability-docs/service-protection-enterprise-validation.md`
