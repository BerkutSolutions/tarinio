# Справочник env-параметров WAF

Эта страница относится к текущей ветке документации.

Документ описывает все поля сервиса WAF в env-формате, который используется при raw-редактировании и экспорте `.env` из раздела `Сервисы`.

## Как читать этот документ

- Все ключи начинаются с префикса `WAF_SITE_`.
- Массивы и сложные значения задаются в JSON-формате: `[]`, `["GET","POST"]`, `[400,401,403]`, `[{"path":"/login","rate":"20r/s"}]`.
- Булевы значения задаются как `true` или `false`.
- Если поле не задано в raw-конфиге, интерфейс применяет значение по умолчанию из текущей версии продукта.
- Примеры ниже используют безопасные тестовые значения и не содержат продовых секретов.

## Идентификация и публикация сервиса

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ID` | Внутренний идентификатор сервиса. Используется в API, ревизиях и связанных сущностях. | Строка в нижнем регистре, обычно на основе хоста. | Нет, задаётся при создании. | `app-example-com` |
| `WAF_SITE_PRIMARY_HOST` | Основное публичное доменное имя сервиса. | FQDN или `localhost` для локальной разработки. | Нет, задаётся при создании. | `app.example.com` |
| `WAF_SITE_ENABLED` | Включает или выключает сервис без его удаления. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SECURITY_MODE` | Базовый режим реакции WAF на угрозы. | `transparent`, `monitor`, `block` | `block` | `block` |

## TLS и сертификаты

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_TLS_ENABLED` | Включает HTTPS для сервиса. | `true`, `false` | `true` | `true` |
| `WAF_SITE_TLS_SELF_SIGNED` | Пометка, что используется self-signed fallback для локальной среды. | `true`, `false` | `false` | `false` |
| `WAF_SITE_CERTIFICATE_ID` | Идентификатор сертификата, который привязывается к сервису. | Строка | Производное от `site_id`: `<site_id>-tls` | `app-example-com-tls` |
| `WAF_SITE_AUTO_LETS_ENCRYPT` | Автоматически выпускать сертификат через ACME. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_LETS_ENCRYPT_STAGING` | Использовать staging ACME endpoint вместо production. | `true`, `false` | `false` | `false` |
| `WAF_SITE_USE_LETS_ENCRYPT_WILDCARD` | Выпускать wildcard-сертификат, если сценарий это поддерживает. | `true`, `false` | `false` | `false` |
| `WAF_SITE_CERTIFICATE_AUTHORITY_SERVER` | Какой CA использовать для управляемого сертификата. | `letsencrypt`, `zerossl`, `custom`, `import` | `letsencrypt` | `letsencrypt` |
| `WAF_SITE_ACME_ACCOUNT_EMAIL` | Email ACME-аккаунта для выпуска и обновления сертификатов. | Валидный email | Пусто | `ops@example.com` |

