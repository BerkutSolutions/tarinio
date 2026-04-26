# Security Profiles

TARINIO `3.0.6` включает 5 встроенных security-профилей сервиса:

1. `strict`
2. `balanced`
3. `compat`
4. `api`
5. `public-edge`

Профиль — это базовый preset. Оператор может оставить профиль и добавить свои override-параметры.

## Как работает применение профиля

- Выбранный профиль записывается в `front_service.profile` и применяет набор базовых настроек.
- `balanced` — профиль по умолчанию (база из `DefaultProfile(...)`).
- Остальные профили накладываются поверх этой базы.
- Ручные правки в UI/API сохраняются и не «сбрасываются» при обычном сохранении.

## Подробно по каждому профилю

### `strict`

Используйте, когда допустима более агрессивная защита и возможны редкие ложные срабатывания.

Ключевые значения preset:

- `front_service.security_mode = block`
- `security_behavior_and_limits.use_bad_behavior = true`
- `bad_behavior_status_codes = [400,401,403,404,405,429,444]`
- `bad_behavior_threshold = 60`
- `bad_behavior_count_time_seconds = 60`
- `bad_behavior_ban_time_seconds = 900`
- `use_limit_req = true`, `limit_req_url = /`, `limit_req_rate = 80r/s`
- `use_limit_conn = true`, `http1/http2/http3 = 120/220/220`
- `security_antibot.antibot_challenge = javascript`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

### `balanced`

Профиль по умолчанию для большинства production-сайтов.

Базовые значения (часть может настраиваться через `WAF_DEFAULT_*`):

- `front_service.security_mode = block`
- `use_bad_behavior = true`
- базовый набор bad-behavior кодов (ориентирован на явные злоупотребления)
- `use_limit_req = true`, базовый `limit_req_rate = 120r/s`
- `use_limit_conn = true`, базовые `200/400/400`
- `security_antibot.antibot_challenge = no`
- `security_api_positive.use_api_positive_security = false`
- `security_modsecurity.use_modsecurity = true`

Для management-site базовые лимиты автоматически мягче и фокусируются на `/api/`, чтобы уменьшать ложные срабатывания в admin UI.

### `compat`

Профиль для legacy-приложений, где важнее совместимость и мягкое внедрение.

Ключевые значения preset:

- `front_service.security_mode = monitor`
- `use_bad_behavior = false`
- `use_limit_req = false`
- `use_limit_conn = true`, `http1/http2/http3 = 300/500/500`
- `security_antibot.antibot_challenge = no`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

### `api`

Профиль для API-first сервисов.

Ключевые значения preset:

- `front_service.security_mode = block`
- `http_behavior.allowed_methods = GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD`
- `http_headers.use_cors = true`
- `http_headers.cors_allowed_origins = ["*"]`
- `use_limit_req = true`, `limit_req_url = /api/`, `limit_req_rate = 200r/s`
- `security_antibot.antibot_challenge = no`
- `security_api_positive.use_api_positive_security = true`
- `security_api_positive.enforcement_mode = monitor`
- `security_api_positive.default_action = allow`

### `public-edge`

Профиль для публичных edge-точек с повышенным риском ботов и злоупотреблений.

Ключевые значения preset:

- `front_service.security_mode = block`
- `use_bad_behavior = true`
- `bad_behavior_status_codes = [400,401,403,404,405,429,444]`
- `bad_behavior_threshold = 80`
- `bad_behavior_count_time_seconds = 60`
- `bad_behavior_ban_time_seconds = 600`
- `use_blacklist = true`
- `use_dnsbl = true`
- `use_limit_req = true`, `limit_req_url = /`, `limit_req_rate = 100r/s`
- `security_antibot.antibot_challenge = javascript`
- `security_modsecurity.use_modsecurity = true`
- `security_modsecurity.use_modsecurity_crs_plugins = true`

## Как выбрать профиль

- `strict` — когда нужна максимальная жёсткость на внешнем периметре.
- `balanced` — универсальный выбор по умолчанию.
- `compat` — для сложных legacy-систем при миграции.
- `api` — для API-шлюзов и сервисов с OpenAPI/endpoint-политиками.
- `public-edge` — для публичных входных точек с высоким bot-pressure.

## Практические замечания

- Профиль не блокирует ручные изменения: все поля остаются редактируемыми.
- `security_auth_basic.users[]` и `session_inactivity_minutes` настраиваются отдельно от выбора профиля.
- В RAW-редакторе и API профиль хранится в `front_service.profile`.
