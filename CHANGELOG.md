## [1.5.15] - 24.07.2026

### CI and Requests reliability

- The disposable E2E healthcheck now starts only `deploy/compose/e2e` under a unique Compose project. It verifies control-plane/runtime health, sentinel output publication, and actual sentinel write permissions on its isolated state/output volumes; the production default Compose profile is not part of any E2E path.
- Dashboard CPU and Memory widgets now consistently use the Docker container overview: the aggregate and every displayed row are container metrics, and their detail views identify the corresponding container rather than mixing in host-process values.
- Dashboard now renders statistics and the container overview from one snapshot. CPU preserves Docker's per-container percentages and exposes the host capacity for a correctly scaled progress bar; Memory aggregate is weighted by actual container usage and limits.
- Added an isolated default-profile health regression job. It renders `deploy/compose/default` with a temporary unique-name override, so the production stack remains untouched while the sentinel self-healing write paths are tested.
- Sentinel in the default profile now repairs ownership of its mounted state/output volumes at startup and immediately drops to UID/GID `65532:4`; this fixes persistent `permission denied` output/state errors without keeping the model process privileged.
- Split the longest browser E2E groups into independently isolated Dashboard, Records, Services preview, Platform, Settings, and cross-module jobs. Each slice has its own Compose project and host ports, so the four runner slots can execute safely in parallel while the E2E gate still requires every result with no skips.
- Requests now starts the runtime request together with auxiliary API reads and gives every dependency the same bounded abort signal. A paused runtime therefore produces the visible Requests error instead of leaving the UI in a permanent loading state.
- Dashboard telemetry seeding now waits for the real Requests API to expose both normal and blocked rows before browser E2E starts. Browser navigation assertions also wait for page hydration after settings updates and language changes.
- CI warms the pinned Node image before unit work and uses that local image for lockfile, documentation, and audit checks, avoiding transient Docker Hub DNS/auth failures during verification.
- Browser fault evidence now pauses and restores the isolated runtime independently for desktop and mobile; Settings locale evidence waits for runtime hydration rather than a merely visible control.
- Settings and Services editor browser scenarios now wait for the actual hydrated form before reading or mutating controls, preserving their roundtrip and cancel assertions across desktop and mobile.
- Browser page navigation now tolerates a bounded sequence of transient runtime reload connections before authentication; strict result validation still rejects skipped, retried, or failed test executions.
- The pre-security gate now reports compact primary browser failures from Playwright results and the nearby failing log context, excludes report-only aggregate jobs from its failure count, and retains forbidden skipped executions as an explicit blocker.
- Anti-Bot preview E2E now waits for the real Service editor form and its independently mounted wizard tab before interacting with the preview controls.
- Browser E2E now waits for the hydrated Services and Dashboard surfaces after navigation, reload, and transient runtime reconnection. Slow Events interception holds a deterministic loading state until the assertion observes it, and certificate-export setup uses the shared transient-navigation recovery.
- CI browser shards are serialized on the shared shell runner, and Dashboard Docker metrics are scoped to the current Compose project; concurrent disposable stacks can no longer contaminate each other's container rows or exhaust the runner during long UI suites. The disposable default-profile healthcheck resets the production `.env` dependency, uses internal test credentials, and warms its pinned OpenSearch image before enforcing `--pull never`.
- Language changes now await the application's asynchronous sidebar, metadata, and page rerender before resolving, preventing overlapping navigation from leaving raw i18n keys in long cross-locale browser suites.
- Dashboard CPU/Memory browser assertions pin one real container-overview response while checking aggregate and row values, use row-local container-name locators, and validate detail rows by unique container name instead of an unspecified Docker metric order; background polling and locator/order assumptions can no longer create false mismatches.
- Generic browser navigation no longer treats legitimate domain text such as `app.example.com` as an untranslated `app.*` key; strict raw-key detection remains scoped to the dedicated cross-locale assertions.
- Browser fixture revision restoration now retries transient catalog timeouts during runtime apply/reload instead of aborting the entire test on the first 30-second fetch interruption; the original revision ID must still be restored within 120 seconds.
- Browser navigation retry also recognizes Playwright's pre-document `page.goto` timeout during a runtime reload, while retaining the existing four-attempt bound and rejecting non-transient navigation errors immediately.
- Dashboard CPU edge assertions preserve Docker totals above 100% while checking that only the visual progress width is capacity-clamped; Memory percentage remains clamped to the 0–100 display range.
- GitHub mirror/release publication is explicit opt-in via `PUBLISH_GITHUB=1`; ordinary verified `main` pipelines perform no GitHub or GHCR writes.
- Cross-locale settings saves no longer let the server-language metadata refresh re-enter and overwrite the user's next locale during the awaited page rerender; initial metadata synchronization remains unchanged.
- Stack healthcheck now uses its own pipeline-derived host port range, preventing its disposable MTLS/DAST services from colliding with parallel E2E jobs on the shared runner.
- The document language attribute is now committed only after all asynchronous language-change rerenders finish, making locale switches atomic instead of exposing an early intermediate success signal.
- The CI browser image now installs the lockfile dependencies at image-build time; individual shards run the baked Playwright toolchain and no longer perform repeated runtime downloads from npm registry.
- Requests browser coverage now clears the exercised security-reason filter before validating full-dataset pagination, so filtered-result counts are not mistaken for a pagination failure.
- Cross-locale browser coverage now waits for and verifies each real runtime-settings PUT before asserting the atomic DOM-language commit, including restoration of the original locale.
- Browser E2E shards now run concurrently in runner-slot-specific host-port ranges while retaining isolated Compose projects, networks, and volumes; their Playwright image is built once per pipeline, and each shard waits for the disposable management HTTPS endpoint before browser setup.
- Dynamic service-control read-back coverage now waits for the asynchronously hydrated editor after navigation instead of treating the initial HTML shell as a rendered form.
- Bans and revision timeline browser coverage now waits for page-specific asynchronously rendered anchors after navigation and compile operations.
- Locale persistence coverage now verifies the stored runtime setting through API read-back instead of depending on a response event across a full language rerender; service navigation similarly waits for the asynchronously restored list after returning from the editor.
- The disposable E2E Vault healthcheck now allows enough startup time under concurrent stack builds, avoiding a false unhealthy dependency failure during CPU-heavy runner startup.
- Revision apply/delete coverage now waits for the target service tile after rollback navigation before opening the revision modal.
- Shared browser authentication now polls the real session/login endpoints through short runtime reload gaps, while TLS and responsive-editor coverage waits for page-specific hydrated controls before interaction.
- Cross-locale coverage now waits for the runtime-ready settings contract before clicking Save and tolerates only transient read-back transport gaps inside the bounded persistence assertion.
- Settings validation coverage now enters the page through its runtime-ready contract before switching tabs and asserting atomic validation behavior.
- Browser shards are distributed across three independent resource lanes: the 6-vCPU runner keeps three fully isolated stacks active while reserving scheduler and Docker headroom, avoiding the measured load-average-13 starvation seen with four simultaneous stack builds.
- Logging-settings API assertions now tolerate only bounded transport gaps caused by the same stack's runtime reload while preserving exact persisted-value and restore checks.
- Disposable E2E Vault bootstrap now polls readable initialization/seal state and verifies init/unseal transitions, eliminating the one-shot empty-status race that could leave a fresh Vault unhealthy indefinitely.
- Revision modal and status-clear coverage now waits for the asynchronously rendered revisions page or service tile instead of treating DOMContentLoaded as application readiness.
- Candidate staging now treats an active revision as immutable: identical reapplies are no-ops and differing content is rejected, preventing runtime symlinks from observing a half-rewritten bundle with missing per-site artifacts.
- Service list and bulk-delete coverage now waits for hydrated list controls both initially and after fixture creation, preserving cleanup time instead of blocking on an absent shell-era locator.
- Security-settings roundtrip coverage now waits for the page's runtime-ready contract before reading and mutating direct-IP settings.
- Dashboard live-navigation coverage now consistently waits for hydrated page/widget anchors after initial entry, persisted-layout reloads, and restoration from intercepted resilience states.
- CRS busy-state coverage accepts only a successful official release check or the typed `crs_release_unavailable` external-network result, while Basic Auth reveal coverage now waits for and verifies the real password-reveal POST before asserting UI state.
- Each authenticated browser test now creates its own real server session instead of sharing the setup cookie across dozens of isolated contexts, eliminating cross-test `session_missing` invalidation.
- Events browser coverage now waits for the rendered page contract before asserting intercepted loading, empty, error, and malformed-response states under concurrent CI load.
- The browser gate now merges API/runtime evidence from the existing workflow, security, management-rate-limit, full, and registry shards instead of treating the registry-only shard as complete coverage.
- Sentinel and Playwright CI images now declare unprivileged runtime users; Sentinel prepares its writable named-volume paths at build time and no longer starts as root to change ownership or call `su`.
- The Trivy image baseline records newly reported Debian 13 upstream-unfixed findings; findings with an available fixed version remain blocking.
- Auto-start E2E keeps a bounded 60-second HTTP budget for real profile persistence under four-stack CI load instead of inheriting the generic 20-second UI request timeout.
- Revisions browser coverage retries only the idempotent catalog read across the brief connection gap caused by runtime activation; apply and delete mutations remain single-attempt operations.