## Upstream и reverse proxy

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_UPSTREAM_ID` | Идентификатор upstream-ресурса. | Строка | Производное от `site_id`: `<site_id>-upstream` | `app-example-com-upstream` |
| `WAF_SITE_UPSTREAM_HOST` | Хост или IP upstream-приложения. | FQDN, контейнерное имя, IPv4/IPv6 | `ui` | `10.0.10.15` |
| `WAF_SITE_UPSTREAM_PORT` | Порт upstream-приложения. | `1..65535` | `80` | `8080` |
| `WAF_SITE_UPSTREAM_SCHEME` | Протокол upstream. | `http`, `https` | `http` | `http` |
| `WAF_SITE_USE_REVERSE_PROXY` | Включает reverse proxy до upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_REVERSE_PROXY_HOST` | Полный адрес upstream для easy-profile. | URL | Производится из `scheme://host:port` | `http://10.0.10.15:8080` |
| `WAF_SITE_REVERSE_PROXY_URL` | Базовый путь на upstream. | Путь, начинающийся с `/` | `/` | `/` |
| `WAF_SITE_REVERSE_PROXY_CUSTOM_HOST` | Пользовательский `Host` при проксировании, если он нужен приложению. | FQDN или пусто | Пусто | `backend.internal.example` |
| `WAF_SITE_REVERSE_PROXY_SSL_SNI` | Включает SNI при HTTPS-проксировании к upstream. | `true`, `false` | `false` | `true` |
| `WAF_SITE_REVERSE_PROXY_SSL_SNI_NAME` | Имя SNI для TLS upstream-соединения. | FQDN | Пусто | `backend.internal.example` |
| `WAF_SITE_REVERSE_PROXY_WEBSOCKET` | Разрешает websocket-upgrade до upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_REVERSE_PROXY_KEEPALIVE` | Включает keepalive-соединения до upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_PASS_HOST_HEADER` | Передавать исходный `Host` заголовок на upstream. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_FORWARDED_FOR` | Добавлять `X-Forwarded-For`. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_FORWARDED_PROTO` | Добавлять `X-Forwarded-Proto`. | `true`, `false` | `true` | `true` |
| `WAF_SITE_SEND_X_REAL_IP` | Добавлять `X-Real-IP`. | `true`, `false` | `false` | `false` |

## HTTP-поведение и базовые лимиты

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ALLOWED_METHODS` | Список разрешённых HTTP-методов. | JSON-массив строк | `["GET","POST","HEAD","OPTIONS","PUT","PATCH","DELETE"]` | `["GET","POST","OPTIONS"]` |
| `WAF_SITE_MAX_CLIENT_SIZE` | Максимальный размер тела запроса. | Значение nginx-формата, например `10m`, `100m`, `1g` | `100m` | `50m` |
| `WAF_SITE_HTTP2` | Включает HTTP/2 на фронте. | `true`, `false` | `true` | `true` |
| `WAF_SITE_HTTP3` | Включает HTTP/3 на фронте. | `true`, `false` | `false` | `false` |
| `WAF_SITE_SSL_PROTOCOLS` | Разрешённые TLS-протоколы. | JSON-массив строк | `["TLSv1.2","TLSv1.3"]` | `["TLSv1.3"]` |

## HTTP-заголовки и browser security

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_COOKIE_FLAGS` | Шаблон флагов cookie, которые WAF добавляет или нормализует. | Строка | `* SameSite=Lax` | `* SameSite=Strict; Secure` |
| `WAF_SITE_CONTENT_SECURITY_POLICY` | Значение заголовка `Content-Security-Policy`. | Строка или пусто | Пусто | `default-src 'self'; img-src 'self' data:` |
| `WAF_SITE_PERMISSIONS_POLICY` | Список директив `Permissions-Policy`. | JSON-массив строк | `[]` | `["camera=()","geolocation=()"]` |
| `WAF_SITE_KEEP_UPSTREAM_HEADERS` | Какие upstream-заголовки сохранить без фильтрации. | JSON-массив строк | `["*"]` | `["X-Request-Id","X-Upstream-Time"]` |
| `WAF_SITE_REFERRER_POLICY` | Значение заголовка `Referrer-Policy`. | Строка | `no-referrer-when-downgrade` | `strict-origin-when-cross-origin` |
| `WAF_SITE_USE_CORS` | Включает CORS-ответы на фронте. | `true`, `false` | `false` | `true` |
| `WAF_SITE_CORS_ALLOWED_ORIGINS` | Какие origin разрешены по CORS. | JSON-массив строк | `["*"]` | `["https://app.example.com","https://admin.example.com"]` |

