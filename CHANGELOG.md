## [2.0.8] - 23.04.2026

### Установка и обновление
- Release-синхронизация доведена до `2.0.8`: версия продукта, UI и release metadata обновлены согласованно.
- `install-aio.sh` и `install-aio-enterprise.sh` дополнительно упрощены в финальном footer-блоке, чтобы сценарий `curl | sh` не завершался ложным `[FAIL]` после успешно пройденного health gate.
- Сохраняется безопасный upgrade-путь для существующих standalone-инсталляций: старые секреты не ротируются без необходимости, `POSTGRES_DSN` нормализуется под фактические текущие credentials.

### Onboarding и первый apply
- Onboarding standalone-контура уже включает исправленный порядок `ACME http-01`: первый `compile/apply`, затем выпуск сертификата, затем финальный TLS cutover.
- Первый apply поверх bootstrap-runtime остаётся исправленным: relink корректно заменяет bootstrap-каталоги и больше не валит `/reload` на непустых директориях.
- Во время onboarding после первого apply сохраняется доступность служебных admin-маршрутов (`/api/*`, `/login`, `/onboarding/*`, `/static/*`), чтобы браузер не упирался в `Failed to fetch`.

### Импорт сервисов
- Импорт `.env` и `.json` продолжает работать в staged-режиме без промежуточного auto-apply на каждом `POST/PUT`.
- После завершения импорта выполняется один финальный `compile/apply`, поэтому импорт не создаёт каскад промежуточных ревизий и не шумит частично собранными конфигурациями.
- Проверка существующих site/upstream/tls-ресурсов при импорте опирается на инвентарь списков, а не на серию лишних `404` probe-запросов.

### Anti-bot: ранее сделанные исправления
- Easy anti-bot и rate-limit protection уже исключают служебные admin UI маршруты, чтобы залогиненный администратор не мог случайно заблокировать себе доступ статикой, SPA-route навигацией и частыми `/api/*` запросами.
- Для admin host сохраняются bypass-правила по `waf_session` и `waf_session_boot`, чтобы anti-bot и easy access-guard не стреляли по самому control-plane.

### Anti-bot: новые изменения в 2.0.8
- Для easy anti-bot добавлен interstitial browser-check flow вместо жёсткого `403` на GET/HEAD-запросах без challenge-cookie.
- Режим `javascript` теперь реально открывает challenge-страницу в стиле Cloudflare/Pastebin: страница выполняет client-side проверку, вызывает verify endpoint, получает site-scoped anti-bot cookie и возвращает пользователя на исходный URL.
- Easy challenge routes расширены verify endpoint-ом, который завершает проверку, ставит cookie и делает redirect обратно на исходный путь.
- Провайдерные режимы (`captcha`, `recaptcha`, `hcaptcha`, `turnstile`, `mcaptcha`) пока ведут через тот же interstitial/browser-check flow, а не через полноценную backend-валидацию внешнего challenge token.

### Диагностика runtime
- Ошибки `waf-runtime-l4-guard` продолжают подниматься с реальным stderr helper-а вместо одного только `exit status 1`.
- Ошибки failed apply в onboarding уже показывают фактическую причину failed-job, а не общий `Initial apply failed`.

### Документация
- `CHANGELOG.md` очищен до компактного release-summary и теперь явно разделяет ранее внесённые anti-bot исправления и новый challenge interstitial flow.
