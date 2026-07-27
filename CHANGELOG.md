## [1.5.15] - 27.07.2026

- UI image clean builds now include every repository contract fixture and Python required by `go test ./ui/tests`, preventing AIO upgrades from failing on hosts without a matching Docker cache.

### Интерфейс и управление сервисами

- Удалены отдельные виджеты загрузки CPU и использования памяти с Dashboard; состояние контейнеров и их фактические метрики остаются в виджете здоровья контейнеров.
- Из Basic Auth удалены неработающие настройки расположения и произвольного текста: защита всегда охватывает сервис целиком, а realm задаётся безопасным системным значением.
- Для Anti-Bot добавлен срок действия проверочной сессии от 5 минут до 24 часов и режим без ограничения. Выбранное значение сохраняется в профиле и точно переводится в Max-Age runtime-cookie.
- Страновая политика получила отдельное разрешение локальных, loopback, link-local и private IP-адресов. Неизвестная страна по-прежнему блокируется whitelist-политикой по умолчанию.
- Исправлено сохранение белых списков: Easy-редактор сразу использует сохранённую access policy, а переход через Raw-режим больше не стирает allowlist/denylist.
- Редактор сервиса сохраняет и повторно читает динамические списки, Anti-Bot, Basic Auth, Geo, TLS, mTLS, ModSecurity, WebSocket, лимиты и остальные поддерживаемые поля.
- После compile/apply боковая панель сразу показывает новую активную ревизию без перезагрузки страницы.
- Модалка ревизий разделена на список и подробности. Выбор строки открывает сведения, статусы и точный список изменённых полей сервиса; кнопка раскрытия «+» удалена.
- Поток событий Anti-DDoS ограничен по высоте, получил размер страницы, нумерацию страниц и флаг страны рядом с IP.
- Для защиты уровней L4 и L7 добавлены отдельные подробные справки о назначении, параметрах, рисках и runtime-проверке.
- Все новые строки и сценарии локализованы на русский, английский, немецкий, сербский и китайский языки.

### Runtime, безопасность и Compose

- Sentinel во всех профилях запускается без root с UID/GID 65532:4, без capabilities, с read-only rootfs и выделенными writable tmpfs/volumes.
- State и adaptive volumes переведены на новые rootless-совместимые версии; образ заранее создаёт каталоги с корректными владельцем и правами.
- Согласованы default, auto-start, enterprise, HA-lab, E2E и testpage Compose-профили. E2E использует только собственные изолированные стеки и не изменяет production default Compose.
- Country whitelist корректно учитывает явные исключения и локальные сети, не ослабляя deny-by-default для неизвестных внешних адресов.
- Ревизии сохраняют структурированный diff immutable snapshots без публикации секретных значений.
- Кандидат активной ревизии остаётся неизменяемым: повторное применение идентичного bundle является no-op, а подмена содержимого отклоняется.
- Исправлены reload/readiness-проверки, runtime request backend, фильтрация Events/Audit и безопасное восстановление после кратких перезапусков.

### E2E и CI

- Все E2E/DAST jobs на shell runner теперь привязаны к стабильному Compose project и отдельному блоку портов своего CI_CONCURRENT_ID; прерванный job очищается следующим job того же слота и не может оставить конфликтующий порт для нового pipeline.
- E2E Compose projects use the numeric GitLab runner ID and concurrent slot, keeping Go, browser, stack-health and DAST names valid and collision-free for Compose, networks, volumes and Docker images.
- Runtime syntax validation maps `/etc/waf` references into a disposable shadow bundle and never renames the live tree, so nested volume mounts cannot break bootstrap or revision apply.
- Browser E2E uses four balanced resource lanes matching the four runner slots; each job still receives its own slot-bound Compose project and ports.
- Geo draft rerenders preserve local-IP policy state, Dashboard chart tooltips survive live refreshes, and Anti-DDoS table/detail views show the same country indicator and client IP.
- Dashboard chart overlays explicitly accept pointer events across the transparent plot area, so desktop and mobile tooltips open reliably in Chromium.
- Dashboard chart pointer handlers are anchored to the stable widget body and rebound without leaks, so SVG replacement during resize or background refresh cannot drop hover interaction.
- Dashboard series imports its shared `clamp` helper explicitly, preventing pointer calculations from aborting before the tooltip becomes visible.
- The destructive revision-timeline API workflow restores a real compiled and applied runtime revision before later contracts run, preventing shared-stack test state from leaking between scenarios.
- Rootless UI images force both nginx configuration files to mode `0644`, so restrictive source permissions from upgrade artifacts cannot make nginx unreadable and cascade into runtime DNS failure.
- Default and enterprise AIO upgrades repair legacy installation permissions before image builds while preserving `.env`, private-key and certificate confidentiality; an integration fixture verifies the repair contract.
- Allowlist E2E cleanup removes every compatibility policy for its disposable site before deleting parent resources.