## Allowlist, exceptions и denylist

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_ALLOWLIST` | Включает allowlist-доступ на уровне сервиса. | `true`, `false` | `false` | `false` |
| `WAF_SITE_USE_EXCEPTIONS` | Включает исключения из блокирующей логики. | `true`, `false` | `false` | `true` |
| `WAF_SITE_ACCESS_ALLOWLIST` | Список IP/CIDR, которым разрешён доступ. | JSON-массив строк | `[]` | `["203.0.113.10","198.51.100.0/24"]` |
| `WAF_SITE_EXCEPTIONS_IP` | Список IP/CIDR, исключённых из части защитных правил. | JSON-массив строк | `[]` | `["203.0.113.15"]` |
| `WAF_SITE_ACCESS_DENYLIST` | Список IP/CIDR, которым доступ всегда запрещён. | JSON-массив строк | `[]` | `["198.51.100.23"]` |

## Bad behavior и эскалация блокировок

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_BAD_BEHAVIOR` | Включает счётчик плохого поведения по HTTP-ответам. | `true`, `false` | `true` | `true` |
| `WAF_SITE_BAD_BEHAVIOR_STATUS_CODES` | Какие коды ответа считать инцидентами bad behavior. | JSON-массив чисел | `[400,401,405,444]` | `[400,401,403,429,444]` |
| `WAF_SITE_BAD_BEHAVIOR_BAN_TIME_SECONDS` | Длительность первичной блокировки в секундах. | Целое число `>= 0` | `300` | `600` |
| `WAF_SITE_BAD_BEHAVIOR_THRESHOLD` | Порог срабатывания bad behavior. | Целое число `>= 0` | `120` | `60` |
| `WAF_SITE_BAD_BEHAVIOR_COUNT_TIME_SECONDS` | Окно подсчёта bad behavior. | Целое число `>= 0` | `120` | `60` |
| `WAF_SITE_BAN_ESCALATION_ENABLED` | Включает поэтапную эскалацию банов. | `true`, `false` | `false` | `true` |
| `WAF_SITE_BAN_ESCALATION_SCOPE` | На что распространяется эскалация бана. | `current_site`, `all_sites` | `all_sites` | `all_sites` |
| `WAF_SITE_BAN_ESCALATION_STAGES_SECONDS` | Длительности этапов эскалации. `0` означает перманентный бан и должен быть последним. | JSON-массив чисел | `[300,86400,0]` | `[600,3600,86400,0]` |

## Blacklist, DNSBL и сигнатурные списки

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_BLACKLIST` | Включает сигнатурные blacklists на сервисе. | `true`, `false` | `false` | `true` |
| `WAF_SITE_USE_DNSBL` | Включает DNSBL-проверки. | `true`, `false` | `false` | `true` |
| `WAF_SITE_BLACKLIST_IP` | Локальный blacklist IP/CIDR. | JSON-массив строк | `[]` | `["203.0.113.0/24"]` |
| `WAF_SITE_BLACKLIST_RDNS` | Blacklist по reverse DNS. | JSON-массив строк/паттернов | `[]` | `["scanner.example.net"]` |
| `WAF_SITE_BLACKLIST_ASN` | Blacklist по ASN. | JSON-массив строк или чисел | `[]` | `["AS12345"]` |
| `WAF_SITE_BLACKLIST_USER_AGENT` | Blacklist по User-Agent/regex. | JSON-массив строк | `[]` | `["curl/.*","sqlmap","HeadlessChrome"]` |
| `WAF_SITE_BLACKLIST_URI` | Blacklist по URI/regex. | JSON-массив строк | `[]` | `["/wp-admin","/\\.env","/phpmyadmin"]` |
| `WAF_SITE_BLACKLIST_IP_URLS` | URL-источники для внешних IP blacklist-списков. | JSON-массив URL | `[]` | `["https://lists.example.net/ip-deny.txt"]` |
| `WAF_SITE_BLACKLIST_RDNS_URLS` | URL-источники для внешних RDNS blacklist-списков. | JSON-массив URL | `[]` | `["https://lists.example.net/rdns-deny.txt"]` |
| `WAF_SITE_BLACKLIST_ASN_URLS` | URL-источники для внешних ASN blacklist-списков. | JSON-массив URL | `[]` | `["https://lists.example.net/asn-deny.txt"]` |
| `WAF_SITE_BLACKLIST_USER_AGENT_URLS` | URL-источники для внешних User-Agent blacklist-списков. | JSON-массив URL | `[]` | `["https://lists.example.net/ua-deny.txt"]` |
| `WAF_SITE_BLACKLIST_URI_URLS` | URL-источники для внешних URI blacklist-списков. | JSON-массив URL | `[]` | `["https://lists.example.net/uri-deny.txt"]` |

## Connection limiting и request limiting

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_LIMIT_CONN` | Включает ограничения по одновременным соединениям. | `true`, `false` | `true` | `true` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP1` | Лимит одновременных HTTP/1.x соединений. | Целое число `>= 0` | `200` | `80` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP2` | Лимит одновременных HTTP/2 соединений/потоков по модели продукта. | Целое число `>= 0` | `400` | `200` |
| `WAF_SITE_LIMIT_CONN_MAX_HTTP3` | Лимит одновременных HTTP/3 соединений/потоков по модели продукта. | Целое число `>= 0` | `400` | `200` |
| `WAF_SITE_USE_LIMIT_REQ` | Включает ограничение частоты запросов. | `true`, `false` | `true` | `true` |
| `WAF_SITE_LIMIT_REQ_URL` | Базовый путь, на который действует общий rate-limit. | Путь, начинающийся с `/` | `/` | `/` |
| `WAF_SITE_LIMIT_REQ_RATE` | Общая ставка rate-limit в формате `Nr/s`. | Строка вида `20r/s` | `120r/s` | `20r/s` |
| `WAF_SITE_CUSTOM_LIMIT_RULES` | Точечные rate-limit правила по отдельным путям. | JSON-массив объектов `{ "path": "/...", "rate": "Nr/s" }` | `[]` | `[{"path":"/login","rate":"10r/s"},{"path":"/api/2/envelope/","rate":"30r/s"}]` |

