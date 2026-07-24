## [Unreleased]

### E2E cleanup (2026-07-24)

- E2E pipeline jobs are grouped into general, API, UI, browser, and reporting stages so failures and duration are visible by test surface. Browser fault synchronization and settings readiness now wait on the actual runtime/UI state.

- Disposable PowerShell, POSIX E2E, and DAST runners now remove their locally built Compose images together with containers, networks, orphan resources, and named volumes, including after a failed build retry. Cleanup remains scoped to the selected E2E Compose project and does not delete shared/base Docker images.

### E2E re-audit (2026-07-24)

- Fixed the PowerShell disposable E2E runner to publish the namespaced DAST canary URL instead of leaving negative-security probes on the fixed default port.
- Verified four real ModSecurity attack payloads receive HTTP 403 without reaching the canary, while a benign request reaches it once.
- Activity/Audit and Settings/cross-module shards passed their real API/runtime and desktop/mobile browser checks without retries or skips. Full tagged Go E2E remains open because the aggregate run exceeded its 10-minute limit.

## [1.5.14] - 22.07.2026

### Browser E2E foundation

- Added an isolated Playwright package under e2e/browser with a disposable-stack guard and localhost-only base URL policy.
- Added desktop/mobile route smoke checks, read-only API contracts for every implemented WAF tab, Dashboard widget/CPU/Memory checks, and first Services workflows.
- Extended scripts/run-e2e-tests.ps1 with -Browser to run Go and browser suites on the namespaced disposable stack.
- Browser runner now uses the disposable HTTPS management virtual host with local Chromium host mapping, matching the real guard.js redirect path.
- Added 20-minute per-test timeout, real login-form E2E, Services editor mode/validation workflow, and Bans invalid-input validation.
- Verified full desktop and mobile browser suites at 42/42 passed each; registry coverage reports 100% for the currently registered workflow slice.
- Fixed disposable API/runtime evidence helpers to use Compose service container names and dynamically discovered attacker IPs; Services mutation/editor parity and settings roundtrip suites now run against the actual stack.

### Интерфейс входа

- Устранено кратковременное появление legacy-страницы при обновлении `/login` и `/login/2fa`: CSP-safe bootstrap-стиль подключается первым, скрывает экран до установки сохранённой темы и не требует inline CSS.
- Экран входа становится видимым только после выбора оформления и следующего кадра браузера, поэтому пользователь сразу видит сохранённую тему.
- Legacy-правила centered login card удалены из общего stylesheet; login и 2FA используют только выбранные тематические стили.

### Basic Auth

- Страница Basic Auth получает широкий и компактный логотипы TARINIO, а также favicon из локальных runtime-ресурсов `/auth/assets/...`.
- Запросы оформления Basic Auth не направляются в upstream защищаемого сайта.

### Runtime и применение ревизий

- Проверка готовности revision после reload вынесена на отдельный loopback-only маршрут Nginx. Она не зависит от upstream, host-map, AntiBot, Basic Auth и правил защищаемого сайта.
- Перед reload launcher выполняет `nginx -t`; невалидная конфигурация сразу возвращает точную причину ошибки вместо ложного таймаута и незаметного сохранения старой revision.
- Добавлен E2E-сценарий применения valid revision с недоступным upstream: revision должна стать активной, а runtime обязан подтвердить свой маркер локально.

### Проверки

- Добавлены контрактные проверки раннего bootstrap-скрытия login/2FA, локальных Basic Auth assets и изолированного readiness-маршрута runtime.
- Полный набор Go-тестов и целевые E2E-сценарии Basic Auth и reload readiness проходят успешно.

### Безопасность и E2E

