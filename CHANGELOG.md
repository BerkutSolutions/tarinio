## [1.5.0] - 13.07.2026

### Ядро и runtime

- Добавлена сохраняемая настройка management-hosts с нормализацией DNS/IP, оптимистичной блокировкой, аудитом и включением в snapshot ревизии. Она определяет management-site независимо от fast-start переменных и общего upstream.
- В default compose `localhost` является встроенным management-host; для внешних DNS/IP-хостов применяется сохранённая настройка management-hosts.
- Access policy не может запретить доступ к management-host: это исключает самоблокировку панели правилом `deny` после сохранения политики доступа.
- L4 guard больше не выполняет тяжёлый bootstrap каждые пять секунд: launcher сверяет отпечаток L4-конфигурации и adaptive-ban файла, а bootstrap не переписывает совпадающие iptables-цепочки и jump-правила.
- Расширены проверки startup, revision compile/apply, TLS и Anti-DDoS; обновлены API метаданных приложения и RBAC для управления management-hosts.

### ModSecurity и защита

- Внутренние `/api/*` management-host получают отдельный nginx location с `modsecurity off` и прямым proxy к control-plane; easy-конфигурация management-host больше не включает ModSecurity. Это исключает блокировку users, access-policies, revisions и других административных операций. Обычные сайты сохраняют WAF-защиту.
- Исправлена генерация structured ModSecurity exclusions для `REQUEST_URI`: правило исключается по ID в ограниченном запросе, а не удаляется некорректная цель правила.
- Добавлены настройки и проверки security modes, anti-bot, error pages, виртуальных патчей и L4-защиты.

### UI и API

- В Settings добавлено управление management-hosts; интерфейс, API-контракты, локализации `ru`, `en`, `de`, `sr`, `zh` и документация обновлены.
- Истёкшая или отсутствующая сессия панели теперь сразу открывает `/login?reason=…`; фронтенд больше не инициирует `/challenge` при перезагрузке контейнеров, ошибке загрузки ассетов или ответе авторизации.
- Исправлена работа редактора `localhost`: UI больше не подменяет сохранённый ID сайта устаревшим `control-plane-access`, поэтому сохранение access policy и профиля сервиса снова адресуется существующему сайту.
- Доработан редактор ModSecurity exclusion rules в Services, включая сохранение draft-состояния, валидацию, подсказки и многоязычные тексты.
- Обновлены страницы Services и Settings, E2E-контракты панели, обработка сессий, onboarding и отображение событий безопасности.

### Тесты и релиз

- Добавлен полный E2E панели на изолированном `e2e-management.test`: через nginx с включённым ModSecurity проверяются все зарегистрированные WAF API-маршруты. Контракт mux/catalog завершится ошибкой при добавлении endpoint без покрытия.
- E2E защиты панели включён в обязательные preflight-проверки обоих release entrypoint’ов.
- Release preflight учитывает все 14 обязательных проверок, включая выделенный E2E management-панели; runtime image собирает launcher вместе с fingerprint-логикой L4 guard.
- Восстановлены и расширены реальные runtime E2E для ModSecurity, administration CRUD, profile roundtrip, dashboard, Anti-DDoS, TLS и API-матриц.
