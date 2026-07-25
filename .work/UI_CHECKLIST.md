2026-07-21

2026-07-25

- Requests outage state: open Requests on an isolated stack, stop the runtime backend, click Refresh, and verify that the loading placeholder changes to the localized load-error message while navigation and the application shell remain usable.
- Requests recovery: restore the runtime backend, refresh Requests again, and verify that the normal table or empty state returns without a page reload or stale loading indicator.
- CI/API: run the browser `requests-backend-failure` job and confirm the pre-security report includes its focused failure excerpt and reports no skipped browser executions.

2026-07-24 — E2E re-audit stages 12–14

- [x] Activity/Audit: `audit-stage12` passed 3 real Go API/runtime subtests and 6 desktop/mobile Playwright checks, with no retry or skip.
- [x] Settings and cross-module/security: `settings-cross-stage13-14` passed 8 real Go E2E suites plus 28 desktop/mobile browser checks (including auth setup), with no retry or skip.
- [x] Negative security probes: four hostile requests were blocked by runtime with HTTP 403 and never reached the real canary; the benign request reached it exactly once.
- [>] Full release proof remains required: full tagged Go E2E exceeded the enforced 10-minute limit before completion; repeat after runner optimization, all CI shards, and exact aggregate coverage must be green before closing stages.

2026-07-24 — Этап 5 Bans повторно подтверждён на изолированном WAF стенде.

- Bans UI: на desktop и mobile создать ban для TEST-NET IP, открыть detail через keyboard, проверить country/detail, Extend, Cancel и Unban. После Unban строка должна исчезнуть, а denylist API и audit должны показать фактическую мутацию.
- Bans RBAC/API: отдельная read-only сессия должна видеть отказ `403` при Create/Extend/Unban без изменения denylist; invalid IP и missing site/duration должны быть отвергнуты сервером.

2026-07-23 — Browser E2E continuation: проверить в disposable stack Services export download (JSON содержит sites/upstreams/tls_configs), invalid import rejection и возврат в редактор; Revisions clear-statuses не должен показывать backend error; Bans create modal cancel должен закрываться без mutation. Проверить Requests page size, сортировку и открытие detail с клавиатуры/Escape на desktop/mobile.

- Services mutations: создать изолированный service, изменить host через UI Save и убедиться по API list, что значение сохранилось; удалить через editor и подтвердить отсутствие. Для двух изолированных сервисов проверить bulk Delete Cancel (объекты остаются) и Confirm (оба удаляются).
- Bans mutation: для уникального TEST-NET IP выполнить Create, Extend и Unban; denylist должен изменяться в API, строка исчезать после unban, а audit содержать `accesspolicy.unban`.
- Administration mutation: создать и изменить изолированного user и role через modal, подтвердить API readback и permission selection; проверить, что modal permission clicks не перекрываются sidebar. Зафиксировать отсутствие Delete-кнопок как product gap, не подменять его API cleanup.
- TLS mutation: создать certificate metadata через UI, связать с изолированным site, подтвердить оба объекта через API, затем удалить binding и certificate через UI и проверить cleanup.
- Settings persistence: изменить `update_checks_enabled` через UI, подтвердить API и состояние после reload, затем восстановить исходное значение и повторно подтвердить API.
- Anti-DDoS persistence: переключить model enabled, Save, проверить API/reload; задать invalid L7 status code 99 и убедиться, что API не изменился; восстановить полный исходный payload.

- Login first paint: with cache disabled in DevTools, hard-refresh `/login` and `/login/2fa` for every saved appearance. Before the public appearance endpoint answers, the page may be blank but must never show the legacy centred login card; the first visible frame must use the selected theme.
- Login first paint fallback: temporarily block `/api/public/login-appearance` and reload both login routes. The default themed appearance may be shown after the request fails, but no legacy layout may be painted at any point.
- Login CSP: reload `/login` and `/login/2fa` with Console open. There must be no CSP violation for an inline `<style>` or inline `onerror` handler.

- Error-page preview: after signing in, open a custom-error-page preview from the service editor. It must load through `/api/error-pages/preview/<slug>` with the authenticated UI session; an unauthenticated request must remain rejected.
- Runtime/API: start a fresh environment and run the first compile/apply immediately after the panel is ready. The active revision must remain the revision returned by the operation; startup bootstrap must not replace it later.

- Login appearance: hard-refresh `/login` and `/login/2fa` with each saved login theme. The selected themed screen must be the first visible UI; the legacy card must never flash before it.
- Login appearance runtime: temporarily delay `/api/public/login-appearance` in browser DevTools and reload both login routes. The page may stay blank while the theme resolves, but it must not render the unthemed legacy layout.
- Basic Auth branding: open a protected site without its authentication cookie. The `/auth` page must show the TARINIO logo and TARINIO favicon before access to the upstream; Network must show successful local PNG responses from `/auth/assets/logo-wide.png`, `/auth/assets/logo-mark.png`, and `/auth/assets/favicon.png`.
- Revision apply recovery: apply a revision for a protected service whose upstream is unavailable or returns an error. The runtime must still confirm its local revision marker through the loopback-only readiness route; a valid Nginx configuration must not be rolled back because the protected upstream is unhealthy.

- Экран входа: обновить `/login` и `/login/2fa` с выбранным оформлением и убедиться, что до загрузки темы не показывается прежняя карточка входа.
- Маршрутизация UI: проверить прямые ссылки на известные разделы и их вложенные маршруты; запрос `/.ssh/id_ed25519` должен вернуть `404`, а не dashboard.
- Администрирование: после входа тестового пользователя проверить отображение времени в поле «Последний вход»; после смены пароля второе открытое окно этого пользователя должно потерять сессию.
- Basic Auth: после успешной проверки учётной записи сервиса обновить экран сайта и убедиться, что у этого пользователя появилось фактическое время «Последнего входа»; пароль, `Authorization` и cookie не должны появиться в «Запросах».

- Services: create a protected service with upstream `http://127.0.0.1:8080`, save it, then change the simple-editor fields to `http://privatebin:8080` and save again. Open Raw view without reloading and after reload: `WAF_SITE_UPSTREAM_HOST`, scheme and port must show `privatebin`, `http`, and `8080`; the old address must not remain.
- Services: change representative text, toggle, numeric and list settings in the simple editor, then switch to Raw before saving. Every corresponding `WAF_SITE_*` value must immediately reflect the currently visible simple form, not the previous service state.
- Services: in Raw change `WAF_SITE_UPSTREAM_HOST` from `127.0.0.1` to `privatebin`, retain port `8080`, switch back to Simple, save, and reload. The simple fields and Raw view must both retain `privatebin:8080`; after compile/apply the active service must proxy to that upstream.
- Services: add an allowlist entry, save the service, change an unrelated Easy Profile setting, save again, and reload. The allowlist must remain in both the simple editor and raw resource representation.
- API/runtime: after compile/apply, compare `/api/upstreams`, `/api/access-policies`, the revision artifacts `nginx/sites/<site>.conf` and `nginx/access/<site>.conf`, and `/etc/waf/current/` in runtime. They must contain the same upstream and access policy; an allowlisted client reaches the upstream while an unlisted client receives `403`.

2026-07-19

- Антибот и аутентификация: у селекторов шаблонов антибота и Basic Auth проверить, что кнопка «Просмотр» расположена по центру по высоте поля и не прилипает к нижней границе на широком и узком экране.
- Антибот и аутентификация → Аутентификация: убедиться, что варианты `v5`–`v9` отображаются в позициях 1–5, а бывшие `v1`–`v4` — в позициях 6–9; открыть предпросмотр каждого варианта и проверить соответствующий визуал.
- API/runtime: сохранённое значение `security_auth_basic.auth_basic_template` должно по-прежнему принимать `v1`–`v9`, а `/api/error-pages/preview/auth-vN` должен отображать новый порядок визуальных вариантов.

2026-07-17

- 2026-07-18 — локальные Basic Auth-превью: открыть все 15 файлов в `.work/sidebar-previews`. Убедиться, что в каждой группе Login, Healthcheck и Antibot пять страниц отличаются компоновкой, а не только фоном; на каждой есть только поля логина и пароля и одна кнопка входа. Проверить отсутствие верхних служебных подписей вида `10 · Basic Auth...`, SSO, ключей доступа и 2FA. Это локальные макеты и они не меняют runtime/API.

- Onboarding management host: complete initial setup with a new panel domain. Before the first compile/apply, verify the domain appears under Settings → Security → Management hosts; after applying, management `/api/*` must not receive a CRS 403.
- Requests security telemetry: trigger a 403 in an isolated environment. Requests must show a Security row with the blocking reason, while normal panel UI/API 2xx traffic remains absent.

- Management anti-bot/rate-limit: в чистом профиле откройте management host, затем быстро обновите страницу несколько раз. При отсутствии действующей anti-bot cookie первый ограниченный запрос должен перенаправить на challenge, а не показать fallback-страницу `429`. После прохождения challenge повторите нагрузку: проверенный клиент при реальном превышении квоты по-прежнему получает `429`.
- Runtime/API: после compile/apply убедитесь, что в `nginx/easy/<management-site>.conf` anti-bot redirect расположен до возврата `429`/`403` по cookie rate-limit. Исключения anti-bot и активная административная сессия должны сохранить свои существующие bypass-правила.

2026-07-16

- Healthcheck route: confirm `/healthcheck` returns the `healthcheck-app` shell and loads `/static/js/healthcheck.js`; after module load, verify API/runtime status checks are rendered.

2026-07-16