- E2E теперь завершает `compile + apply` только после подтверждения, что runtime переключился именно на созданную revision. При нарушении инварианта отчёт содержит активный указатель revision и последние логи runtime, а не маскирует проблему последующей HTTP-проверкой.
- Повтор Docker Compose при временной ошибке Docker Registry использует возрастающую паузу и отдельно отмечает сетевую нестабильность в evidence; функциональные ошибки сборки и тестов по-прежнему не ретраятся как успешные.
- Kubernetes manifest учебных HTTP-сервисов переведены на `RuntimeDefault` seccomp, read-only root filesystem, запрет повышения привилегий и пустой набор capabilities. У runtime удалена необоснованная capability `NET_RAW`; для сетевой защиты остаётся только `NET_ADMIN`.
- Обновлён reviewed baseline образа Debian 13: из него удалены 14 закрытых upstream CVE. Остались только 22 подтверждённых уязвимости без опубликованного исправления у поставщика; любая новая находка или CVE с доступным фиксирующим пакетом блокирует pipeline.
- DAST baseline стал переносимым между GitLab Linux runner и Docker Desktop: ZAP использует временный каталог внутри контейнера, отчёты пишет в выделенный evidence mount, а принудительный UID/GID применяется только на Linux. Это исключает зависание ZAP из-за недоступного host-mounted home-каталога.
- Устранена гонка асинхронного Dev Fast Start и API-применения revision: bootstrap больше не может применить устаревший снимок после уже активированной оператором revision. В single-node режиме все apply дополнительно сериализуются process-local lock; добавлен regression-тест stale bootstrap apply.
- Убраны все HIGH/CRITICAL Kubernetes misconfiguration в lab manifests и profile overlays: включены `RuntimeDefault` seccomp, запрет повышения привилегий, минимальные capabilities и read-only root filesystem с явными `emptyDir`/PVC для state, logs, runtime и temporary data.
- Служебные образы CLI, sentinel, request archive и HA/enterprise toolbox запускаются непривилегированными UID. Sentinel получает только read-only группу Nginx access log, сохраняя запись state и adaptive output в выделенные volumes; это подтверждено L4/L7 E2E.
## Unreleased

- bans/e2e: повторный Stage 5 audit на fresh disposable stack прошёл без skip/retry: tagged Go API/runtime `2/2` (с negative validation subtests), isolated desktop/mobile Playwright shard `9/9`. Подтверждены реальные denylist/audit readback, Create/Extend/Unban/cancel, country/detail, pagination и server-side read-only RBAC `403`.
- requests/api: устранён hidden fallback в `/api/requests`: недоступный runtime request backend теперь честно возвращает `502`, а не успешный пустой список. Handler regression и disposable E2E с pause/unpause runtime подтверждают отказ и восстановление.

- services/authentication: fixed persisted Protection order hydration. The editor now restores both `security_auth_basic.auth_mode` and `auth_order` from the API instead of silently resetting the UI to `auth_first` after reload. Desktop/mobile E2E changes the dropdown, verifies API readback, reloads, and confirms the selected value.
- services/e2e: added real tagged virtual-patch coverage: API create/readback and delete each use an explicit compile/apply, compare the active runtime artifact, and prove the protected URI changes from runtime `403` to `200` after deletion.
- services/virtual-patches: connected the rendered editor to the real Virtual Patches API. The tab now loads persisted entries and performs add/delete mutations with API readback; a desktop/mobile browser E2E is registered in the Services GitLab shard and exact coverage baseline.
- services/mtls: Easy-profile mTLS references created through certificate-material APIs are now staged into the runtime candidate and translated to the material paths accepted by the compiler. Upstream client certificate, private key and CA are staged alongside the incoming client-CA material.
- services/mtls: revision snapshots now retain materials referenced only by Easy-profile mTLS, and HTTPS upstreams now render an HTTPS `proxy_pass` target instead of silently using HTTP. A disposable client-auth upstream proves the WAF uploads material, compiles/applies it, verifies SNI and proxies a real mTLS request successfully.
- services/mtls: added live incoming client-to-WAF mTLS evidence. The tagged E2E uploads an ephemeral trusted CA through the product API, binds real site TLS, explicitly compiles/applies the Easy profile, requires Nginx to reject a certificate-less HTTPS client, proves a CA-signed client reaches the upstream with `200`, then proves disable plus a second apply restores no-certificate access. Candidate and active runtime artifacts are checked for both the enabled and removed directives.
- services/rate-limits: request-rate unit `r/m` is now consistently accepted by the Easy editor serializer and control-plane validation alongside `r/s`; it no longer silently converts to `100r/s` or fails saving. A fresh desktop/mobile Services E2E now persists, reads back and reloads every editor select field through the real API.
- services/runtime: a fresh tagged behavioral E2E now reconfirms the real WAF captcha/verify, Geo, unknown-host, ACME HTTP-01 and direct-IP branches after the `r/m` contract fix; the disposable test stack is removed with its own volumes after evidence capture.

