## [3.0.2] - 24.04.2026

### Пакет Security Benchmark и документация
- Добавлен отдельный контур документации benchmark-пакета:
  - `docs/eng/security-benchmark-pack/README.md`
  - `docs/ru/security-benchmark-pack/README.md`
- Добавлены навигационные ссылки на benchmark-пакет в точках входа документации.
- Реализован benchmark/scenario-контур и CI-гейты для mixed/human/scanner/flood/anti-bot сценариев.

### Профили сервисов и каталог профилей
- В сервисы добавлено поле `profile` (модель/API/UI).
- В таблицу сервисов добавлена колонка `Profile`.
- Добавлен каталог из 5 встроенных security-профилей с полной документацией:
  - `strict`
  - `balanced`
  - `compat`
  - `api`
  - `public-edge`

### Позитивная безопасность API
- Добавлен контур OpenAPI schema enforcement.
- Добавлены policy-настройки per-endpoint/per-token.

### Anti-Bot и Sentinel (объяснимость и challenge-контур)
- Реализован двухслойный anti-bot challenge.
- Добавлены per-URL challenge-правила в UI (по аналогии с URL rate-limit правилами).
- Расширена explainability Sentinel (`reason_codes`, `top_signals`, операторские пояснения).

### Threat Intel, гео и безопасность поставки
- Добавлена интеграция threat-intel/reputation feeds.
- Логи расширены с country до country+city без регресса существующего поведения.
- Усилен supply-chain security-контур: подписи, provenance и security-гейты.

### UX дашборда
- Исправлено сохранение позиции и размеров виджетов после drag/resize: layout сохраняется сразу при изменении и дополнительно при `pagehide`.
- Дефолтная позиция `Unique attacker IPs` перенесена под `Blocked attacks`.

### Публичные страницы и локализация
- Исправлены битые локализации (`????`/кракозябры) на публичных страницах:
  - `compiler/templates/errors/antibot.html.tmpl`
  - `compiler/templates/errors/403.html.tmpl`
  - `compiler/templates/errors/50x.html.tmpl`
  - `compiler/templates/errors/status.html.tmpl`
- Исправлены битые строки в fallback-блоке UI:
  - `ui/app/static/js/app.js`
- Исправлены названия языков в переключателе:
  - `ui/app/static/js/i18n.js`

### Язык браузера для входных страниц
- Для страниц входа и первичной проверки язык теперь выбирается по языку браузера при загрузке:
  - `ui/app/static/js/login.js`
  - `ui/app/static/js/login-2fa.js`
  - `ui/app/static/js/healthcheck.js`
  - `ui/app/static/js/onboarding.js`
- Добавлен экспорт определения языка браузера в i18n-слой:
  - `ui/app/static/js/i18n.js`

### Anti-Bot redirect hardening
- Устранён некорректный redirect в `/challenge/verify` при `return_uri=%2F` (невалидный `https://localhost%2F`).
- Валидация `return_uri` усилена: encoded-path значения и не-абсолютные пути сбрасываются на безопасный `/`.

### HTTP Auth: мультипользовательские настройки (UI + профиль)
- В разделе сервиса добавлено управление пользователями auth вместо одного логина/пароля:
  - таблица пользователей (`username`, `password`, `enabled`, `last_login`);
  - добавление/удаление пользователей;
  - сохранение `users[]` в `security_auth_basic`.
- Добавлен выбор TTL неактивности сессии (`session_inactivity_minutes`) с пресетами:
  - `5, 10, 15, 30` минут;
  - `1..24` часа;
  - `без ограничений`.
- Сохранена обратная совместимость с legacy-полями `auth_basic_user/auth_basic_password`.

### HTTP Auth: совместимость и runtime-поведение
- В компиляторе включена генерация `htpasswd` для всех активных пользователей профиля (а не только одного пользователя).
- Для standalone-сценария `site_id=localhost` admin-path bypass теперь трактуется как management-site, чтобы anti-bot не перехватывал служебные login-пути control-plane.
- Для `auth_basic` добавлен bypass на служебных путях (`/login`, `/login/2fa`, challenge endpoint’ы), чтобы убрать повторный auth prompt после редиректов.

### Локализация и стабильность UI
- Исправлены i18n-артефакты в новых ключах антибота/аутентификации для `ru`:
  - `sites.easy.antibot.authSessionTtl`
  - `sites.easy.antibot.authSessionTtlHint`
- Добавлены отсутствующие ключи антибота/аутентификации в дополнительные локали `de`, `sr`, `zh`:
  - `sites.easy.antibot.frameTitle`
  - `sites.easy.antibot.authSectionTitle`
  - `sites.easy.antibot.authSessionTtl`
  - `sites.easy.antibot.authSessionTtlHint`
  - `sites.easy.antibot.authSessionUnlimited`
  - `sites.easy.antibot.authUsers*`
  - `sites.validation.authBasicPasswordRequired`
- Восстановлен прохождение полного smoke-чека `go test ./... -count=1` (падал на i18n полноте/качестве локалей).

### Публичные страницы: footer spacing
- Для публичных страниц (`auth/challenge/status/403/429/50x`) выровнен нижний отступ подписи `Secured by Tarinio` (не прилипает к краю экрана).