- Easy-site mTLS: submit valid `/etc/ssl/...` client CA/certificate/key references and confirm save → compile → apply remains successful.
- TOTP step-up/API: for a user with enabled 2FA, call `POST /api/auth/step-up/totp` with the current authenticator code, then export certificate material through the approved export flow. Confirm export is rejected before step-up and the assertion does not work after logging into a second session. Five bad codes must return a retry indication and the next attempt must remain locked.
- K8s secure storage profile: create the four documented external storage secret objects without replacing existing certificate keys, apply `opensearch-clickhouse` plus `guardrails`, and confirm OpenSearch accepts only authenticated HTTPS with its CA while ClickHouse rejects 8123/9000 and permits only TLS 8443 from runtime/control-plane pods.
- Easy-site mTLS: try a path containing a newline, semicolon, or an unapproved root such as `/tmp`; confirm the API rejects it and no revision is applied.
- Runtime/API: compile a site with ordinary host, wildcard alias, and safe WAF custom rule include; then try newline/semicolon injection in those fields and confirm no NGINX or ModSecurity directive is emitted.
- Services -> Access policy: save a valid trusted proxy CIDR (for example `10.0.0.0/8`), compile/apply, and confirm client IP restoration still works through that proxy.
- API/runtime: submit `trusted_proxy_cidrs` containing a newline, semicolon, hostname, or directive text. Confirm the API rejects it and no new `set_real_ip_from` directive is emitted; existing valid policies continue compiling.

2026-07-15

- README screenshots: открыть `README.md` и `README.en.md` на GitHub. Убедиться, что все шесть изображений интерфейса доступны, а ссылок на удалённый `ui/app/static/screen4.png` не осталось.

- Sidebar SVG cache: open DevTools Network, navigate between at least three panel sections, and confirm `/static/icons/svg/*.svg` is served from disk/memory cache after the first load with `Cache-Control: public, max-age=604800`.

- Management ban safety: complete the normal anti-bot flow on the WAF management host, including challenge redirects and favicon requests. Confirm no new rate-limit/security event or auto-ban row is created for the administrator IP; a real `429` on a protected service must still create its security event.
- Bans detail country: open a ban detail with country `RU` and confirm the country cell renders the local flag and `RU`, not an HTML `<img>` string.

- Request telemetry self-traffic: open the WAF panel, navigate through Dashboard, Services, Revisions, and Requests, then refresh `/api/requests` and dashboard stats. Management-host API/UI calls must not appear or increment counters; a request with the same path to a protected service must remain visible.

- Protected-site anti-bot return path: open `/auth/login?next=%2Faccount` on a non-management site in a fresh profile. Confirm the challenge appears first and, after verification, redirects exactly back to `/auth/login?next=%2Faccount`; the WAF management login behaviour must not alter this route.

- Management login anti-bot flow: in a fresh browser profile, open `/login`. Confirm it redirects to the branded challenge before any login HTML is cached; after verification, confirm the login page and `login.js` load normally. Expire or remove the anti-bot cookie and repeat: the browser must show the challenge again, never a blank page or a JavaScript MIME error.

- Sidebar SVGs: refresh the panel normally and confirm the uploaded local SVG icons appear for every navigation item plus notification, collapse, profile and logout. Verify expanded, collapsed and mobile sidebars retain their existing navigation behaviour and no icon is clipped or rendered black.
- Sidebar logout: confirm the exit SVG is optically centred in its circular footer button in both expanded and collapsed menu states.
- Management API error boundary: with custom error pages enabled, force a control-plane API upstream error (for example 504). `/api/` must retain its original status and JSON response rather than return branded HTML; WAF, anti-bot and rate-limit rejection paths must remain active.
- Dashboard attack totals: with WAF security events present, confirm the sum of `attacks_series` equals the daily attack value. Request-log attack entries may populate the series only when no WAF attack events exist.
- ACME onboarding: create a new TLS-enabled service whose hostname resolves to WAF before it is in the active host map. Confirm HTTP-01 challenge requests under `/.well-known/acme-challenge/` bypass the branded 421 page and ModSecurity, while unknown paths on that host still receive the normal 421 response.
- Testpage management runtime recovery: after the next apply, confirm `tarinio-testpage-runtime-mgmt` is healthy and healthcheck no longer lists a `runtime reload failed` error. API/runtime: check the generated `base.conf` direct-IP map and management login rate-limit include with `nginx -t`.
- Dashboard series widget: confirm the localized heading is “Запросы и атаки во времени (24ч)” and its width is one 20px grid step narrower (1040px), including after reloading an existing saved dashboard layout. Confirm hover data and widget resizing still work. Static assets: confirm `ui/app/static/icons/svg/` is available for new sidebar SVG files; no API/runtime change is expected.
- Error accordion containment: in variants 1 and 4, expand Errors and confirm it has the same fixed width as Checks. A long error must only increase row height; Refresh and Continue must be 44px high.
- Healthcheck width containment: in variants 1 and 4, open Errors containing an unbroken long token, URL or stack trace. The accordion, main frame and page must retain their original width; only the row content may gain height through wrapping.
- Healthcheck appearance refinement: open `/healthcheck?appearance=variant-1`, collapse all groups, and confirm every button retains its normal 44px height. Open a long error entry and confirm the text wraps inside the frame without horizontal page growth.
- Healthcheck appearance refinement: in variants 2 and 4, expand the Errors accordion with a long log entry. Confirm variant 2 has exactly one border per accordion and neither frame grows horizontally.
- Healthcheck appearance refinement: in variant 3, confirm the former “Внимание” and error cards are replaced by a wider right-side diagnostics console, while compatibility remains in the left workflow. Confirm the readiness title and description use the reduced type scale.
- Healthcheck appearance refinement: in variant 5, confirm all three accordions are collapsed after initial load and the live console retains recent successful-check messages rather than replacing the prior line.
- Healthcheck appearance refinement: in every variant, compare Refresh and Continue controls with the approved reference: outlined dark Refresh, filled blue Continue, equal compact height, hover and navigation behavior. API/runtime: refresh must still execute session, probe, container-issue and compatibility requests.

2026-07-14

- Внешний вид и healthcheck previews: в «Настройки → Основные» проверьте заголовок «Внешний вид» и подписи «№ 1 вариант» / «№ 2 вариант» во всех языках. Откройте десять файлов `healthcheck-01` … `healthcheck-10` из `.work/sidebar-previews`: каждый должен сохранять структуру рабочей healthcheck — проверки, ошибки, совместимость, статусы, счётчики и действие перехода в панель. После выбора пяти макетов требуется отдельный этап подключения их к реальной `/healthcheck` и preview-кнопкам.
- Уточнение preview healthcheck: `healthcheck-02` должен выглядеть как широкая контрольная комната с метриками и лентой, `healthcheck-03` — как центрированная карта готовности с этапами, `healthcheck-07` — как статусная доска с телеметрическим логом. `healthcheck-04` и `healthcheck-06` должны остаться визуально неизменными.
- Финальные уточнения preview: у «№ 2 вариант» фон Control Room должен быть приглушённым; у «№ 3 вариант» справа от карты готовности и этапов должен быть отдельный янтарный фрейм предупреждений/ошибок; у «№ 7 вариант» обе кнопки действий должны быть центрированы внутри нижней части фрейма. Файлы `healthcheck-04` и `healthcheck-06` не изменять.
- Вход и anti-bot: в селекторе внешнего вида должны идти ровно «1 вариант» … «5 вариант». При обычной перезагрузке `/login` и `/login/2fa` не должна на миг появляться legacy-форма или другая тема: до готовности допустим только пустой фон. С живой `waf_session` откройте `/login` на default management host: должен начаться challenge. API/runtime: после compile/apply проверьте, что `nginx/conf.d/base.conf` включает `ratelimits` и `easy` management-site для обоих маршрутов входа.
- Темы healthcheck: в «Настройки → Основные» выберите и сохраните каждый из 1–5 вариантов healthcheck. Откройте кнопку preview: URL должен быть `/healthcheck?appearance=variant-N`, а не статический HTML; в нём должны выполняться реальные проверки сессии, API, журналов контейнеров и совместимости. После сохранения откройте обычную `/healthcheck` без query: она должна использовать сохранённую тему. Проверьте кнопку «Обновить» и переход в панель в каждой теме.
- Геометрия healthcheck: в вариантах 1, 2 и 4 сверните и разверните все три аккордеона — скрытая панель не должна оставлять лишний фон, а фрейм не должен расти по высоте или ширине; длинная ошибка остаётся внутри фрейма. В варианте 1 счётчик «СЕССИЯ / ПРОВЕРОК» отсутствует. В варианте 2 блоки ошибок и совместимости не имеют внешних вложенных карточек. В варианте 3 compatibility находится слева, а справа показана консоль ошибок. В варианте 5 все три списка открыты при первой загрузке. В каждой теме primary-кнопка перехода использует индивидуальный синий цвет темы.
- Management anti-bot and passkey result: while a valid `waf_session` cookie is present, open `/login` and `/login/2fa`; both must still redirect to the configured challenge before the form renders. Complete it and confirm the return URL is preserved. On Security Card, enter a username with no passkey and press the passkey button: the page must stay on the login screen and show the localized «ключ доступа не найден» message, not a WAF 404 page. API/runtime: compile and apply the revision, then verify the generated easy configuration clears only the session bypass for `^/login(?:/2fa)?$`.
- Login variants: in «Настройки → Основные» ensure the selector order is Command Center, 1 вариант, Карточка защиты, Консоль инцидентов, 2 вариант. Select «1 вариант» and open `/login` and `/login/2fa`: its smaller heading, protected-access text and ticking local time must be visible without displacing the form; switch browser language to verify all added strings are localized in ru/en/de/sr/zh.
- Login appearance refinements: confirm both Incident Console themes keep all password, passkey, SSO, 2FA and back controls inside their rounded frames with compact spacing and no sharp background corners. Confirm Command Center (original) centres its form vertically without a blank gap, Security Card has no `⌾` emblem, and selecting a username without a registered passkey shows the localized “key not found” message instead of an HTTP 400 error page.
- Login templates fidelity: confirm the list contains exactly five themes and no Sidebar Rail. For every theme, compare each preview to `/login` and `/login/2fa` after saving: layout, backgrounds, logo, controls and actions must match. For both Incident Console variants, login and 2FA frames must have the same top position and fixed height; passkey, SSO, recovery-code and back controls must remain visible and usable.
- Login appearance: in «Настройки → Основные» select each of Command Center, Sidebar Rail and Incident Console, save, then open both previews. Reload `/login` and `/login/2fa`; confirm the saved theme and `logo800x300.png` are used, password/passkey/SSO/2FA controls remain available, and an anonymous login page emits no `/api/auth/me` 401 request. API/runtime: `GET /api/public/login-appearance` exposes only the selected identifier; `PUT /api/settings/runtime` remains protected by `settings.general.write`.
- Login logo: open `/login` at desktop and mobile widths; confirm the text «Berkut Solutions - TARINIO» is absent, the 280px logo remains sharp and does not overflow the card, and all sign-in methods work as before. API/runtime contracts are unchanged.
- Sidebar footer alignment: verify profile and logout are centred in their two right-side columns, including after collapsing the sidebar; the logout glyph must sit optically centred after its 2px right offset. API/runtime contracts are unchanged.
- Sidebar account controls: confirm profile and logout have the same size, outline, background, colour and hover state as the notification button; verify both actions still work.

