## [1.4.3] - 28.06.2026

### Ядро

- Добавлено поле `show_geo_block_page` в профиль сайта (страновая политика) — при включении WAF возвращает HTTP 451 вместо 403 для заблокированных по стране запросов и отдаёт информационную страницу.
- Создан глобальный артефакт `errors/_global/geo_block.html` — HTML-страница с тёмной темой, трёхязычной локализацией (ru/en/de) и инфо-блоком (время запроса, IP клиента, страна по GeoIP, Request ID).
- `error_page 451` в `site.conf.tmpl` передаёт параметры `rid`, `ip`, `ts`, `cc` через query string — страница читает их через `URLSearchParams`.
- Добавлен preview-location `/preview/geo-block` в management server block (`base.conf.tmpl` и `site.conf.tmpl`).
- Обновлены тесты компилятора: `expectedArtifacts = 4 + (2 * len(supportedErrorStatusCodes))`, добавлен `show_geo_block_page` в contract-тест control-plane.

### UI

- На вкладке «Гео» добавлен чекбокс «Показывать страницу ограничения по региону» — первым элементом, выше списков стран.
- Добавлено пояснение к чекбоксу в хелп-модалке гео-вкладки.
- Исправлен CSS: добавлено `.waf-checkbox.full { grid-column: 1 / -1 }` — чекбокс корректно занимает всю строку в двухколоночном гриде.

### Локализация

- Добавлен ключ `sites.easy.geo.showGeoBlockPage` на всех 5 языках (ru, en, de, sr, zh).
- Добавлены ключи `sites.help.geo.showGeoBlockPage.label` и `sites.help.geo.showGeoBlockPage.usage` на всех 5 языках.
