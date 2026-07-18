## [1.5.5] - 17.07.2026

### Dashboard telemetry

- Requests now provide a localized block-reason selector. Authentication and anti-bot failures are recorded as exact `auth` and `antibot` security reasons, while successful authentication remains a normal request.
- Removed the customer-facing quick security summary, legacy-compatibility labels, and `legacy_row_type_support` response field from Requests. Compatibility inference remains server-side only.
- The 24-hour requests/attacks chart now labels its middle Y-axis tick as half of the peak value rather than the sparse-window arithmetic mean; a peak of 54 therefore shows a middle tick of 27.
- Dashboard request widgets now use the authenticated runtime aggregate for the same complete 24-hour observation window. Totals, hourly series, top sites/URLs, unique IPs, error statuses, and request-derived security metrics no longer depend on the paginated request-list limit.
- Runtime exposes the protected `GET /requests/dashboard-summary` endpoint for this aggregate. It applies the same panel-traffic exclusion as the dashboard while preserving protected-service security events.
- Successful ModSecurity access denials are no longer classified as container-health errors; they remain available as blocked security telemetry.
- Top attacker IPs now include their observed country, so dashboard IP rows and click-through details remain informative even when the paginated request feed does not contain that historical row.

### Installer

- Fixed the UI image build: its container-side contract tests now receive every nginx configuration and the UI Dockerfile they inspect.

### DevSecOps CI

- GitLab CI now gates the private source repository with repository validation, full Go tests, a separate documentation build, standard and clean-onboarding Docker E2E suites, Gitleaks, Trivy, govulncheck, and npm audit. No release or GitHub publication job is enabled.
- The GitLab runner keeps npm packages and required E2E base images outside the disposable checkout; documentation and dependency-audit jobs reuse the package cache, while E2E starts only after PostgreSQL, Vault, and echo-server images are present.
- Gitleaks uses reviewed occurrence-level history exceptions without suppressing future matches, and govulncheck is pinned to the Go 1.26-compatible v1.6.0 release.
- Trivy keeps its full vulnerability, secret, and configuration scan. Existing reviewed infrastructure exceptions are tracked per file and rule; new or expanded high/critical findings fail CI.

### Dependency security

- **CVE-2026-54466 / CWE-130.** Updated the transitive documentation-development dependency `websocket-driver` to 0.7.5 through the root npm override. This rejects malformed oversized WebSocket length headers.

### E2E runner

- Windows PowerShell E2E runner now captures Docker BuildKit stdout/stderr safely and determines success from the Docker exit code, so normal build progress no longer aborts the test run.

### Management hosts and request security telemetry

- Initial onboarding persists its panel domain in management hosts before the first revision is compiled; CRS therefore cannot lock the control-plane API after initial setup.
- Requests now retain 403/444 blocks against the management host as security records while ordinary management UI/API traffic remains excluded.
- Global Anti-DDoS L7 limits now exclude every explicitly configured management host, including installations whose panel has a custom service ID; onboarding and the control-plane API cannot self-rate-limit with HTTP 429.
- The management shell and onboarding no longer replace their DOM with the legacy synthetic 429 fallback after a background API request or a failed static asset; real rate-limit responses remain handled by the branded runtime error page.
- The UI image normalizes readable static-asset permissions after copy, so onboarding and the management shell cannot degrade to a raw nginx 403 when source-file ACLs are restrictive.
- A fresh HTTP onboarding now uses a dedicated session-cookie pair. It remains valid even if the browser retains stale Secure cookies from a deleted previous deployment, and switches to the normal Secure session on HTTPS login.

### E2E onboarding

- The E2E runner supports a clean onboarding mode without a seeded administrator or fast-start revision. It verifies first-run bootstrap, self-signed TLS issue and binding, revision apply, and authenticated HTTPS login.

### Исправления уязвимостей и защита данных

#### Учётные записи, права и секреты

- **CWE-287, CWE-307, CWE-362.** Экспорт приватного материала сертификата защищён свежей TOTP-проверкой, одноразовым подтверждением другого пользователя и общей атомарной фиксацией ошибок. В «Настройки → Безопасность» это подтверждение можно отключить явно; после обновления оно остаётся включённым по умолчанию.

#### Цепочка поставки и обновления

- **CWE-494.** Обновления набора правил WAF получают контрольную сумму точного архива из официального релиза, проверяют источник и SHA-256 до атомарной активации. Пользователь больше не обязан задавать хэши вручную.

#### Инфраструктура, сети и хранение

- **CWE-200.** Публичный статус первичной настройки больше не раскрывает состав развёртывания; существующая установка после обновления всегда возвращается на страницу входа, а не в первичную настройку.
- **CWE-284, CWE-319, CWE-306.** Хранилища в корпоративных и лабораторных профилях требуют TLS, отдельные учётные записи и ограниченные сетевые политики. Существующие тома, ключи, сертификаты и хэши пользователей не перезаписываются.
- **CWE-693.** Среда выполнения сквозных тестов ожидает атомарно опубликованный и проверенный стартовый набор файлов, поэтому не запускается между публикацией TLS-конфигурации и ключевого материала.