- Sidebar compact layout: confirm Runtime healthy and `rev-…` remain on one line without a left indent; header notification/collapse buttons are smaller; MODEL is on the second protection row. Confirm CRS text equals `active_version` from `/api/owasp-crs/status` and ENFORCE is absent.
- Sidebar footer alignment: confirm Runtime healthy is the first line, the active revision is directly below it, neither can overlap the account controls, and profile/logout buttons are 29px circular controls.
- Sidebar compact controls: confirm the expanded sidebar is narrower, header and footer controls use the same outline style, and collapsed profile/logout icons are centered beneath a separate centered status/revision block.
- Sidebar width/cache: after a normal refresh, confirm the expanded width is 250px, footer controls start immediately after the status block, and collapsed “Runtime healthy” wraps by words without crossing the sidebar boundary.
- Sidebar final compact sizing: confirm the expanded width is 245px, MODEL is on the first protection row, and profile/logout have the exact same circular style and hover state as notification/collapse.

- Sidebar compact status: confirm L4, L7 and MODEL dots switch green/red according to `/api/anti-ddos/settings`; confirm the displayed revision matches `revision_apply.active_revision_id` from `/api/reports/revisions`. Collapse the menu: the protection strip must disappear and the expand button must be centered directly below the logo.
- Sidebar footer: at full width, confirm runtime status and revision sit on the left of the same row as profile and logout controls; all footer controls remain usable.

- Management anti-bot regression: in a clean browser profile, open `/login` and confirm a `302` redirect to `/challenge`; after verification, confirm the login page loads its logo, CSS and JavaScript. Direct `GET /static/logo700x250.png` must return `200`, never a challenge redirect.
- Sidebar status: confirm the footer shows a green or red dot next to the localized runtime status and the latest revision ID. Disconnecting runtime in a test environment must switch only the dot/status to unavailable and must not affect navigation.
- Sidebar profile and labels: confirm ENFORCE / CRS 4 / L4 ON appear above Monitoring, Incidents is visibly present but non-navigable until its page exists, Certificates replaces “TLS / Certificates”, and header notification/collapse controls share one row.

- Login: open `/login` and confirm the traffic-analysis subtitle is absent while all authentication controls retain their original placement and behavior.
- Sidebar refinement: verify the notification bell is above the collapse button, group separators appear before Management, Protection and System, and no dark/shadow background strip overlaps the left edge of page content. Open Notifications and confirm its dropdown is still fully visible.

- Sidebar navigation: verify the TARINIO logo panel uses `/static/logo512.png`, its logo frame is visibly lighter than the sidebar, and the chevron is centered in its circle. Click the chevron twice and confirm the menu collapses to icons and then restores without layout overlap.
- Sidebar groups: confirm the visible groups are Monitoring, Protection, Management and System. In System, verify the routes open as Administration → `/administration`, Journal → `/events`, Audit → `/activity`, Settings → `/settings`; unavailable sections must stay hidden according to the current RBAC permissions.
- Sidebar controls: open and close the notification bell, then use profile and logout controls. Confirm their existing behavior is unchanged after the visual migration. No API or runtime contract changes are expected for this UI-only update.

- Management anti-bot: open `https://localhost/login` in a clean browser profile. When anti-bot is enabled, confirm the first response redirects to `/challenge` and returns to the login page after verification; `/challenge` itself must not loop. Confirm login still has its rate-limit protection.
- Management credential API: without the anti-bot cookie, `POST /api/auth/login` must return `403`. After completing `/challenge/verify`, the same endpoint must reach the authentication service (an invalid payload may return `401` or `400`, but not a challenge bypass). Confirm `/login` and `/login/2fa` inherit the site-wide connection and request limits.
- Anti-DDoS protection E2E: run `go test ./ui/tests -run TestE2EL4L7AdaptiveProtection -count=1 -v` with `WAF_E2E_L4_L7_PROTECTION=1` and testpage URLs. Confirm real `429`, adaptive `drop`, the matching L4 DROP rule, and a timed-out new attacker connection.
- Dashboard: open "Attacks over time (24h)". Confirm the purple usage-style area chart shows the 24-hour total, peak and average attacks; the dashed average guide aligns with the Y scale and X axis keeps hourly labels.
- Dashboard hover: move across the full chart width. Confirm the cursor snaps to each hourly bucket and the tooltip shows its exact date, hour and attack count.
- API/runtime: request `/api/dashboard/stats`. Confirm the UI uses `attacks_series` unchanged; no token or billing data is requested or displayed.
- Dashboard visual refinement: confirm there are no total, peak or average counters. The graph must be a single thin blue attack line over an orange baseline, with sparse dashed grid and labels along the bottom edge.
- Dashboard telemetry seeder: run `scripts/seed-dashboard-telemetry.ps1`, wait up to 10 seconds and refresh Dashboard. Confirm request, attack and blocked-attack counters are non-zero; countries include RU, DE, US, JP and BR, and each hourly attack bucket has hover data.
- Dashboard chart and country flags: confirm the blue Requests and orange Attacks lines use their own hourly values and visibly peak at different hours. Confirm flags render without a `flagcdn.com` request; a localhost/unknown-country record intentionally has no country flag.
- Dashboard axis and demo distribution: confirm there is no extra orange baseline; the left Y axis shows maximum requests at the top and average requests at mid-height. Run the telemetry seeder twice and confirm the hourly peaks vary between runs while every generated event remains in its intended hourly bucket.
- Local country flags: confirm all country codes available in `static/flags/16x12` appear as PNG flags in Dashboard and Bans with no external network request and no country-code emoji fallback.

2026-07-13

- Services → Custom error pages: открыть preview для 403, 421, 451 и 500 и переключить язык браузера между ru/en/de/sr/zh; убедиться, что всегда показывается новый WAF-дизайн и локализованный текст, без legacy fallback-страницы.
- Runtime/API: для включённой custom error page проверить WAF-страницу; отключить конкретный код и убедиться, что ответ для этого кода передаётся от upstream без WAF HTML.

- Management login/session: истечь сессии на `/services` или перезапустить runtime/control-plane при открытой панели. Ожидается один переход на `/login?reason=session_missing` либо `session_check_failed`, без запросов к `/challenge` и без burst-редиректов.
- Management login assets: временно перезапустить UI/runtime при открытой `/login` и `/login/2fa`. Ошибки загрузки CSS/JS не должны перенаправлять браузер в `/challenge`; после восстановления контейнера страница должна штатно загрузиться или быть доступной для ручного refresh.
- API/runtime: подтвердить в DevTools, что `GET /api/auth/me` без сессии отвечает `401`, а не порождает клиентский запрос `/challenge`; при включённом или выключенном anti-bot management-host остаётся исключённым из challenge flow.
- Services → `localhost`: изменить безопасную настройку (например, лимит или allow/deny list), сохранить и применить ревизию. Запрос `/api/access-policies/upsert` должен передавать `site_id: "localhost"`, без `control-plane-access`; сохранение и reload должны завершиться успешно.

2026-07-12

- Anti-bot preview and challenge flow: verify the JavaScript templates redirect through the configured verify URI without a reload loop; covered by the real runtime E2E suite.
- RU localization: open Settings → Management hosts and the TLS certificate rebinding dialog; confirm Russian text is fully localized while the selected site and certificate names are interpolated correctly.
- API/runtime: save Management hosts, compile and apply a revision; confirm the localized status text does not change the save/apply workflow or server-side access controls.