- dashboard/e2e: closed the repeatable Stage 1 audit on a fresh disposable stack. Dashboard telemetry is seeded through real runtime Nginx access-log rows and the browser-only runner now fails closed unless the API reports 24 buckets, two sites, and populated attacker IP/country/URL/error widgets. Desktop/mobile Dashboard passed `19/19` with zero skipped, flaky, retries, or failures; tagged API/runtime checks passed `2/2` and verified a real anti-bot attack is visible in Dashboard aggregation.

- e2e/final-gate: completed the strict Stage 0 proof on 2026-07-24. Fresh isolated browser shards passed core `119/119`, services `45/45`, platform `51/51`, and settings-cross `29/29`; strict JSON verification reported zero skipped, flaky, retries, and failures. Aggregate browser plus API/runtime registry coverage is 100% for every registered tab/workflow.
- e2e/runtime: full tagged `TestE2E` completed on a disposable stack with 291 test entries, zero failures/skips, in 379.886 seconds; the independent no-bootstrap Fresh Onboarding/self-signed HTTPS workflow also passed. `go test ./...` passed in 160.4 seconds.
- e2e/autostart: replaced the permissive Anti-DDoS branch with a real disposable site/upstream/profile workflow: it compiles and applies the revision, then requires an actual runtime HTTP `429` under the configured L7 limit. Auto-start container scoping is explicit and restored after the test.
- e2e/direct-ip: behavioral direct-IP checks now use an unclaimed loopback Host while preserving the mapped runtime port and refusing redirects, preventing a prior E2E site host-map entry from masking default-server `421` or `444` behavior.
- e2e/runner: исправлена передача shard spec в основной npm runner — аргументы теперь получает Playwright, а strict verifier запускается только после него. Это устраняет ошибочный запуск всех 241 тестов в каждом shard и подтверждено точечным `2/2` + zero skip/flaky/retry gate.
- e2e/coverage: API/runtime registry теперь требует machine-readable exact Go test/subtest evidence; missing/fail/skip блокируют gate. Добавлен изолированный `e2e:api-registry`, per-tab 100% baseline и 8 unit guards.
- e2e/strictness: удалены все Playwright и Go skip/fixme/only ветки; Bans, Revisions, Events и Activity получили реальные tagged Go E2E вместо автоматического зачёта registry.
- services/runtime: исправлен порядок enable/disable Easy profile и site; nginx template получил `server_names_hash_max_size 4096` и `server_names_hash_bucket_size 128`, устраняя реальный apply rollback на длинных hostnames. Targeted runtime proof прошёл `2/2`.
- events/api: cached fallback теперь хранит полный retention-scoped snapshot и повторно применяет type/severity/site/date filters, deterministic sort и pagination вместо выдачи неотфильтрованных cached rows.
- browser/e2e: усилены Bans pagination/negative contracts, Revisions clear statuses, TLS step-up denial, Anti-DDoS log detail, Dashboard live CPU/Memory correlation и five-locale Events modal; targeted проверки прошли без retry.
- cross-module/e2e: этап 14 закрыт на desktop/mobile: session revocation/login return, reader RBAC read/write matrix, Settings/Services/TLS/Anti-DDoS compile-apply-rollback с audit, API resilience, пять локалей, CSP/cookies/no-secret-leak. Cross-module run прошёл 11/11, объединённый прогон этапов 11–14 — 41/41 без retry/flaky/skipped.
- settings/ui: вся страница настроек остаётся `inert` и `aria-busy` до завершения initial runtime hydration и публикует `data-runtime-ready` после неё; пользовательские изменения appearance, storage, security, logging и update toggle больше не перезаписываются поздним initial render.
- runtime/reload: проверка revision marker после nginx SIGHUP больше не переиспользует keep-alive старого worker; каждый readiness attempt открывает новое соединение, что исключает ложный stale-revision результат. Добавлен launcher regression test.
- events/activity/e2e: empty-state assertions принимают фактические русские формулировки «Событий пока нет» и «Данных пока нет» на desktop/mobile.
- services/e2e: Basic Auth editor workflow теперь доказан на desktop/mobile: dynamic users, v6 preview, TTL readback, точная маска, reveal/hide и scoped audit assertion; targeted run прошёл `3/3` без retry.
- browser/e2e: storage state setup и desktop/mobile projects используют единый абсолютный путь относительно Playwright package, поэтому isolated и root-level запуски больше не расходятся по working directory и не создают ложные ENOENT/зависания.
- services/e2e: list search/sort теперь доказывают отсутствие API-мутации и точный `name-asc` порядок относительно GET `/api/sites`; update workflow подтверждает неизменность `site.id`. Desktop/mobile Services suite прошёл `15/15` без retry.
- services/ui: восстановлена DNSBL help-кнопка и ее modal renderer; удален duplicate allowlist dialog ID; все help dialogs закрываются Escape и возвращают focus на trigger. Новый desktop/mobile workflow перебирает все 20 фактических help controls и прошёл `3/3` без retry.
- services/e2e: новый editor navigation workflow доказывает Easy/Raw draft round-trip и обе Back-кнопки как cancel без API mutation на desktop/mobile (`3/3`, без retry).
- services/e2e: custom error pages workflow закрывает bulk/individual controls, preview, profile readback, compile/apply и реальное различие disabled 404 / branded 500 с runtime metadata на desktop/mobile.
- services/ui: compact editor grids теперь сжимаются без mobile overflow; Geo и ModSecurity используют adaptive auto-fit columns. Новый workflow проверяет все 11 вкладок и прошёл desktop/mobile `3/3` без retry.
- services/ui: active stable и runtime renderers используют общий unsaved-change guard с beforeunload, Back confirm/dismiss и очисткой после Save/Delete; добавлена локализация en/ru/de/sr/zh и keyboard/label E2E (`5/5` desktop/mobile).
- services/e2e: Anti-Bot template preview workflow проверяет cookie notice, JavaScript/captcha popup routes и persisted v4 selection на desktop/mobile без retry.
- services/security: certificate export в active stable/runtime editor переведен с прямого POST на общий approval + TOTP step-up; E2E закрывает picker, invalid archive, protected ZIP download и TLS binding на desktop/mobile.
- services/validation: primary host теперь уникален без учёта регистра; upstream нельзя переназначить другому site; Easy editor валидирует scheme `http|https` во всех active/compat validation paths с локализацией ru/en/de/sr/zh.
- services/e2e: добавлена полная validation/reference matrix с UI submit, API/runtime readback, duplicate host, nonexistent references, immutable cross-site ownership и strict cleanup; desktop/mobile targeted runs прошли без retry.
- services/e2e: новый vertical workflow сохраняет rate limit и custom ModSecurity через UI, проверяет persisted profile, собственную active revision и реальные runtime `429`/`403` на desktop/mobile.
- control-plane/revisions: локальные compile операции сериализованы поверх distributed coordinator, исключая duplicate `rev-N` при параллельных E2E; конкурентный allocator regression test проходит 20 повторов.