- Browser E2E выполняются параллельно на полностью раздельных Compose projects, сетях, volumes и диапазонах портов; стеки не разделяют runtime, cookies, Docker-метрики или данные.
- Добавлены реальные проверки Anti-Bot TTL в сгенерированном nginx, локальных IP в country policy, белых списков в Easy/Raw, diff ревизий и пагинации Anti-DDoS.
- Длинные browser-наборы разделены на независимые shards с единым строгим gate: запрещены skip, retry, fixme и скрытый успешный fallback.
- Улучшены ожидания hydration и кратких runtime reload для Dashboard, Requests, Events, Settings, Services, TLS, CRS, Bans и Revisions.
- Одноразовые E2E/DAST-стенды удаляют только свои контейнеры, сети, volumes и локальные образы.
- Release evidence содержит таблицу матрицы, E2E, DAST, стабильности и Trivy, а также исходные машиночитаемые доказательства.
- publish:github-mirror и publish:github-release снова отображаются в main/tag pipeline, но запускаются только вручную и не мешают зелёному статусу обычного pipeline.
- GitHub и GHCR не получают автоматических записей: токены и ключи требуются только после ручного запуска publish-job.

## [1.5.14] - 22.07.2026

### Основные изменения

- Добавлен изолированный Playwright-пакет с desktop/mobile-проектами, безопасным localhost base URL и строгой проверкой результатов.
- Реализованы browser-сценарии Dashboard, Services, Requests, Bans, Revisions, Events, Activity, Anti-DDoS, OWASP CRS, TLS, Administration и Settings.
- E2E подтверждает полный путь настройки: API или UI, compile/apply ревизии, активный runtime artifact и фактический HTTP-ответ WAF.
- Добавлены реальные проверки ModSecurity, Anti-Bot, Geo, Basic Auth, L4/L7 Anti-DDoS, mTLS, virtual patches, custom error pages и rate limit.
- Реализованы server-side RBAC, TOTP step-up, audit/readback и очистка тестовых сущностей для критичных мутаций.
- Исправлены сохранение порядка Basic Auth, единицы r/s и r/m, HTTPS upstream, mTLS-материалы и уникальность host/upstream references.
- Расширены import/export, Easy/Raw parity, unsaved-change guard, адаптивная мобильная раскладка и справочные модалки редактора.
- Dashboard получил 24-часовые согласованные серии, реальные top IP/country/URL/error данные и устойчивую фоновую агрегацию.
- Requests и Events перестали скрывать отказ backend пустым успешным ответом; ошибки видимы пользователю и проверяются E2E.
- Добавлены безопасные операции управления пользователями, ролями, сертификатами и настройками с server-side проверкой прав.
- Runtime и служебные образы переведены на непривилегированных пользователей; Compose и healthchecks получили изолированные ресурсы.
- CI получил registry покрытия API/runtime/browser, DAST, Trivy, evidence artifacts и блокировку при пропусках, повторах или критичных находках.
- Все основные экраны и новые пользовательские строки локализованы на ru, en, de, sr и zh.