## Anti-bot и challenge

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_ANTIBOT_CHALLENGE` | Тип антибот-поведения. | `no`, `js`, `recaptcha`, `hcaptcha`, `turnstile` | `no` | `turnstile` |
| `WAF_SITE_ANTIBOT_URI` | Путь challenge-страницы. | Путь | `/challenge` | `/challenge` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SCORE` | Порог score для reCAPTCHA. | Число `0..1` | `0.7` | `0.8` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SITEKEY` | Site key reCAPTCHA. | Строка или пусто | Пусто | `recaptcha-site-key` |
| `WAF_SITE_ANTIBOT_RECAPTCHA_SECRET` | Secret reCAPTCHA. | Строка или пусто | Пусто | `recaptcha-secret` |
| `WAF_SITE_ANTIBOT_HCAPTCHA_SITEKEY` | Site key hCaptcha. | Строка или пусто | Пусто | `hcaptcha-site-key` |
| `WAF_SITE_ANTIBOT_HCAPTCHA_SECRET` | Secret hCaptcha. | Строка или пусто | Пусто | `hcaptcha-secret` |
| `WAF_SITE_ANTIBOT_TURNSTILE_SITEKEY` | Site key Cloudflare Turnstile. | Строка или пусто | Пусто | `turnstile-site-key` |
| `WAF_SITE_ANTIBOT_TURNSTILE_SECRET` | Secret Cloudflare Turnstile. | Строка или пусто | Пусто | `turnstile-secret` |

## HTTP Basic Auth

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_AUTH_BASIC` | Включает HTTP Basic Auth на сервисе. | `true`, `false` | `false` | `false` |
| `WAF_SITE_AUTH_BASIC_LOCATION` | Где применяется Basic Auth. | Обычно `sitewide` | `sitewide` | `sitewide` |
| `WAF_SITE_AUTH_BASIC_USER` | Имя пользователя Basic Auth. | Строка | `changeme` | `ops` |
| `WAF_SITE_AUTH_BASIC_PASSWORD` | Пароль Basic Auth. | Строка или пусто | Пусто | `strong-password` |
| `WAF_SITE_AUTH_BASIC_TEXT` | Текст realm для Basic Auth. | Строка | `Restricted area` | `Private admin area` |