- services/e2e: добавлены select one/all/unselect, row/edit/Back и external popup URL на desktop/mobile; popup destination проверяется локальным browser route без внешней сети.
- services/e2e: valid ENV import проходит полный цикл selected export, delete, chooser cancel, UI import, site/upstream readback, editor prefill, cleanup и active-revision restore на desktop/mobile.
- services/e2e: enable/disable подтверждает UI/API readback и реальное переключение active runtime между upstream virtual host и default management response; list loading/empty/503 states покрыты на desktop/mobile.
- services/rbac: read-only роли видят список и detail, но не Create/Import/Delete/selection/Toggle; прямой write API остаётся server-side 403. Исправлены оба активных/compat list renderer, добавлен contract против рассинхрона.
- dashboard/e2e: Этап 1 закрыт на 100% (API/runtime 9/9, Browser UI 14/14, registry 10/10); добавлены real seeded totals/top data, Docker overview/logs, details, tooltip, picker persistence, resize/reset и resilience edge states.
- dashboard: 24-hour totals приведены к тем же 24 календарным hourly buckets, что отображает график; устранён рассинхрон daily total/series на старом неполном часе.
- dashboard/ui: initially-hidden widgets теперь сохраняются после reload, а Memory label и progress bar одинаково clamp’ятся к диапазону 0–100%.
- ui/e2e: добавлен bounded navigation retry только для pre-document Chromium network errors; assertions не ретраятся. Dashboard прошёл 19/19, полный desktop/mobile suite — 133/133 без Playwright retry/skipped.
- administration: добавлены DELETE API и UI-кнопки для пользователей и ролей; DELETE зарегистрирован в zero-trust router с global+entity write permissions, удаление пользователя отзывает сессии, built-in сущности и назначенные роли защищены, операции пишутся в audit. Read-only UI скрывает Create/Edit/Delete.
- ui/e2e: Administration workflows расширены до create/edit/delete/readback на desktop/mobile; проверяются cancel без мутации, confirm, API absence, audit и обязательный cleanup. Targeted run прошёл `5/5` без retry.
- e2e: Этап 0 закрыт общими wait helpers, compact namespace, strict cleanup ledger, active-revision restore guard, deterministic Requests/Dashboard/Events/Audit seed и registry regression/missing-ID gates.
- compiler/runtime: в nginx template добавлены `map_hash_max_size 4096` и `map_hash_bucket_size 128`, устраняющие reload failure на host-map ключах; добавлен compiler contract.
- ui/e2e: финальный Stage 0 browser run прошёл 119/119 без retry, coverage regression gate и полный `go test ./...` прошли.
- ui/e2e: добавлен browser workflow сохранения всех поддерживаемых локалей Settings (`en`, `ru`, `de`, `sr`, `zh`) с UI-действием, API readback, reload persistence и восстановлением исходного значения.
- ui/e2e: language workflow синхронизирован с полным `app:language-changed` rerender страницы, чтобы следующий клик выполнялся по актуальному DOM; stress run прошёл 11/11 без retry.
- ui/e2e: итоговые desktop/mobile suites прошли 58/58 на каждом viewport, объединённый JSON-reporter run — 115/115 вместе с setup; registry coverage показывает Settings 4/4 и 100% всех зарегистрированных browser workflow.
- ui/e2e: добавлены реальные browser workflow для Services export/invalid import, Revisions clear-statuses и Bans create cancel; cancel использует видимую кнопку вместо перекрываемого modal overlay.
- ui/e2e: добавлен Requests pagination/sort/detail workflow; полный browser suite подтверждён 49/49 на desktop и mobile.
- ui/e2e: добавлены Services UI Save/Delete и bulk Delete confirm/cancel mutation workflows с уникальным API seed, повторным чтением и обязательным cleanup.
- ui/e2e: добавлен Bans create/extend/unban mutation workflow с denylist readback, audit assertion и cleanup.
- ui/e2e/ui: добавлены Administration user/role create-edit-readback workflows; исправлен z-index modal над sidebar, добавлен CSS contract.
- ui/e2e: добавлен TLS certificate metadata и site binding create/delete workflow с API readback и cleanup.
- ui/e2e: добавлен обратимый Settings runtime persistence workflow с API readback, reload и восстановлением исходного значения.
- ui/e2e: добавлен Anti-DDoS valid save/invalid boundary/reload/restore workflow с проверкой отсутствия partial save.
### E2E completion — Bans, Revisions, TLS