### Конвейер E2E и очистка стендов

- Все E2E-job объединены в общий параллельный этап, чтобы четыре слота runner использовались одновременно; отдельным остаётся только этап отчёта. Browser fault-синхронизация, готовность редактора Services и hydration Settings ожидают фактическое состояние runtime/UI.

- Одноразовые PowerShell/POSIX E2E- и DAST-runner удаляют локально собранные Compose-образы вместе с контейнерами, сетями, orphan-ресурсами и именованными volume, включая неудачные попытки сборки. Очистка ограничена выбранным E2E Compose project и не удаляет общие базовые образы.

### Повторная проверка E2E

- Исправлен PowerShell disposable E2E-runner: он публикует namespaced URL DAST-canary вместо фиксированного порта по умолчанию.
- Четыре реальные атакующие нагрузки ModSecurity получают HTTP 403 и не достигают canary, а безопасный запрос достигает его один раз.
- Шарды Activity/Audit и Settings/cross-module проверяют реальный API/runtime и desktop/mobile без повторов и пропусков.
- Dashboard при холодном старте теперь сразу возвращает полный пустой 24-часовой контракт, пока фоновый сборщик формирует первый снимок. Это исключает блокировку API при параллельном запуске изолированных E2E-стендов; после публикации снимка API отдаёт фактические данные.
- E2E Requests теперь доказывает отображение ошибки при остановленном runtime без зависания в состоянии загрузки; browser-навигация учитывает краткие смены сети во время применения revision. Проверка CRS не зависит от внешней сети: при недоступности официального источника она требует сохранения активного набора правил и видимого диагностического статуса.
- Events API E2E теперь явно применяет режим блокировки перед созданием site-scoped ModSecurity события: реальный запрос обязан вернуть `403`, а фильтр по сайту — найти соответствующее runtime-событие.
- Browser E2E повторно открывает страницу только при незавершённой UI-гидратации после успешной навигации; это устраняет переходное состояние runtime reload в Dashboard и редакторе Virtual Patches, не повторяя assertions или сами тестовые сценарии.
- Dashboard не использует устаревший снимок дольше фонового интервала: при старом snapshot запускается неблокирующее обновление, поэтому свежие runtime security-события появляются в API после их записи без подвешивания HTTP-ответа.
- Dashboard не использует устаревший снимок дольше фонового интервала: при старом snapshot запускается неблокирующее обновление, поэтому свежие runtime security-события появляются в API после их записи без подвешивания HTTP-ответа.
- Events API E2E теперь явно применяет режим блокировки перед созданием site-scoped ModSecurity события: реальный запрос обязан вернуть `403`, а фильтр по сайту — найти соответствующее runtime-событие.