## Геополитики

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_BLACKLIST_COUNTRY` | Список стран/региональных групп, которым доступ запрещён. | JSON-массив кодов стран и групп | `[]` | `["RU","CN","APAC"]` |
| `WAF_SITE_WHITELIST_COUNTRY` | Список стран/региональных групп, которым доступ разрешён. | JSON-массив кодов стран и групп | `[]` | `["DE","FR","PL"]` |

## ModSecurity и CRS

| Параметр | Назначение | Что можно указать | По умолчанию | Пример |
| --- | --- | --- | --- | --- |
| `WAF_SITE_USE_MODSECURITY` | Включает ModSecurity для сервиса. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_MODSECURITY_CRS_PLUGINS` | Подключает CRS plugins поверх базового CRS. | `true`, `false` | `true` | `true` |
| `WAF_SITE_USE_MODSECURITY_CUSTOM_CONFIGURATION` | Разрешает отдельный кастомный ModSecurity-конфиг. | `true`, `false` | `false` | `false` |
| `WAF_SITE_MODSECURITY_CRS_VERSION` | Версия CRS, которую нужно использовать. | Строка/номер, поддерживаемый продуктом | `4` | `4` |
| `WAF_SITE_MODSECURITY_CRS_PLUGINS` | Список CRS plugin IDs. | JSON-массив строк | `[]` | `["plugin-php","plugin-wordpress"]` |
| `WAF_SITE_MODSECURITY_CUSTOM_PATH` | Путь кастомного ModSecurity-конфига внутри runtime bundle. | Строка | `modsec/anomaly_score.conf` | `modsec/custom-rules.conf` |
| `WAF_SITE_MODSECURITY_CUSTOM_CONTENT` | Содержимое кастомного ModSecurity-конфига. | Многострочная строка или пусто | Пусто | `SecRuleEngine On` |

## Практические рекомендации

- Для экспорта и raw-редактирования лучше сохранять полный `.env`, а не только изменённые поля. Так проще переносить сервис между окружениями.
- `WAF_SITE_CUSTOM_LIMIT_RULES`, blacklist-массивы и geo-списки должны оставаться валидным JSON. Не смешивайте JSON и “через запятую” в одном значении.
- Если сервис использует HTTPS к upstream, включайте `WAF_SITE_REVERSE_PROXY_SSL_SNI` и задавайте `WAF_SITE_REVERSE_PROXY_SSL_SNI_NAME`, когда upstream ждёт конкретный `server_name`.
- Секреты вроде `*_SECRET`, `WAF_SITE_AUTH_BASIC_PASSWORD` и ACME-учётных данных лучше подставлять из безопасного секрета окружения, а не хранить в пересылаемом файле.

## Минимальный пример

```env
WAF_SITE_ID=app-example-com
WAF_SITE_PRIMARY_HOST=app.example.com
WAF_SITE_ENABLED=true
WAF_SITE_TLS_ENABLED=true
WAF_SITE_CERTIFICATE_ID=app-example-com-tls
WAF_SITE_SECURITY_MODE=block
WAF_SITE_UPSTREAM_ID=app-example-com-upstream
WAF_SITE_UPSTREAM_HOST=10.0.10.15
WAF_SITE_UPSTREAM_PORT=8080
WAF_SITE_UPSTREAM_SCHEME=http
WAF_SITE_USE_REVERSE_PROXY=true
WAF_SITE_REVERSE_PROXY_HOST=http://10.0.10.15:8080
WAF_SITE_REVERSE_PROXY_URL=/
WAF_SITE_ALLOWED_METHODS=["GET","POST","HEAD","OPTIONS"]
WAF_SITE_USE_LIMIT_REQ=true
WAF_SITE_LIMIT_REQ_URL=/
WAF_SITE_LIMIT_REQ_RATE=20r/s
WAF_SITE_CUSTOM_LIMIT_RULES=[{"path":"/login","rate":"10r/s"},{"path":"/api/","rate":"40r/s"}]
WAF_SITE_USE_MODSECURITY=true
WAF_SITE_MODSECURITY_CRS_VERSION=4
```
