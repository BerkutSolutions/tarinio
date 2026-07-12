## [1.4.9] - 30.06.2026

### Ядро

- Release preflight теперь печатает и валидирует ожидаемое, фактически выполненное и успешно пройденное число startup-проверок. Это не позволяет скрыть удаление или пропуск целой проверки в release-скрипте.
- В режиме `monitor` ModSecurity больше не отключается из runtime-конфигурации: профиль и rules file остаются подключёнными с `SecRuleEngine DetectionOnly`. Это сохраняет детектирование и журналирование событий без блокировки запросов; режим `transparent` по-прежнему полностью отключает ModSecurity.
- Для management/UI host добавлен жёсткий self-management safeguard: `/login`, `/login/2fa`, admin UI routes и `/api/*` самой панели теперь всегда считаются защищёнными внутренними route-исключениями и не должны попадать под собственный ModSecurity/CRS blocking path, чтобы WAF не мог заблокировать собственные mutation endpoints.
- Уточнена генерация management/admin protected paths: `localhost` считается management-site только при явном `CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID`, а шаблоны внутренних management-маршрутов нормализуются без дублирования. Это устраняет ложный management-режим для обычного `localhost` site и регрессии с duplicate `location` для внутренних admin/login route.

### UI

- Логи Anti-DDoS событий и live-логи Dashboard теперь ставят polling на паузу в скрытых вкладках, не допускают наложения параллельных refresh-запросов и чисто возобновляют обновление после возврата вкладки в фокус, снижая лишнюю нагрузку на UI/API.

- Истекшие management-сессии теперь возвращают пользователя через antibot challenge перед показом логина, а устаревшие вкладки `login` и `login-2fa` автоматически обновляют доступ к challenge, чтобы CSS и сам вход не деградировали до страницы без стилей или позднего `403` после простоя.

- Настройки безопасности easy-site теперь можно редактировать и сохранять даже при отключенном antibot или когда сайт работает в режимах `transparent`/`monitor`. Они по-прежнему хранятся как preset профиля и применяются к runtime только в режиме `block`.

- Добавлен live regression-сценарий для ModSecurity, который проверяет цикл enable → disable → re-enable через Services API и реальный runtime compile/apply, включая структурированное exclusion-правило, разрешающее только предназначенный для него URI.

- Management UI ModSecurity safeguard теперь имеет явное regression-покрытие для аутентифицированного `POST /api/app/ping`: кастомное deny-правило не может заблокировать доступ администраторам, при этом обычные сайты не получают management-bypass.
- Во вкладке Services → Site editor → ModSecurity добавлен полноценный редактор exclusion rules с добавлением/удалением строк без перезагрузки: UI теперь сохраняет path/path pattern, mode, methods, rule IDs, targets и comment в draft-состоянии вместо потери данных при ререндере.
- Для нового ModSecurity exclusions UI добавлены строки локализации во всех поддерживаемых языках (`ru`, `en`, `de`, `sr`, `zh`), чтобы подписи и help-тексты не выпадали в raw i18n keys.
- Для ModSecurity exclusions UI дополнительно выровнены validation keys во всех поддерживаемых языках, а русские help-строки очищены от английских слов, чтобы UI проходил i18n quality gate без raw keys и language-artifact предупреждений.

### Тесты

- Compiler/runtime regression-покрытие для security modes теперь фиксирует, что `monitor` публикует и подключает ModSecurity rules file с `DetectionOnly`, а `transparent` не публикует ModSecurity-артефакт. Для Windows добавлен PowerShell entrypoint штатного Docker e2e-стека.
- Добавлены regression tests на self-management bypass pattern: management-site обязан покрывать `/login`, `/login/2fa`, `/dashboard`, `/services`, `/api/sites/*` и `/api/access-policies/*`, при этом обычные site routes не получают такой bypass автоматически.
- UI contract tests переведены на актуальные маркеры модульного frontend/runtime-контракта: проверяются onboarding guard и auto-apply flow, sidebar/menu, anti-ddos и requests routes, dashboard widget/frame markers, settings storage/logging panels и stable/modular sites page markers вместо устаревших строк старого UI.
- Compiler regression tests для ModSecurity дополнительно фиксируют, что management-site получает self-management safeguard только в management nginx config, safeguard исчезает при `UseModSecurity=false`, CRS-версии и plugin toggles не ломают internal bypass contract, а обычные сайты не наследуют management safeguard markers автоматически.
- Live/runtime regression tests теперь прогоняют `transparent`/`monitor`/`block` на выделенном easy-site через реальный compile/apply и детерминированный ModSecurity trigger: неблокирующие режимы обязаны сохранять профиль без runtime-deny, а возврат в `block` снова включает реальную блокировку по probe-запросу.