- Completed Bans desktop/mobile workflows, including real read-only 403 error paths for create, extend and unban without denylist mutation.
- Completed Revisions desktop/mobile workflows with bulk delete, cross-role RBAC, visible apply errors and active-revision invariants.
- Completed TLS desktop/mobile workflows with self-signed/ACME, auto-renew validation/runtime coverage, PEM/ZIP import and distinct-approver TOTP-protected export download.
- Moved Playwright authentication state outside `test-results` so result cleanup cannot invalidate dependent projects.
### Events E2E completion

- Added server-side Events type/severity/site/date filtering, RFC3339 validation, deterministic ordering and pagination-before-response semantics.
- Completed Events desktop/mobile coverage for detail, keyboard, pagination, loading, empty, backend-error and malformed states.
- Added accessible `aria-current` marking to the active Events pager button.
### Activity/Audit E2E completion

- Added server-side Audit category filtering with totals calculated before pagination, plus validated dates/status/category/offset and a 500-row limit clamp.
- Completed desktop/mobile Activity presets, filters, keyboard submit, previous/next pagination and resilience states.
- Added direct evidence queries for critical Bans, Revisions, Anti-DDoS, CRS, TLS and Administration mutations.
### Settings E2E completion

- Completed all Settings desktop/mobile workflows for runtime, locales, storage, security, logging, secrets, management hosts, appearance and update checks.
- Made runtime settings updates transactional so rejected mixed payloads cannot partially mutate in-memory state.
- Normalized storage-index responses with canonical streams and per-item storage types, including offline empty payloads.
- Prevented masked Vault saves from rewriting unchanged secrets and removed global-pepper coupling from Vault writes.