#### Согласованность операций и защита интерфейса

- **CWE-287, CWE-307, CWE-362.** TOTP-подтверждение привязано к сессии; пять ошибочных проверок блокируются на 15 минут, повторные ошибки повышают срок блокировки до одного часа.
- **CWE-284, CWE-306, CWE-319.** Лабораторные хранилища требуют TLS, аутентификацию и точные сетевые политики; наблюдаемость доступна только локально и не запускается с типовыми паролями.
- **CWE-362.** Запрос на экспорт приватного ключа связан с конкретными сертификатами, подтверждается другим пользователем и расходуется ровно один раз.
- **CWE-693.** Виртуальные патчи создаются и удаляются только после успешной компиляции и применения ревизии; устаревшая среда выполнения не маскируется успешным ответом.
- **CWE-601.** После внешнего входа разрешён только канонический относительный путь текущей панели; внешние и закодированные обходы отклоняются.
- **CWE-295.** Клиенты хранилищ поддерживают проверку собственного центра сертификации и не требуют отключать проверку TLS.
- **CWE-284, CWE-306, CWE-319.** Координация высокой доступности использует TLS, ограниченную ACL и сетевую изоляцию; исходящие подключения к хранилищам разрешены только к доверенным адресам без перенаправлений.
- **CWE-200, CWE-522.** Сохранённые данные панели больше не доступны через статические nginx-пути; диагностика и экраны мониторинга не раскрывают токены, секреты и параметры процессов.
- **CWE-798, CWE-732.** Удалены встроенные пароли; прикладные токены заменили корневые, а `.env`, резервные копии и диагностические файлы создаются только с правами владельца.
- **CWE-284.** Экспорт приватного материала сертификатов требует отдельного разрешения; изменение критичных адресов и настроек секретов не допускает сохранения маскированного секрета в новом месте.
- **CWE-400.** Ограничены размеры ответов, архивов, метрик, проверок состояния, истории запросов и выдачи событий; удалённые операции не удерживают критические блокировки.
- **CWE-74, CWE-22.** Ввод для nginx, ModSecurity, путей, идентификаторов, целевых сервисов и доверенных прокси валидируется до генерации конфигурации; обновления правил WAF проверяются до активации.
- **CWE-362.** Одноразовые проверки, создание первого администратора, журнал аудита и распределённые блокировки выполняются атомарно в общем хранилище.

### Другие изменения

- При частом обновлении страницы неподтверждённый браузер сначала получает антибот-проверку, а не внутреннюю резервную страницу `429`; подтверждённый клиент по-прежнему ограничивается при превышении квоты.
- HTTP-01 проверка сертификата выдаёт токен до появления домена в активной карте хостов и не подменяется страницами WAF.
- Настройки прямого доступа к IP WAF и политики экспорта сертификатов сохраняются одной кнопкой раздела «Безопасность».
- Ошибки обновления CRS локализованы на русском, английском, немецком, сербском и китайском языках.
- Локальная проверка перед выпуском передаёт защищённый публичный ключ генератора в проверку артефактов, поэтому режим «Проверить и опубликовать» не останавливается перед публикацией из-за отсутствующего параметра.

### Совместимость и проверки

- Обновление не перезаписывает существующие ключи, сертификаты, тома, хэши пользователей и сохранённые настройки.
- Добавлены регрессии для TOTP, экспорта сертификатов, CRS, ACME HTTP-01, прямого IP, виртуальных патчей и порядка антибот-проверки/ограничения частоты; ручные проверки перечислены в `.work/UI_CHECKLIST.md`.
- Сквозная проверка панели управления с ModSecurity ждёт фактического применения ревизии runtime, поэтому не даёт ложный сбой из-за обращения к предыдущей конфигурации.
### UI previews (2026-07-18)

- Added ten standalone, responsive Basic Auth page concepts under `.work/sidebar-previews`; they are asset-free and intended for visual selection before product integration.
### UI previews (2026-07-18)

- Added five Basic Auth concepts informed by the current login layouts and five informed by the current anti-bot verification layouts under `.work/sidebar-previews`.
### Basic Auth appearances (2026-07-18)

- Added nine selectable, branded Basic Auth page appearances with authenticated previews and runtime artifact generation.
- Added persistence, server validation, compiler coverage, and E2E coverage for changing and using the selected Basic Auth page.
### Basic Auth appearance refinements (2026-07-18)

- Localized the runtime Basic Auth pages for `en`, `ru`, `de`, `sr`, and `zh`; the footer now links to Tarinio on GitHub.
- Refined logo use, contrast, alignment, and preview-button positioning across the selected Basic Auth and anti-bot appearance controls.
### Basic Auth preview polish (2026-07-18)

- Centered anti-bot and Basic Auth preview controls against their template selectors and renamed Basic Auth choices to neutral numbered variants.
- Refined the light-blue `v4`, restored `v6` copy, and adjusted `v7`–`v9` branding and layout details.