## [1.5.14] - 22.07.2026

### Браузерные E2E

- Добавлен изолированный пакет Playwright в `e2e/browser`, с защитой disposable-стенда и политикой base URL только для localhost.
- Добавлены smoke-проверки маршрутов desktop/mobile, read-only API-контракты для всех реализованных вкладок WAF, проверки виджетов Dashboard/CPU/Memory и первые сценарии Services.
- `scripts/run-e2e-tests.ps1` расширен флагом `-Browser` для запуска Go- и browser-наборов на namespaced disposable-стенде.
- Browser runner использует disposable HTTPS management virtual host и локальную Chromium host-map, соответствующую реальному пути редиректа `guard.js`.
- Добавлены 20-минутный таймаут на тест, реальный E2E формы входа, сценарий режимов/валидации редактора Services и валидация неверного ввода в Bans.
- Полные desktop и mobile browser-наборы подтверждены с результатом 42/42 каждый; registry coverage показывает 100% для зарегистрированного среза сценариев.
- Вспомогательные средства evidence для disposable API/runtime переведены на имена Compose-сервисов и динамически найденные IP атакующего; сценарии мутаций/редактора Services и round-trip Settings теперь выполняются на реальном стенде.

### Интерфейс входа

- Устранено кратковременное появление legacy-страницы при обновлении `/login` и `/login/2fa`: CSP-safe bootstrap-стиль подключается первым, скрывает экран до установки сохранённой темы и не требует inline CSS.
- Экран входа становится видимым только после выбора оформления и следующего кадра браузера, поэтому пользователь сразу видит сохранённую тему.
- Legacy-правила centered login card удалены из общего stylesheet; login и 2FA используют только выбранные тематические стили.

### Базовая HTTP-аутентификация

- Страница Basic Auth получает широкий и компактный логотипы TARINIO, а также favicon из локальных runtime-ресурсов `/auth/assets/...`.
- Запросы оформления Basic Auth не направляются в upstream защищаемого сайта.

### Ядро и применение ревизий

