## [1.4.8] - 30.06.2026

### Безопасность

- Восстановлена полная работа WAF в runtime e2e: чёрные списки IP, User-Agent и URI снова блокируют трафик, ModSecurity/CRS реально останавливает SQLi и XSS, виртуальные патчи применяются как SecRule, scanner auto-ban блокирует сигнатуры сканеров независимо от основной antibot-логики, а режимы transparent и monitor не оставляют активные блокирующие правила.
- Для `/login` панели управления выделен отдельный edge rate-limit с HTTP 429, чтобы endpoint оставался под первой линией защиты и не включал self-ban или escalation для операторов.

### Ядро

- Компилятор easy runtime теперь явно включает `modsecurity on` только рядом с easy ModSecurity rules file и не подмешивает legacy `modsecurity/sites/<site>.conf` в custom route locations. Это устраняет рассинхронизацию между easy-профилем и legacy-политиками и возвращает предсказуемое поведение CRS, виртуальных патчей и режимов безопасности.
- Scanner auto-ban отделён от основного antibot challenge: защита от сканеров теперь генерирует самостоятельный guard и продолжает работать даже при отключённом интерактивном challenge.
- Уточнена генерация management/admin bypass paths: `/login` и `/login/2fa` больше не попадают в management easy bypass pattern, а `localhost` считается management-site только при явном `CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID`. Это устраняет регрессии с duplicate `location "/login"` и ложным management-режимом для обычного `localhost` site.
- Генерация HTTPS-сайтов сделана детерминированной для одновременного существования domain-host и IP-host: named TLS-sites теперь рендерятся раньше IP-based TLS-sites, поэтому default HTTPS TLS ref и порядок `sites/*.conf` больше не подсовывают IP-certificate доменному SNI.

### Сервисы

- API easy-site профиля получил обратную совместимость с alias-полями верхнего уровня, которые используются e2e и legacy-клиентами: security mode, blacklist/exceptions, rate-limit, geo policy, custom limit rules, auth session TTL и virtual patches корректно раскладываются во вложенные структуры профиля.
- `/api/requests` теперь явно помечает каждую строку типом `request` или `security`, чтобы UI мог отделять обычные запросы от ModSecurity и других событий безопасности одним общим фильтром.
- Для `/api/requests` добавлен двусторонний compatibility-layer: handler принимает старые записи без `row_type`, восстанавливает тип из `stream`/`type`/`source_component`, сохраняет приоритет явных `row_type`/`rowType` и помечает временную legacy-поддержку полем `legacy_row_type_support=true`.

### UI

- Во вкладке запросов добавлен первый столбец `Тип`, фильтр по security/request событиям и нормализованные метки причин безопасности, чтобы ModSecurity, SQLi/XSS, rate-limit, geo, auth и challenge-события одинаково читались и в новых, и в legacy-полях.
- Модалка details во вкладке Requests теперь показывает security rows в нормализованном request-like виде, выводит компактное summary в header и явно помечает legacy-compatibility для старых записей без `row_type`.
- Frontend Requests переведён на единый контракт `event_type` и `security_reason` из `/api/requests` с fallback на legacy `details`, поэтому UI и backend выровнены по одной модели security telemetry.
- Behavioral UI/runtime e2e обновлён под реальный контракт: dashboard stats проверяются по актуальным ключам API, captcha challenge flow больше не ломается на абсолютном redirect Location, а unknown-host проверяется по брендированной `421`-странице без утечки raw nginx страницы.
- UI full regression e2e стабилизирован для localhost-routing и окружений с placeholder secret: ключевые страницы проходят через 429-aware helper, а login flow не ломается из-за masked password `***`.

### Тесты

- Полный `TestE2EBehavioral` снова проходит на runtime-stack без skip: подтверждены контракт dashboard, captcha flow, брендированные geo block branches и брендированный unknown-host fallback.
- Compiler, service и UI-contract tests синхронизированы с текущими runtime-инвариантами: safe nginx guards без вложенных `if`, обязательный 429 для route rate-limit, compatibility easy-site profile и актуальные поля security telemetry.
- Добавлены и обновлены regression tests на management-site path ownership, env-driven localhost semantics, coexistence HTTPS domain-site + IP-site, legacy/new rows в `/api/requests`, security summary/details modal и runtime-normalized `event_type`.
- Live/runtime regression tests дополнительно закрепляют, что `transparent` и `monitor` не уводят `/admin` в antibot/auth redirect path, а behavioral/e2e runner выводит компактный expected/actual summary по каждому subtest с полным логом в `.work/logs`.

### Документация

- Актуализирована доказательная база WAF UI и runtime-поведения: tab01–tab11 описывают статические гарантии компилятора, а `TestE2EBehavioral` закреплён как эталон end-to-end работы WAF в Docker Compose stack.
- Changelog очищен от старых пошаговых формулировок, англоязычных хвостов и номеров задач; записи сгруппированы по подсистемам в продуктовом формате.

### Инфраструктура

- Версия синхронизирована на 1.4.7 через штатный механизм: метаданные приложения, npm package metadata и i18n `app.version` во всех поддерживаемых локалях обновлены единообразно.
- Install/upgrade scripts больше не ротируют существующие placeholder-like секреты автоматически при апгрейде уже развернутого стека: существующий `.env` сохраняется как источник истины, а инсталлеры выводят явную подсказку использовать `scripts/rotate-env-secrets.sh` для осознанной аварийной ротации.
- Добавлен `scripts/rotate-env-secrets.sh`: аварийный и операционный сценарий ротации `POSTGRES_PASSWORD`, `POSTGRES_DSN`, `OPENSEARCH_PASSWORD`, `CONTROL_PLANE_SECURITY_PEPPER` и `WAF_RUNTIME_API_TOKEN` с синхронизацией Postgres-пароля и обязательной health-проверкой control-plane/runtime после перезапуска.
- App-compat contract tests теперь страхуют installer-путь: оба инсталлера обязаны маркировать upgrade-flow, ссылаться на script ротации секретов, а `rotate-env-secrets.sh` обязан реально ротировать Postgres и проверять работоспособность стека после ротации.
