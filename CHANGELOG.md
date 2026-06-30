## [1.4.7] - 30.06.2026

### Безопасность

- Восстановлена эталонная работа WAF в полном runtime-stack e2e: чёрные списки IP, User-Agent и URI снова блокируют трафик, ModSecurity/CRS реально останавливает SQLi и XSS, виртуальные патчи применяются как SecRule, scanner auto-ban блокирует сигнатуры сканеров независимо от основной antibot-challenge логики, а режимы transparent и monitor больше не оставляют активные блокирующие правила.
- Исправлены guard-цепочки nginx для Basic Auth и двухслойной antibot-эскалации: вложенные `if` заменены на безопасные составные переменные, поэтому auth-gate стабильно отдаёт 302 на `/auth`, verify endpoint выставляет cookie, а challenge escalation корректно ведёт клиента через stage1.
- Восстановлена корректная работа кастомных страниц ошибок: брендированная 403-страница включается при `use_custom_error_pages=true`, отключение конкретного кода через `disabled_error_pages` возвращает стандартную короткую страницу, а geo-block остаётся на 451.
- Login endpoints панели управления получили отдельный edge rate-limit с HTTP 429, чтобы внешний WAF на отдельной ВМ не оставлял `/login` без первой линии защиты, но при этом не включал self-ban/escalation для операторов панели.

### Ядро

- Easy runtime compiler теперь явно включает `modsecurity on` только рядом с easy ModSecurity rules file и не подмешивает legacy `modsecurity/sites/<site>.conf` в custom route locations. Это устраняет рассинхронизацию между easy-профилем и legacy-политиками и возвращает предсказуемое поведение CRS, виртуальных патчей и security modes.
- Custom route rate-limit стабилизирован как HTTP 429 независимо от набора `bad_behavior_status_codes`, а anti-DDoS L7 defaults больше не перетирают уже заданную site-level rate-limit policy.
- Scanner auto-ban отделён от основного antibot challenge: включённая защита от сканеров теперь генерирует самостоятельный guard и продолжает работать даже если интерактивный challenge отключён.

### Сервисы

- API easy-site профиля получил обратную совместимость с top-level alias-полями, которые используются e2e и legacy-клиентами: security mode, blacklist/exceptions, rate-limit, geo policy, custom limit rules, auth session TTL и virtual patches корректно раскладываются во вложенные структуры профиля.
- `virtual_patches` добавлены в контракт easy-site профиля и проходят весь путь от API до runtime compiler, поэтому per-site виртуальные патчи сохраняются, компилируются и применяются в ModSecurity artifact.
- Runtime apply теперь объединяет виртуальные патчи из easy-profile и отдельного хранилища virtual patches, сохраняя совместимость обоих источников правил.
- Incoming mTLS теперь применяется в server-контексте HTTPS-сайта, а CA bundle для проверки клиентских сертификатов попадает в runtime-артефакты вместе с TLS-материалами.

### Тесты

- Полный `TestE2EBehavioral` прошёл на runtime-stack: подтверждены blacklist, rate-limit, antibot, scanner auto-ban, ModSecurity SQLi/XSS, Basic Auth gate, CORS/CSP/Referrer/Permissions headers, custom error pages, virtual patches, challenge rules, two-layer escalation, cookie flags, strict parsing, WebSocket config, geo time windows, incoming mTLS и сохранение JA3-полей.
- `MTLS_IncomingClientCert_Required` больше не скипается: тест генерирует self-signed CA, server cert и client cert, проверяет отказ HTTPS без клиентского сертификата, успешный проход с клиентским сертификатом и обратный проход после выключения mTLS.
- Обновлены compiler/service contract tests под новые runtime-инварианты: safe nginx guards без вложенных `if`, обязательный 429 для route rate-limit, сохранение explicit site policy при anti-DDoS L7 defaults и расширенный контракт easy-site profile.
- Документы покрытия тестами дополнены вкладкой 11 и behavioral e2e как доказательной базой реальной работы WAF, а не только статической генерации конфигов.
- E2E-раннер для release-проверки теперь показывает компактные этапы запуска стека, healthcheck, compile/apply и по каждому функциональному subtest выводит имя, ожидаемый результат и фактическое совпадение, а подробный Docker/Go вывод сохраняется в `.work/logs` и раскрывается только при ошибке.

### Документация

- Актуализирована доказательная база WAF UI и runtime-поведения: tab01–tab11 описывают статические compiler guarantees, а `TestE2EBehavioral` закреплён как эталон end-to-end работы WAF в Docker Compose stack.
- Changelog очищен от устаревших записей предыдущего шага и приведён к релизу 1.4.7 с группировкой по подсистемам.

### Инфраструктура

- Версия синхронизирована на 1.4.7 через штатный механизм: metadata приложения, npm package metadata и i18n `app.version` во всех поддерживаемых локалях обновлены единообразно.
- Sentinel image больше не выполняет `apk add` во время финальной сборки: сертификаты CA копируются из builder-слоя, поэтому локальный стек и `install-aio.sh` не падают из-за временных TLS/Alpine mirror ошибок при сборке.
