## [3.0.7] - 27.04.2026

### Безопасность
- Добавлен управляемый hardening `tcp_timestamps` для runtime-профилей compose (`default`, `auto-start`, `enterprise`, `ha-lab`) через `sysctls` и переменную `WAF_RUNTIME_SYSCTL_TCP_TIMESTAMPS` (по умолчанию `0`).
- Добавлен диагностический скрипт `scripts/collect-waf-hardening.sh` и интеграция в Admin Scripts (`collect-waf-hardening`) для сбора артефактов `tcp_timestamps`/sysctl/TLS/HSTS.
- В support bundle добавлен hardening baseline-артефакт `hardening/baseline.json`.
- Усилен TLS baseline в runtime-шаблоне сайта: зафиксированы `ssl_protocols TLSv1.2 TLSv1.3`, набор `ssl_ciphers`, `ssl_session_*`, `ssl_session_tickets off`, `ssl_ecdh_curve`.
- Реализована управляемая HSTS-политика в easy-профиле: `hsts_enabled`, `hsts_max_age_seconds`, `hsts_include_subdomains`, `hsts_preload`.
- Добавлены ограничения для preload-режима: preload требует `hsts_enabled=true`, `hsts_include_subdomains=true`, `hsts_max_age_seconds>=31536000`.
- Добавлена формальная compliance-политика `security/compliance/deprecated-controls-policy.json` для устаревших scanner-checks (включая HPKP/Expect-CT) со статусом `deprecated_not_applicable`, обоснованием и сроками пересмотра.
- Добавлен preflight-сценарий `scripts/pci-preflight-perimeter.sh` для самопроверки периметра перед внешним сканированием: порты, `tcp_timestamps`, TLS, HSTS/headers, наличие compliance-policy и итоговый pass/fail с артефактами.

### CI/CD и проверки
- Добавлен workflow `.github/workflows/security-regression-gate.yml` с fail-fast политикой: запуск isolated `auto-start`-стека, выполнение perimeter preflight, публикация артефактов проверки.

### UI и локализация
- Прокинута полная цепочка HSTS `UI/API -> control-plane -> compiler -> nginx template`, включая рендер `Strict-Transport-Security`.
- Обновлены i18n-строки версий (`ru/en/zh/de/sr`) и fallback-версия в UI.

### Версии и релиз
- Выполнена release-синхронизация до `3.0.7` в control-plane/UI/docs metadata: `AppVersion`, `package.json`/`package-lock.json`, локальный preflight release-id.

### Документация
- Актуализированы RU/EN разделы `core/operator/integration` под PCI/ASV hardening-цикл: baseline, порядок preflight, интерпретация ASV-результатов (включая deprecated checks), правила безопасного rollback при scan-регрессиях.