- Проверить `GET/PUT /api/settings/management-hosts`: список DNS/IP хостов, ошибку version conflict и невозможность сохранить пустой список.
- После изменения списка выполнить compile/apply; management host должен пропускать API-операции, а обычный сайт на том же `ui:80` — оставаться под CRS.
- В Settings → Management hosts проверить, что новый host нельзя сохранить до создания enabled site с тем же `primary_host`; параллельное сохранение устаревшей версии должно показать conflict, а не тихо перезаписать данные.
- Runtime/API: после первого apply management-site с TLS проверить login, create/update/disable/delete site, update user и policy через public management host; запросы должны идти по HTTPS listener WAF без CRS 403, а upstream management site должен оставаться control-plane, не тестовым echo upstream.

2026-07-06
- Services (/services): после логина открыть страницу Services и убедиться, что модуль /static/js/pages/sites.js загружается без ServicesStableFacadeLoadError, без redirect на /challenge/stage1/verify и без 302/403 на статические JS-модули.
сделано

- Dashboard session ping: после логина убедиться, что POST /api/app/ping уходит без request body, получает 200 и не вызывает challenge/403.
сделано

- Manual API/runtime check: в runtime/edge проверить, что запросы к /static/js/pages/sites.js и связанным JS-модулям панели не попадают под antibot challenge для авторизованной management-сессии.
сделано