- Проверка готовности revision после reload вынесена на отдельный loopback-only маршрут Nginx. Она не зависит от upstream, host-map, AntiBot, Basic Auth и правил защищаемого сайта.
- Перед reload launcher выполняет `nginx -t`; невалидная конфигурация сразу возвращает точную причину ошибки вместо ложного таймаута и незаметного сохранения старой revision.
- Добавлен E2E-сценарий применения valid revision с недоступным upstream: revision должна стать активной, а runtime обязан подтвердить свой маркер локально.

### Проверки и доказательства

- Добавлены контрактные проверки раннего bootstrap-скрытия login/2FA, локальных Basic Auth assets и изолированного readiness-маршрута runtime.
- Полный набор Go-тестов и целевые E2E-сценарии Basic Auth и reload readiness проходят успешно.

### Безопасность, E2E и эксплуатация

- E2E теперь завершает `compile + apply` только после подтверждения, что runtime переключился именно на созданную revision. При нарушении инварианта отчёт содержит активный указатель revision и последние логи runtime, а не маскирует проблему последующей HTTP-проверкой.
- Повтор Docker Compose при временной ошибке Docker Registry использует возрастающую паузу и отдельно отмечает сетевую нестабильность в evidence; функциональные ошибки сборки и тестов по-прежнему не ретраятся как успешные.
- Kubernetes manifest учебных HTTP-сервисов переведены на `RuntimeDefault` seccomp, read-only root filesystem, запрет повышения привилегий и пустой набор capabilities. У runtime удалена необоснованная capability `NET_RAW`; для сетевой защиты остаётся только `NET_ADMIN`.
- Обновлён reviewed baseline образа Debian 13: из него удалены 14 закрытых upstream CVE. Остались только 22 подтверждённых уязвимости без опубликованного исправления у поставщика; любая новая находка или CVE с доступным фиксирующим пакетом блокирует pipeline.
- DAST baseline стал переносимым между GitLab Linux runner и Docker Desktop: ZAP использует временный каталог внутри контейнера, отчёты пишет в выделенный evidence mount, а принудительный UID/GID применяется только на Linux. Это исключает зависание ZAP из-за недоступного host-mounted home-каталога.
- Устранена гонка асинхронного Dev Fast Start и API-применения revision: bootstrap больше не может применить устаревший снимок после уже активированной оператором revision. В single-node режиме все apply дополнительно сериализуются process-local lock; добавлен regression-тест stale bootstrap apply.
- Убраны все HIGH/CRITICAL Kubernetes misconfiguration в lab manifests и profile overlays: включены `RuntimeDefault` seccomp, запрет повышения привилегий, минимальные capabilities и read-only root filesystem с явными `emptyDir`/PVC для state, logs, runtime и temporary data.
- Служебные образы CLI, sentinel, request archive и HA/enterprise toolbox запускаются непривилегированными UID. Sentinel получает только read-only группу Nginx access log, сохраняя запись state и adaptive output в выделенные volumes; это подтверждено L4/L7 E2E.

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
### Завершение E2E: Bans, ревизии и TLS

- Завершены desktop/mobile-сценарии Bans, включая реальные read-only пути с `403` для create, extend и unban без мутации denylist.
- Завершены desktop/mobile-сценарии Revisions: массовое удаление, cross-role RBAC, видимые ошибки apply и инварианты активной revision.
- Завершены desktop/mobile-сценарии TLS: self-signed/ACME, валидация auto-renew/runtime, импорт PEM/ZIP и защищённое TOTP скачивание экспорта другим approver.
- Playwright authentication state вынесен за пределы `test-results`, поэтому очистка результатов не может инвалидировать зависимые проекты.
### Завершение E2E: события

- Добавлены server-side фильтры Events по type/severity/site/date, валидация RFC3339, детерминированная сортировка и пагинация до формирования ответа.
- Завершено desktop/mobile-покрытие Events: detail, keyboard, пагинация, loading, empty, backend-error и malformed states.
- Активная кнопка пагинатора Events получила доступную маркировку `aria-current`.
### Завершение E2E: активность и аудит

- Добавлена server-side фильтрация Audit по category с подсчётом totals до пагинации, а также валидация dates/status/category/offset и ограничение `limit` до 500 строк.
- Завершены desktop/mobile-сценарии Activity: presets, filters, отправка с клавиатуры, previous/next pagination и resilience states.
- Добавлены прямые evidence-запросы для критичных мутаций Bans, Revisions, Anti-DDoS, CRS, TLS и Administration.
### Завершение E2E: настройки

- Завершены все desktop/mobile-сценарии Settings для runtime, locales, storage, security, logging, secrets, management hosts, appearance и проверки обновлений.
- Обновления runtime settings сделаны транзакционными: отклонённый смешанный payload не может частично изменить in-memory state.
- Ответы storage-index нормализованы: canonical streams и тип storage для каждого элемента, включая offline пустые payload.
- Сохранение маскированных Vault-значений больше не переписывает неизменённые secrets; удалена зависимость Vault writes от global pepper.
