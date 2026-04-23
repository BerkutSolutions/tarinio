## [2.0.9] - 23.04.2026

### Установка и релиз
- Release-синхронизация доведена до `2.0.9`: версия продукта, UI, документации и release metadata обновлены согласованно.
- Release summary в `CHANGELOG.md` сокращён и пересобран под актуальное состояние релиза без накопленного шума из промежуточных фиксов.
- Добавлены лёгкие `scripts/push.ps1` и `scripts/push.sh` для обычного source push без полного release-контура с тегами и Docker publish.

### Anti-bot challenge
- Easy anti-bot interstitial доведён до полноценного browser-redirect flow: после client-side проверки страница переводит браузер на verify endpoint обычным переходом, а не фоновым `fetch`.
- Verify endpoint продолжает ставить site-scoped anti-bot cookie и возвращать пользователя на исходный URL, включая исходный path и query string.
- Challenge page оформлена как полноценная status/interstitial страница с явным состоянием проверки и визуальным progress/spinner вместо пустого ответа на `/challenge`.

### Admin UI и ложные атаки
- В runtime request archive больше не попадает служебный TARINIO admin-трафик с публичного admin host: служебные `/api/*`, `/static/*`, `favicon`, dashboard/menu routes и challenge-маршруты не раздувают пользовательскую request-статистику.
- Генерация security events обновлена так, чтобы admin UI активность на публичном host не выглядела как атаки и блокировки по боевому сайту.
- Dashboard backend перестал считать служебные TARINIO admin routes за пользовательские атаки и заблокированные запросы, даже если admin UI опубликован как обычный site вроде `waf.hantico.ru`.
- Клиентская детализация dashboard синхронизирована с backend-логикой и теперь не показывает админские `/api/*`, `/static/*`, SPA routes и challenge endpoints как атакующие страницы/IP.

### Onboarding, импорт и совместимость
- Standalone onboarding сохраняет исправленный порядок первого `compile/apply`, ACME `http-01` и финального TLS cutover.
- Первый apply поверх bootstrap-runtime по-прежнему корректно заменяет bootstrap-каталоги и не падает на непустых директориях.
- Импорт `.env` и `.json` остаётся staged: без промежуточного auto-apply на каждом `POST/PUT`, с одним финальным `compile/apply`.

### Диагностика
- Ошибки `waf-runtime-l4-guard` продолжают подниматься с реальным stderr helper-а вместо одного только `exit status 1`.
- Ошибки failed apply в onboarding по-прежнему показывают фактическую причину failed-job, а не общий `Initial apply failed`.
