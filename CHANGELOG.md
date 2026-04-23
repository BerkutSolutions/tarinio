## [2.0.11] - 23.04.2026

### Security hardening
- Hardened administration support scripts input validation to block shell metacharacter injection in script environment fields.
- Restricted `/api/dashboard/containers/logs` access by requiring `administration.read` in addition to dashboard/report read permissions.
- Added container-name allowlist checks for container log retrieval.
- Removed insecure control-plane default pepper value and made compose secrets for pepper/runtime token/postgres password explicit required inputs.
- Added stricter Easy profile validation for header and host fields used in NGINX templates to reduce config/template injection risk.
- Added centralized runtime `security` settings with a new UI tab `Settings -> Security`:
  - login brute-force rate limiting controls;
  - policy gate for Vault insecure TLS mode.
- Enforced Vault TLS skip-verify policy checks when saving runtime logging settings.

## [2.0.10] - 23.04.2026

### Готовность Enterprise
- Введено разделение прав для операций ревизий: `revisions.write` для `apply/delete` и отдельное `revisions.approve` для согласования.
- Контур документации синхронизирован с фактической enterprise-моделью (OIDC/SCIM, approval flow, support bundle, HA).
- Политика поддержки усилена до формальной модели `Current` / `Stable` / `LTS 2.0` с фиксированными окнами поддержки и SLA-целями реакции.

### Контроль качества
- Закрыт quality regression в RU wiki: устранены смешанные английские фрагменты в обязательных страницах.
- Добавлены дополнительные проверки качества в CI: `go vet`, полный `go test ./...`, doc/i18n quality suite и сборка docs-сайта.
- Дополнительно вычищены смешанные RU/EN формулировки в расширенном наборе wiki-страниц (`secret-management`, `logging-architecture`, `evidence-and-releases`, `enterprise-identity`, `cookbook`).

### CI/CD и цепочка поставки
- Добавлен workflow `ci-quality` с валидацией кода, документации и контрактов release-артефактов.
- Добавлен workflow `security-supply-chain` с `govulncheck`, `npm audit`, `trivy` и secret scanning.
- Усилена доказательная база релизного процесса через обязательные pipeline-проверки release-артефактов.

### Документация и согласованность
- Полностью синхронизированы RU/EN документы под релиз `2.0.10`.
- Устранено противоречие в HA-доках по enterprise-интеграциям: SSO/SCIM отражены как реализованные возможности, SIEM остаётся вне текущего релиза.
- API-документация обновлена с явным описанием permissions для маршрутов `apply/approve/delete`.
- В `docs/ru/cli-commands.md` добавлены Linux-примеры (bash) для запуска CLI и проверочных сценариев test-профиля.
- Удалён дублирующий англоязычный файл `docs/CLI_COMMANDS.md`; ссылки перенаправлены на веточные CLI-страницы RU/EN.

### Синхронизация версий
- Версия продукта и документации обновлена до `2.0.10` в коде и публичных манифестах.