- Management antibot bypass: проверить, что GET/HEAD на /static/*, /services, /dashboard, /auth и /auth/verify для management-site не уводятся в challenge даже при включённом antibot challenge у easy-профиля.
сделано

- Runtime unauth probe: после пересборки/перезапуска сделать прямой probe на https://localhost/static/js/pages/sites.js?v=20260628-16 и подтвердить, что больше нет 302 Location: /challenge/stage1/verify; ожидается 200 и JS content-type/body.
сделано

- WAF self-management safeguard: для management-host проверить, что /login, /login/2fa, /dashboard, /services, /api/sites/* и /api/access-policies/* не режутся собственным ModSecurity/CRS и не попадают в self-block path даже при включённом easy security mode=block.
- ожидает live/prod verify

2026-07-12

- Services → Site editor → Front service: set security mode to `monitor`, enable ModSecurity with a harmless test rule, save and reopen the site. Confirm the ModSecurity toggle and custom configuration remain enabled in the editor; the mode description must still state that requests are not blocked.
Ожидает ручную UI/API/runtime-проверку

- Runtime/API: after compile/apply of the same `monitor` profile, send a request matching the test ModSecurity rule. It must reach the upstream (`200`), while the generated site configuration references the per-site ModSecurity rules file with `SecRuleEngine DetectionOnly`. Switch to `block` and confirm the same request is then blocked.
Ожидает e2e-проверку через WAF runtime

2026-07-11

- Dashboard live logs: open Dashboard, wait for automatic refresh, switch the browser tab away for at least one polling interval, then return and confirm the requests/security widgets resume with fresh data but no burst of duplicated updates.
Ожидает ручную UI/API/runtime-проверку

- Anti-DDoS events: open the Anti-DDoS events view, leave it running, hide the tab or minimize the browser, then return and confirm polling resumes from a single active timer without duplicate network refreshes or frozen status.
Ожидает ручную UI/API/runtime-проверку

- Manual API/runtime check: while Dashboard logs or Anti-DDoS events are open, confirm hidden-tab periods stop repeated `/api/requests` or related polling calls, and returning to the tab triggers only one immediate refresh before the normal interval resumes.
Ожидает ручную UI/API/runtime-проверку
- Management UI: with ModSecurity enabled and an intentionally blocking custom rule for `/api/app/ping`, confirm that an authenticated dashboard ping still returns `200` after compile/apply.
Ожидает ручную UI/API/runtime-проверку

- Management login: let an authenticated management session expire on `/dashboard` or another app page, then confirm the redirect goes through `/challenge` before `/login` so the login screen loads with CSS instead of a broken unstyled shell.
Ожидает ручную UI/API/runtime-проверку

- Management login: leave `/login` open until the antibot challenge is stale, then try password, passkey, or SSO sign-in and confirm the page re-enters challenge flow instead of failing with a raw `403`.
Ожидает ручную UI/API/runtime-проверку

- Management login 2FA: leave `/login/2fa` open until the challenge is stale, then submit the code or passkey and confirm the page refreshes challenge access before retrying instead of showing a late `403` or losing styles.
Ожидает ручную UI/API/runtime-проверку

- Duplicate-tab logout/session expiry: with two management tabs open, sign out or let the session expire from one tab and confirm the other tab recovers to a styled login/challenge flow rather than a CSS-less login page or inconsistent `403`.
Ожидает ручную UI/API/runtime-проверку

- Management UI: confirm Services and its `/static/js/pages/sites.js` module load after login; verify an ordinary service does not receive the management ModSecurity bypass.
Ожидает ручную UI/API/runtime-проверку

- Services → Site editor → ModSecurity: enable the module with a custom rule, apply it, and verify that the test request is blocked; disable it, apply, and verify that the same request reaches the upstream.
Ожидает ручную UI/API/runtime-проверку

- Services → Site editor → ModSecurity exclusions: re-enable ModSecurity, add an exact GET exclusion for one URI and the custom rule ID, then confirm that URI is allowed while the same payload on a different URI remains blocked.
Ожидает ручную UI/API/runtime-проверку

2026-07-07
- Services → Site editor: with security mode set to `transparent` and `monitor`, enable and disable antibot, ModSecurity, rate-limit, blacklist, auth, and API protection settings; save and reopen the site to confirm every selected value is retained.
Автопокрытие для API/runtime добавлено в `TestE2ESecurityModesReality`; ручная browser/UI-проверка save/reopen в этом прогоне не выполнялась и при необходимости остаётся отдельным шагом.

- Runtime/API: after applying `transparent` or `monitor`, confirm generated WAF, access, and rate-limit policies are disabled and requests do not receive antibot/auth redirects or blocking responses; switching to `block` must activate the saved settings.
Сделано автотестом `TestE2ESecurityModesReality`: сценарий создаёт отдельный easy-site, делает реальный compile/apply и проверяет, что `transparent`/`monitor` не блокируют `/admin`, а `block` снова включает защиту.

- Services → Site editor → ModSecurity: открыть редактор exclusion rules и проверить, что подписи path/path pattern, mode, methods, rule IDs, targets и comment рендерятся локализованными строками, а не raw i18n keys.
ожидает ручную UI-проверку

- Services → Site editor → ModSecurity: валидацией проверить кейсы без path/path pattern, без rule IDs, с path без ведущего /, с методом * вместе с другими методами и с превышением лимита правил; для ru/en/de/sr/zh должны показываться локализованные сообщения без английских артефактов в русской локали.
ожидает ручную UI/API-проверку

- UI contract smoke: после сборки открыть onboarding, dashboard, services, requests, settings, anti-ddos и revisions и подтвердить, что страницы и sidebar-роуты открываются без missing-module/missing-marker regressions; отдельно проверить наличие dashboard widgets/services frames и route menu items Requests/Anti-DDoS.
ожидает ручную UI-проверку

- Management-site safeguard verify: на management host вручную проверить, что при включённом ModSecurity login/dashboard/services/api/static маршруты панели не режутся self-WAF, а для обычного сайта такие management-route исключения автоматически не появляются.
ожидает ручную UI/API/runtime-проверку
# 2026-07-13

- Управление страницами ошибок: переключить интерфейс на `ru`, `en`, `de`, `sr` и `zh`, открыть редактор сайта и проверить названия всех групп, HTTP 400–431, 444, 451, 500–511, расширенных кодов и Geo Block. В каждой локали должны отображаться переведённые подписи, а кнопка preview должна открываться для выбранной страницы.

  Ожидает ручную UI-проверку.

- AIO installer: на чистом или обновляемом default-профиле собрать UI-образ через installer. Шаг `go test ./ui/tests` внутри Dockerfile должен завершиться успешно и не сообщать об отсутствующем `waf/control-plane/apiroutes`.

- Management-host + antibot: откройте `https://prewaf.hantico.ru` в чистом профиле браузера. Ожидается branded JavaScript challenge, затем `/login`; после успешной проверки интерфейс, API и статика должны загружаться без повторного challenge.
- Management-host protections: при включённых ModSecurity и лимитах проверьте `/login`, `/dashboard`, `/services` и административные API. Панель не должна self-block, а неадминистративный трафик того же хоста должен по-прежнему проходить защитные модули.
- Custom 504: временно воспроизведите timeout у control-plane API и у custom rate-limit API. В обоих случаях должна отобразиться стилизованная страница HTTP 504, а не стандартный ответ nginx.
- Telemetry: после compile/apply сделайте несколько запросов к management-host и проверьте Dashboard и `/api/requests`: число запросов за 24 часа и индексы архива должны обновиться без долгого ожидания. В Administration запустите сборщики событий и здоровья индексов — каждый должен завершиться и предоставить архив.

- E2E Geo Block: при blacklist/whitelist-миссе в основном behavioral-сценарии ожидается HTTP 403 с branded Geo Block. Страница 451 остаётся отдельной юридической страницей.
- Legacy error-page route labels: open previews for 400, 403, 404, 429, 502, and 503. Confirm Russian routes end with `Сервер` and English routes end with `Origin`, matching the extended pages.
- Extended error-page route labels and colours: in Russian, confirm every route reads `Клиент → WAF → Сервер`; in English, confirm `Client → WAF → Origin`. Confirm the service glyph is amber for 520, 522, and 524, purple for 525, and cyan for 526, matching each node's border.
- Extended error-page descriptions and terminal node: open every extended preview in Russian. Confirm the subtitle explains that code's concrete fault rather than saying that WAF identified a response. Confirm the final node uses the neutral 403-style service glyph whenever the origin is reachable, responds, or is not contacted; use the crossed glyph only for a genuinely unavailable origin. For 524, confirm the clock remains vertically centred inside the message icon.
- Extended error-page request-path visuals: open every extended-page preview. Confirm the red, warning, TLS, certificate, cancelled, and muted styles mark the actual failed hop; a client-side error must not retain a green client-to-WAF arrow, and upstream failures must not mark the client as faulty. Confirm the cross in the final host square is vertically centred.
- Extended error-page icon refinements: confirm the 524 clock is one pixel higher than before and the 526 warning circle is shifted up and left.
- Extended error-page emblem refinement: open previews 524–526. Confirm only the 524 clock moved upward; confirm 525 has a red cross at the centre of the broken chain; confirm the 526 warning circle and exclamation mark are smaller.
- Extended error-page emblems: open previews 495, 522, 524, 525, and 526. Confirm the 495 cross sits lower within its certificate, and the other four symbols visually distinguish connection timeout, response timeout, failed TLS handshake, and invalid certificate.
- Extended error-page Russian i18n: set the browser locale to `ru` and open every extended error-page preview. Confirm that the route nodes, states, action titles, and action descriptions contain no English fallback.
- Extended error-page content: open previews for 451, 494–497, 499, 506, and 520–526 in each browser locale. Confirm the request route identifies the actual endpoint and state for that code, and all three recovery actions remain code-specific rather than falling back to the shared generic text.
- API/runtime: call `/api/error-pages/preview/494` and `/api/error-pages/preview/525`, then compile/apply a site. Confirm the generated files retain the same code-specific route and recovery text after the page-localization script runs.

- Error-page visual review: open previews for 451, 494, 495, 496, 497, 499, 506, and 520–526. Confirm each page has the full WAF card (emblem, request path, three recovery steps, and metadata), plus a distinct accent colour and matching icon.
- Error-page i18n: repeat the preview check with `ru`, `en`, `de`, `sr`, and `zh` browser locales. Confirm the status title, description, and request-state label are localized, while the shared 502 card labels and request path remain visually identical.
- Error-page content review: for every extended page, confirm the request-path labels and both arrow states identify the actual failure point. Confirm all three recovery actions are specific to that status rather than copied from 502.
- Error-page labels: switch the panel locale between `ru`, `en`, `de`, `sr`, and `zh`; confirm pages 451, 494–497, 499, 506, and 520–526 use the corresponding `sites.easy.errorpages.extended.*` translations in the editor list.
- Error-page preview i18n: open preview 494 in each supported browser locale. Confirm request-path nodes, arrow states, recovery actions, empty Request ID, and Code label contain no fallback English; the third request-path node must remain Host.
- API/runtime: request `/api/error-pages/preview/451`, `/494`, `/499`, `/506`, `/520`, and `/526`, then compile/apply a site revision. Confirm the same branded HTML is saved under `errors/<site>/` and query parameters `rid`, `ip`, and `ts` populate the metadata.

- Services → Site editor → Error pages: проверить пять категорий, наличие Geo Block (HTTP 403), 451 Unavailable For Legal Reasons и новых 494–497, 499, 506, 520–526; у vendor-specific кодов должен быть визуальный маркер, у 499 — диагностическое пояснение.
- Services → Site editor → Error pages: выключить Geo Block, сохранить и применить ревизию; геоправило должно перейти на обычный 403, а включённый Geo Block — вернуть 403 с телом geo_block. Отдельно убедиться, что 451 не используется геоправилом.
- API/runtime: открыть `/api/error-pages/preview/geo_block`, `/451`, `/494`, `/499`, `/506`, `/520` и `/526`, затем проверить compile/apply: шаблоны присутствуют в `errors/<site>/`, а неизвестный slug preview возвращает 404.
- Services → Site editor → Error pages: открыть preview 495 и Geo Block. У 495 должны быть собственные HTTP 495, название и текст, а `geo_block` должен отвечать стилизованным HTML, а не HTTP 400 браузера.

- Services → Site editor: rename a live service from an IP-based ID to a DNS-based ID, save once, then reopen and save again. Confirm the new service, TLS binding, certificate material, profile and policies remain attached; no second certificate issuance is requested.
- Runtime/API: apply the renamed service revision and confirm the candidate contains the DNS service only, has no files or upstream references for the retired IP, and an HTTPS request with an unknown Host receives a branded WAF 421 page.
- Dashboard: open the public management host, generate normal UI/API requests and confirm they appear in request telemetry while internal localhost/control-plane calls remain excluded.

2026-07-13

- Dashboard: open “Attacks over time (24h)”. Confirm the amber line and area match the WAF dark palette, each point is a real hourly attack count, and hover shows date, hour, and attack quantity with a vertical cursor.
- API/runtime: call `/api/dashboard/stats` after known blocked traffic. Confirm `attacks_series` has 24 hourly buckets and the relevant bucket increments without replacing request telemetry.

- Version fallback: open the WAF shell before and after locale loading and confirm the footer version is `1.5.2`; it must not briefly show the legacy `v1.3.5` value.

- Settings → Security: toggle “Block direct access to the WAF IP”, save it, reload the page, and confirm the saved state is retained. Confirm the update triggers compile/apply.
- Runtime/API: with the option off, request the WAF public IP over HTTP and HTTPS and confirm the branded 421 page has no `nginx/` signature. Enable it and confirm direct IP requests terminate with 444 and do not reach an upstream.
- Requests: generate direct-IP and unknown-host requests. Confirm Service shows the actual WAF IP or unknown Host, never `.global`, and Security reason shows the localized direct-IP/unknown-host category.

2026-07-15

- Healthcheck: открыть `/healthcheck?appearance=variant-1` и `/healthcheck?appearance=variant-4`, развернуть «Ошибки» с длинным сообщением. Текст должен переноситься внутри аккордеона, а ширина фрейма и страницы не должна меняться. Проверить компактную высоту кнопок «Обновить» и «Перейти в панель».
- Healthcheck: открыть варианты 2, 3 и 5. В варианте 2 не должно быть двойной рамки аккордеонов; в варианте 3 консоль ошибок занимает правую широкую область; в варианте 5 все аккордеоны свёрнуты по умолчанию, а консоль показывает результаты успешных проверок и ошибки.
- Dashboard: открыть виджет «Запросы и атаки во времени (24ч)». Убедиться, что его ширина уменьшена на один шаг сетки, а при наведении видны независимые почасовые значения запросов и атак.
- Sidebar: проверить локальные SVG-значки всех разделов, одинаковое выравнивание управляющих иконок и центрирование значка в кнопке «Выйти». Смена маршрута и RBAC не должны измениться.
- API/runtime: при неизвестном Host и выключенной политике прямого IP проверить брендированную страницу `421` без подписи nginx; запрос `/.well-known/acme-challenge/<token>` на неизвестном Host не должен подменяться `421`. При включённой политике прямой IP должен завершаться `444`.
- API/runtime: включить ModSecurity для management host, сохранить и применить ревизию. Административные API, login, dashboard, services и статические ресурсы должны оставаться доступны; обычный сервис не должен получать это исключение.
- Management login: в чистом профиле браузера открыть `/login`. Сначала должна показаться anti-bot проверка; после её прохождения должны без MIME-ошибок загрузиться `login.js`, стили, SVG и favicon. Проверить, что повторный заход не использует устаревший закешированный login-документ.
- Requests: открыть панель после обычного входа и убедиться, что `/static/*`, favicon, SVG, CSS, JavaScript и прочие asset-запросы не попадают в список или счётчики запросов, а API и пользовательские маршруты продолжают отображаться.
2026-07-17

- Fresh onboarding session migration: after an earlier HTTPS deployment, keep browser cookies, run `docker compose down -v`, and open HTTP onboarding again. Create the first administrator and complete self-signed TLS setup. The flow must not show `authentication required`; after apply, HTTPS login and `/api/auth/me` must work.

- Fresh deployment UI permissions: run `docker compose down -v`, build and start the default profile, then open `/login` and `/onboarding/user-creation` through the runtime public IP. Both pages must load normally; neither may return raw `403 Forbidden` or expose an nginx version banner.

- Management onboarding + Anti-DDoS: enable global L7 rate limiting, configure the panel under a custom management-host ID, then refresh `/login`, `/onboarding/user-creation`, and the app shell. The management flow and its API must not receive a global 429. A non-management service must retain its configured rate-limit behavior.
- Management shell fallback: with browser developer tools simulating a failed static asset or a background `/api/auth/me` response, confirm the loaded onboarding/app shell is not replaced by the legacy 429 document. A real HTTP 429 returned by a protected non-management service must still show the runtime-branded error page.

- Management host + anti-bot: while the browser has no valid anti-bot cookie, trigger a rate-limit response and refresh `/login`, `/login/2fa` and `/healthcheck`. Each refresh must enter the challenge flow instead of rendering the internal `429` fallback page; after successful verification, a genuinely over-limit browser may still receive `429`.

- Settings → Security: verify the new “Certificate private-material export” frame in `ru`, `en`, `de`, `sr`, and `zh`. With approval required enabled, request/approve/export with two users and confirm a replay is rejected. Disable it, save the common Security form, then confirm an authenticated user with fresh TOTP can export without an approval; re-enable and confirm the requirement returns after reload.
- Settings → Security: change “Block direct access to the WAF IP”, use the single bottom “Save” button, reload, and confirm the state persists. The direct-IP frame must not have a second Save button.

- OWASP CRS: in each of `ru`, `en`, `de`, `sr`, and `zh`, open the OWASP CRS page and run “Check latest”. A successful check must show the current release without asking for `WAF_CRS_TRUSTED_SHA256`.
- OWASP CRS API/runtime: simulate an unavailable official release endpoint or a release asset without `digest`, call `POST /api/owasp-crs/check-updates`, and confirm the response remains HTTP 502 with a stable `code`; the page must show the localized explanation and must not expose the runtime’s English technical detail. Confirm the current active CRS path/version remains unchanged.

- Login/onboarding: after upgrading an existing installation with an admin account, clear browser cookies, pass anti-bot if enabled, and open `/login`. The page must remain the login flow and must never redirect to `/onboarding/user-creation`. A brand-new deployment with no users must still enter onboarding over HTTP.

- Settings API/runtime: in the Enterprise Compose profile, confirm runtime logging shows `https://opensearch:9200` and/or `https://clickhouse:8443` with the mounted CA path after upgrade. A missing CA, TLS file, or storage-user configuration must fail the storage workload; an existing non-Enterprise profile must retain its previous behavior.

- Certificate export: request approval for a selected certificate set as user A, approve it as user B, then export as A with `approval_id`. Verify one download succeeds; replay, self-approval, a changed certificate set, or another requester is denied. Existing export URL/body stays the same aside from the new `approval_id` argument.

- Virtual patches: create a blocking patch, wait for the success response, and verify a matching request is blocked immediately. Delete it and verify the request is allowed again; simulate failed apply and confirm UI receives an error rather than a false success.

- OIDC: initiate sign-in with `next=/dashboard?tab=security` and confirm callback returns there. Repeat with `//external.example`, `https://external.example`, `/\\external.example`, and `/%2f%2fexternal.example`; each must land on `/healthcheck` rather than leave the management origin.

- Storage runtime: configure an HTTPS OpenSearch or ClickHouse endpoint with its CA file, restart runtime, and verify indexing/querying succeeds. Confirm an invalid CA file fails closed; an existing HTTP or public-CA configuration remains compatible when no CA file is supplied.

- HA deployment: before applying `ha-control-plane`, create the documented `tarinio-lab-redis-tls` secret with a CA-signed `redis` DNS SAN certificate. Confirm both replicas become ready and that a connection without TLS or the `waf-coordination` ACL cannot issue Redis commands.
- Kubernetes guardrails: after applying the profile, confirm UI/public WAF traffic remains reachable, while a disposable unrelated namespace pod cannot connect to control-plane, PostgreSQL, OpenSearch, Redis, or runtime health ports.

- Dashboard: sign in as a non-administrative dashboard reader and confirm process-detail dialogs show no host process rows; an `administration.read` user still sees the same process metrics without command-line arguments.
- Dashboard: dismiss an upstream error as administrator A, then sign in as administrator B and confirm the error is still visible to B. Confirm the existing DELETE URL and payload still work.
- Settings API/runtime: a user with only `settings.general.write` must receive 403 when saving Storage or Logging; a user with `settings.storage.write` may save. For a custom ClickHouse deployment, set `WAF_CLICKHOUSE_ALLOWED_ENDPOINTS` to its exact base URL before upgrade and verify logging continues after restart.
2026-07-18

- Basic Auth preview selection: open each of the ten files in `.work/sidebar-previews` at desktop and narrow widths. Confirm its username/password fields remain usable, its primary button is visible, and the style is visually distinct from the other nine variants.
- Basic Auth preview scope: these are standalone design references only; verify no runtime endpoint, authentication behavior, or server-side authorization was changed.
2026-07-18

- Basic Auth previews: review `11-login-command-center.html` through `15-login-minimal-ops.html` against the visual language of the login screen; confirm the management-security hierarchy and field readability are retained.
- Basic Auth previews: review `16-antibot-progress.html` through `20-antibot-timeline.html` against the anti-bot screen; confirm the browser-verification state leads clearly into the Basic Auth form and remains usable on a narrow viewport.
2026-07-18

- Антибот и аутентификация → Аутентификация: выбрать каждый из девяти шаблонов Basic Auth и открыть «Просмотр». Убедиться, что вариант, поля учётных данных и оба локальных логотипа отображаются корректно, в том числе на узком экране.
- Антибот и аутентификация → Аутентификация: сохранить выбранный шаблон, обновить страницу редактора и подтвердить сохранение выбора. Выполнить compile/apply и открыть `/auth/login` на защищаемом сайте: страница должна соответствовать выбранному варианту.
- API/runtime: `security_auth_basic.auth_basic_template` принимает только `v1`–`v9`; после успешного Basic Auth POST на `/auth/verify/basic` должна выдаваться сессионная cookie и исходный маршрут должен открываться без повторного запроса учётных данных.
2026-07-18

- Basic Auth previews: open variants `v1` through `v9` with browser languages `en`, `ru`, `de`, `sr`, and `zh`. Confirm all visible page text, validation errors, and the `Secured by Tarinio` footer use the selected language; Tarinio must link to its GitHub project.
- Basic Auth layout: verify `v1`–`v5` show only the wide logo; check darker Nordic (`v3`) and Coral (`v4`) colors, the taller Minimal Ops (`v5`) header, centered Command Center (`v6`) login card, and the reduced Classic Split (`v9`) logo with a centered, frameless form.
- Antibot and authentication tab: verify both template-preview buttons are vertically centered beside their selectors at desktop and narrow widths.
2026-07-18

- Антибот и аутентификация: убедиться, что обе кнопки «Просмотр» совпадают по высоте с соответствующим select, а варианты Basic Auth названы нейтрально — «1 вариант» … «9 вариант» в русской локали.
- Basic Auth previews: проверить светло-синий `v4`, восстановленный текст в `v6`, увеличенный логотип без рамки в `v7`, правое выравнивание только текста соединения в `v8` и немного увеличенный логотип в `v9`.

2026-07-18

- Dashboard: generate more than 1,000 protected-service requests during 24 hours across at least two hosts. Confirm the request card, graph buckets, top sites/URLs, unique IPs, error list and blocked-security widgets all reflect the same full time window rather than only the latest page of requests.
- Runtime/API: call the authenticated `/requests/dashboard-summary?since=<RFC3339>` endpoint and compare its totals with `/api/dashboard/stats`; panel/login/static traffic must be absent while blocked protected-service requests remain present.
- Healthcheck: generate a ModSecurity 403 for a protected service. It must be visible in Requests and attack/block widgets, but must not appear as a runtime container error in the healthcheck issues list.
- Dashboard: click a top attacker IP that is older than the current Requests feed page. Its detail modal must still open and show the observed country rather than `Unknown`.
- E2E seed: run `scripts/seed-dashboard-telemetry.ps1`, set `WAF_E2E_DASHBOARD_SEEDED=1`, and run the dashboard E2E test. Confirm it detects two seeded sites, country metadata, and exactly 24 request/blocked buckets.
- Dashboard chart: after seeding 54 requests per hour, open "Requests and attacks over time (24h)". The top Y-axis label must be 54 and the middle label must be 27; it must not display the arithmetic mean (for example, 2.25) as the scale midpoint.

2026-07-18

- Requests: open the block-reason select in every supported locale. Verify it contains only observed security reasons, filters the table immediately, and the selected reason remains visible in the detail row as the localized reason.
- Requests: open a failed Basic Auth and a failed anti-bot verification. Each must be marked as a security row with reason `auth` or `antibot`; a successful Basic Auth verification must be a normal request without a block reason.
- Requests: open details and raw JSON for old and new rows. The UI must not show the quick security summary, legacy labels/fields, `legacy_row_type_support`, or the legacy-compatibility detail field.

2026-07-19

- Services → Anti-bot and authentication → Basic Auth: save a password of a known length and reload the editor. The field must display exactly that many mask characters without treating them as a password. Click the eye once to retrieve and show the saved value, then click again to mask it; save an unrelated setting and verify the password still works. API/runtime: the normal profile response must remain masked; `POST /api/easy-site-profiles/<site>/auth-password/reveal?username=<name>` must require site-write access and create an audit event.

- Custom error pages: trigger a branded 4xx or 5xx response through the runtime. Request ID, client IP and time must be populated rather than `н/д`; inspect the response header and confirm `Server-Timing` contains `rid`, `ip` and `ts` values.
## 2026-07-21

- [ ] Management UI: открыть `/login` и `/dashboard`, убедиться, что интерфейс и вход работают после включения CSP и защитных HTTP-заголовков.
- [ ] API/runtime: проверить ответ management UI на наличие `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN`, `Referrer-Policy: same-origin` и CSP без разрешённого внешнего происхождения.

2026-07-23

- [x] Services Basic Auth: в disposable stack добавить и удалить второго пользователя, сохранить v6 и TTL=5, подтвердить API readback, после reload проверить маску точной длины, reveal/hide исходного пароля и audit `easysiteprofile.auth_password.reveal` для текущего site. Targeted run: setup + desktop + mobile `3/3`, `--retries=0`.
- [x] Browser auth state: запуск Playwright из package и из корня использует один абсолютный storage-state path; setup и dependent desktop/mobile projects не расходятся по working directory.
- [x] Services list/update integrity: search и sort не меняют GET `/api/sites`, `name-asc` совпадает с API host/id ordering, а UI update хоста сохраняет исходный `site.id`. `services.spec.ts` desktop/mobile: `15/15`, `--retries=0`.
- [x] Services help: открыть каждый из 20 chapter/frame/Auth/Anti-Bot help controls на desktop/mobile; dialog должен быть единственным по ID, получать focus, закрываться Escape с возвратом focus и кнопкой Close. DNSBL help должен иметь собственную кнопку и модалку. Targeted run: `3/3`, `--retries=0`.
- [x] Services editor draft/cancel: изменить host/id/upstream, перейти Easy → Raw → Easy, проверить round-trip полей; верхняя и нижняя Back должны вернуть в список без создания site по API. Desktop/mobile: `3/3`, `--retries=0`.
- [x] Services custom error pages: Disable all/Enable all должны переключать весь список, 404 — отключаться отдельно, Preview — открывать `/api/error-pages/preview/404`; после save/readback и compile/apply runtime 404 остается обычным, а 500 получает branded HTML и `Server-Timing`. Desktop/mobile: `2/2` каждый, без retry.
- [x] Services responsive editor: пройти все 11 wizard tabs на desktop/mobile и проверить отсутствие горизонтального document/panel overflow и controls вне viewport. Blocking, Geo и ModSecurity grids должны сжиматься до одной колонки на узком экране. Targeted run: `3/3`, без retry.
- [x] Services unsaved/keyboard: label host переводит focus в input, Tab — в Service ID; изменение формы активирует beforeunload и confirm на обеих Back. Dismiss сохраняет URL/draft, Accept выходит без API mutation, успешные Save/Delete не показывают повторный discard. Desktop/mobile navigation run: `5/5`, без retry.
- [x] Services Anti-Bot preview: cookie показывает notice без popup; JavaScript v4 открывает `/api/error-pages/preview/antibot-v4`, captcha v4 — `/api/error-pages/preview/captcha-v4`; save/readback сохраняет captcha + v4. Desktop/mobile: `2/2` каждый, без retry.
- [x] Services certificate: выбрать self-signed cert через import picker, получить ошибку на invalid ZIP, запросить export approval отдельным TOTP-пользователем, одобрить другим admin и скачать ZIP; сохранить TLS binding и подтвердить `/api/tls-configs`. Desktop/mobile: `2/2` каждый, без retry.

- [x] Services validation/reference matrix: desktop/mobile submit отклоняет пустые Service ID/host/upstream host, port 0 и 65536, а также invalid upstream scheme; API отклоняет duplicate primary host, missing/nonexistent site reference и cross-site upstream reassignment без частичной записи. Targeted runs: desktop `2/2`, mobile `2/2`, `--retries=0`.
- [x] Services Traffic/ModSecurity: включить request limit `1r/s` и custom ModSecurity rule в видимых вкладках, сохранить, подтвердить API profile readback и собственную active revision; runtime обязан вернуть `429` на flood и `403` на rule path. Desktop/mobile: `2/2` каждый, `--retries=0`.

- [x] Administration → Users/Roles Delete: на desktop и mobile сначала отменить confirm и убедиться через API, что сущность сохранилась; затем подтвердить и проверить исчезновение строки, API absence, audit event и cleanup. Targeted browser run: `5/5`, `--retries=0`.
- [x] Administration RBAC: пользователь без обеих write-permissions не видит Create/Edit/Delete; прямой DELETE защищён сервером сочетанием `administration.write` и соответствующего `administration.users.write`/`administration.roles.write`. Built-in admin и стандартные роли не имеют Delete-кнопки.

- [x] Services list: select one/select all/unselect all, row navigation, Edit/Back и external service popup с точным URL подтверждены на desktop/mobile.
- [x] Services import: selected ENV export, file chooser cancel без мутации, valid import, site/upstream API readback, editor prefill, cleanup и revision restore подтверждены на desktop/mobile.
- [x] Services enable/disable: после UI-клика дождаться смены кнопки и API `enabled`; disabled host не должен попадать в upstream, enabled host должен снова вернуть upstream echo после active revision apply.
- [x] Services resilience: normal, loading, empty и upstream 503 list states подтверждены на desktop/mobile.
- [x] Services RBAC: роль только с resource read permissions видит список/просмотр, не видит mutation controls; прямой POST `/api/sites` возвращает 403.

- [x] Dashboard API/runtime: deterministic seed подтверждает 24 hourly buckets и точное равенство requests/attacks/blocked series дневным totals; top IP/country/URL/error breakdowns непустые, Docker overview/logs читаются для реального контейнера.
- [x] Dashboard widgets/details: Services, traffic counters, top lists, popular errors, containers, CPU и Memory открывают detail; modal закрывается overlay, Escape и кнопкой Close.
- [x] Dashboard layout: initially-hidden widget сохраняется после reload; desktop resize изменяет geometry/localStorage и переживает reload, Reset возвращает normalized default; mobile не включает drag resize.
- [x] Dashboard CPU/Memory: labels, used/free/total или cores/goroutines/heap, progress width и границы 0/fraction/100/>100 clamp подтверждены на desktop/mobile.
- [x] Dashboard resilience: loading, upstream 503, empty, zero и partial payload визуально отрисовываются без пустого mount и без необработанной ошибки. Dashboard run 19/19, полный suite 133/133 без retry/skipped.

- [x] Administration → Users: создать и изменить disposable пользователя, нажать Delete, подтвердить диалог; строка должна исчезнуть, API readback не должен содержать пользователя, audit должен содержать `administration.user.delete`, активные сессии пользователя должны быть отозваны. Built-in admin не должен иметь кнопку удаления.
- [x] Administration → Roles: создать и изменить custom role, нажать Delete и подтвердить; API и таблица не должны содержать роль, audit должен содержать `administration.role.delete`. Стандартные и назначенные пользователям роли удалить нельзя.
- [x] Browser infrastructure: общий helper подтверждает navigation/API payload/toast/modal/loading/error/stable-DOM ожидания на desktop/mobile; полный run 119/119 прошёл без retry.
- [x] E2E cleanup/runtime: после mutation workflows не остаются Services/Bans/TLS/Administration объекты; active revision возвращается к исходной. Deterministic seed подтверждает Requests, Dashboard, Events и Audit.
- [x] Runtime compile/apply: сервис с E2E host-map ключами сохраняется и удаляется без nginx `could not build map_hash`; active revision restore проходит после mutation.
- [x] Browser E2E: disposable management host https://e2e-management.test:10443 открывает /login и все 12 реализованных вкладок; Chromium mapping направляет host на 127.0.0.1.
- [x] Browser E2E: desktop и mobile suites завершены без зависания; лимит теста 20 минут, результат 58/58 passed на каждом viewport и 115/115 в объединённом reporter run вместе с setup.
- [x] Dashboard: виджеты, widget picker, Memory/CPU labels, проценты, progress width и detail modal подтверждены E2E на disposable stack.
- [x] Services: list refresh/search/sort/select-all, editor route, Easy/Raw, settings search, validation и Back подтверждены E2E.
- [x] Bans: create modal, invalid IP validation и закрытие без записи объекта подтверждены E2E.
- [ ] Ручная проверка Vivaldi: /login, /dashboard, /services, /bans; Chromium evidence не заменяет конкретный пользовательский браузер.
- [ ] API/runtime: продолжить mutation/compiler assertions из .work/TASKS.md; текущие 100% относятся только к зарегистрированному browser/read-only набору.
- [x] API/runtime: disposable Services mutation, compiler/runtime artifact parity, allowlist 200/403 behavior and settings/security-mode roundtrip were verified; helper evidence uses actual Compose container names and runtime-discovered attacker IPs.
- [x] Settings → General → Language: через UI последовательно сохранить `en`, `ru`, `de`, `sr`, `zh`; после каждого сохранения проверить API `/api/settings/runtime`, reload и выбранное значение, затем восстановить исходную локаль. Desktop/mobile полный suite и stress run 11/11 прошли без retry.
- [x] Settings language rerender: после смены языка дождаться завершения глобального `app:language-changed` rerender; новая кнопка Save и select должны быть интерактивны, без потери следующего клика из-за замены DOM.
- [x] Browser coverage: registry показывает 100% зарегистрированного среза для Auth, Dashboard, Services, Requests, Bans, Revisions, Anti-DDoS, OWASP CRS, TLS, Administration, Events, Activity и Settings; Settings = 4/4.
## 2026-07-23 — E2E этапы 5, 6 и 9

- Bans: вручную проверить filter IP/site/country/module, Extend/Unban cancel и ошибки read-only пользователя; в API проверить, что 403 не меняет denylist.
- Revisions: вручную проверить bulk Delete others, apply/delete confirmation и видимую ошибку read-only пользователя; в API проверить неизменность active revision после 403.
- TLS: вручную проверить approval ID, повторную проверку approval, invalid/valid TOTP и ZIP download на desktop/mobile; в API проверить distinct approver, fresh step-up и auto-renew 1..365.
- Browser infrastructure: проверить, что auth state сохраняется в `e2e/browser/.auth`, а очистка `test-results` не ломает последующие проекты.
## 2026-07-23 — E2E этап 11 Events

- Events: проверить type/severity/site/date filters и reset, page size и переход на страницу 2 с `aria-current`.
- Detail: открыть строку мышью, Enter и Space; закрыть Escape, overlay и кнопкой; проверить все related fields/details JSON.
- API/UI resilience: проверить RFC3339 validation, loading, empty, 503 и malformed payload без разрушения sidebar/shell.
## 2026-07-23 — E2E этап 12 Activity/Audit

- Activity: проверить presets all time/last hour/last day, category/actor/site/status/date filters и reset.
- Pagination: выбрать 25, пройти Next/Previous и сверить page info с API total/offset.
- API resilience: проверить invalid date/status/category/offset, limit clamp 500, loading/empty/503/malformed без разрушения shell.
- Audit evidence: точечно проверить actions Bans, Revisions, Anti-DDoS, CRS, TLS и Administration.
## 2026-07-23 — E2E этап 13 Settings

- Storage/Security: проверить retention, indexes streams, login rate ranges, direct-IP/Vault/export toggles, reload и restore; invalid mixed PUT не должен частично сохраняться.
- Logging/Secrets: проверить file/OpenSearch/ClickHouse routes/migration, masked placeholders и show/hide; masked Vault save не должен делать network write.
- Management Hosts: проверить enabled-site ownership, invalid/unowned/stale version, add/remove, drift/apply-required и восстановление.
- Appearance/Updates: проверить login/2FA/healthcheck previews, save/reload и update-check success/disabled/offline.

## 2026-07-23 — E2E этап 14 Cross-module/security

- [x] Settings initial hydration: при открытии любой панели `#settings-page` должен иметь `aria-busy=true` и быть `inert`; после загрузки runtime — `aria-busy=false`, `data-runtime-ready=true`, после чего controls принимают ввод без возврата к дефолтам.
- [x] Session/RBAC: удалить disposable reader с активной сессией, проверить отзыв сессии и возврат на login с причиной; reader читает разрешённые endpoint, а критические writes получают server-side `403`.
- [x] Compile/apply/rollback: по отдельности изменить Settings, Services, TLS и Anti-DDoS, скомпилировать candidate, применить, сверить active revision и вернуть исходную; management host и обычный upstream должны оставаться доступны.
- [x] Runtime reload: после SIGHUP readiness должен подтвердить точный `X-WAF-Runtime-Revision` через новое соединение, не принимая ответ keep-alive старого worker.
- [x] Resilience/i18n: проверить `429`, `403`, `5xx`, malformed и slow API без разрушения shell; Dashboard, Events, Activity, Settings и modal проверить во всех `ru/en/zh/de/sr`.
- [x] Security headers: вручную сверить CSP, защитные headers, cookie flags `Secure`/`HttpOnly`/`SameSite` и отсутствие password/session ID в DOM и console.
- [x] Evidence: cross-module desktop/mobile `11/11`; объединённый этапов 11–14 `41/41` за 5.7 минуты, без retry/flaky/skipped.

## 2026-07-23 — повторный строгий аудит Этапа 0

- [x] Services enable/disable: проверить реальный порядок Easy profile/site, API `enabled`, состояние Toggle и runtime. Подтверждено targeted `2/2`: disabled host возвращает branded 421 с revision header и не достигает upstream; enabled host возвращает реальный upstream HTTP 200.
- [x] Dashboard CPU/Memory: заголовок должен точно соответствовать текущей локали `en/ru/de/sr/zh`; summary metrics и каждая process row/PID должны совпадать с live `/api/dashboard/stats`. Targeted `3/3` green.
- [x] Events localization: Dashboard/Events/Activity/Settings chrome и Events detail modal проверены для пяти локалей; динамические audit actions не трактуются как i18n keys. Targeted `2/2` green.
- [x] Bans: self-seeded 12-IP policy обязан дать 10+2 строки, mouse/Enter/Space detail и cancel без мутации; invalid duration не отправляет `/ban` POST. Targeted оба сценария `2/2` green.
- [x] TLS export: invalid TOTP обязан вернуть 401, не отправить export POST и не создать download; valid fresh TOTP создаёт ровно один ZIP download. Targeted `2/2` green.
- [x] Anti-DDoS logs: выбранная реальная security row обязана совпасть по пяти значениям с 11-полевым detail modal. Targeted `2/2` green.
- [ ] Ручной/UI остаток: после нового full shard run проверить artifacts каждого отдельного Compose project и отсутствие global language/settings/revision drift между shard.
- [ ] API/runtime остаток: повторить live Events filter/pagination после cache fallback fix; затем выполнить полный API registry evidence и aggregate coverage.

## 2026-07-24 — Stage 0 final strict evidence

- [x] Requests → отказ runtime: открыть `/requests`, остановить runtime в disposable-стенде и нажать Refresh; не позднее 15 секунд вместо `Loading requests...` должно появиться локализованное сообщение ошибки, при этом sidebar и основной shell остаются видимыми.
- [x] Browser navigation: во время краткого reload/apply revision переход на `/login` или страницу Services переносит только преддокументные `ERR_NETWORK_CHANGED`/`ERR_CONNECTION_CLOSED`; после повторной попытки UI должен пройти обычную готовность элемента, а assertion не повторяется.
- [x] OWASP CRS: при недоступности официального release endpoint кнопки Check/Update возвращаются в интерактивное состояние, консоль показывает диагностическую ошибку, а активная версия правил не меняется; при доступном endpoint по-прежнему проверяется успешный update.

- [x] Requests → отказ runtime: открыть `/requests`, остановить runtime в disposable-стенде и нажать Refresh; не позднее 15 секунд вместо `Loading requests...` должно появиться локализованное сообщение ошибки, при этом sidebar и основной shell остаются видимыми.
- [x] Browser navigation: во время краткого reload/apply revision переход на `/login` или страницу Services переносит только преддокументные `ERR_NETWORK_CHANGED`/`ERR_CONNECTION_CLOSED`; после повторной попытки UI должен пройти обычную готовность элемента, а assertion не повторяется.
- [x] OWASP CRS: при недоступности официального release endpoint кнопки Check/Update возвращаются в интерактивное состояние, консоль показывает диагностическую ошибку, а активная версия правил не меняется; при доступном endpoint по-прежнему проверяется успешный update.

- [x] Services → Anti-Bot → Protection order: choose `Anti-bot first`, save, verify `/api/easy-site-profiles/:siteID` returns `security_auth_basic.auth_order=antibot_first`, reload the editor and confirm the same selected option on desktop and mobile. This caught and fixed the missing runtime-hydration mapping on 2026-07-24; fresh strict result `3/3`, no skipped/flaky/retries.
- [x] Services → Front → incoming mTLS: on a disposable stack upload a trusted client CA through the product API, enable required client-certificate verification, explicitly compile/apply, and verify the live HTTPS WAF rejects a client without a certificate, proxies a client bearing a certificate issued by that CA, then permits the no-certificate client again after disable and a second compile/apply. Fresh tagged evidence: `TestE2EIncomingMTLSClientCertificateRuntime`, `1/1`, JSON `test-results/strict-audit/stage3-incoming-mtls/go-results.json`.
- [x] Services → every select field: change security/profile/CA, upstream scheme, rate unit, ban scope, Anti-Bot template/challenge/escalation and Authentication mode/order/template/TTL; save, verify the exact Easy Profile and upstream API values, reload, and confirm every selected value again on desktop and mobile. Fresh disposable proof `3/3`, strict JSON `test-results/strict-audit/stage3-selects/results.json`; `r/m` is now a supported persisted value rather than a silent fallback to `100r/s`.
- [x] Services → runtime behavioral protections: on a fresh stack persist and explicitly apply an Anti-Bot captcha profile using `1r/m`; require redirect, challenge page and verify-cookie behavior through WAF, then check Geo, unknown-host, ACME HTTP-01 and direct-IP runtime responses. Tagged `TestE2EBehavioral` passed `6/6` in 49.643 s; JSON `test-results/strict-audit/stage3-behavioral/go-results.json`.
- [x] Services → Virtual Patches: add URI/block patch, verify API readback, reload and see the patch, delete it and verify API absence on desktop/mobile. The tagged runtime check must then prove the explicit compile/apply changes the real WAF response from `403` to `200` after removal.

- [x] Dashboard fresh-stack telemetry: before opening Dashboard, append deterministic runtime access-log entries spanning 24 hours and two sites; require non-empty top attacker IP/country/URL/error widgets, then confirm desktop/mobile `dashboard.spec.ts` opens details and preserves all data. The 2026-07-24 disposable proof completed `19/19` with no skipped/flaky/retries; API/runtime proof additionally generated a real anti-bot response from the attacker container and observed it in Dashboard.

- [x] Browser shards are self-contained: Activity creates its own audit mutations and pagination rows; Management Hosts creates and removes its own site/host; resilience pages reuse the authenticated context. Four fresh stacks completed core `119/119`, services `45/45`, platform `51/51`, and settings-cross `29/29` with zero skipped, flaky, retries, or failures.
- [x] Runtime mutation proof: Services enable/disable, compiler/apply, L7 policy, Bans, Revisions, Events, Activity, Settings, TLS, and RBAC assertions exercised API readback and the real WAF runtime. Auto-start creates a disposable site/upstream, applies the revision, and now requires a real HTTP `429` from the runtime HTTP listener.
- [x] Direct-IP contract: the behavioral workflow uses an unclaimed loopback Host on the same mapped runtime port, so it verifies the default-server branded `421` response when disabled and a dropped connection when direct-IP blocking is enabled; it cannot accidentally target a site created by another E2E.
- [x] Fresh onboarding contract: an independent stack without bootstrap admin completes admin bootstrap, service/upstream creation, self-signed certificate binding, compile/apply, and authenticated HTTPS login.
- [x] Manual follow-up: when validating a deployed environment, verify that an unclaimed direct-IP Host receives the branded `421` page while direct-IP blocking is disabled and is dropped after enabling it; do not use a host currently mapped to a test or production site.
