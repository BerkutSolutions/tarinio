## [3.0.4] - 26.04.2026

### Версия и релизные метаданные
- Синхронизирована версия `3.0.4` в релизных точках продукта и документации.
- Обновлены связанные release/docs проверки и UI-метаданные.

### Kubernetes/Terraform Lab (experimental)
- Сформирован отдельный контур `deploy/lab-k8s-terraform/`:
  - baseline для Kubernetes;
  - Terraform-оркестрация;
  - CI/runbook/training/guardrails материалы;
  - no-trace preflight-скрипты.

### Проверки релиза
- В `scripts/release.ps1` добавлены проверки:
  - `lab k8s no-trace preflight`;
  - `lab terraform no-trace preflight`.
- Добавлены no-trace тесты:
  - `deploy/lab-k8s-terraform/scripts/test-k8s-no-trace.ps1`;
  - `deploy/lab-k8s-terraform/scripts/test-terraform-no-trace.ps1`.

### Документация RU/EN
- Проведена синхронизация ключевых разделов core/operators/architecture/HA.
- Исправлены поврежденные RU-страницы с артефактами кодировки:
  - `docs/ru/core-docs/support-lifecycle.md`;
  - `docs/ru/high-availability-docs/antibot-enterprise-validation.md`.
- Исправлены битые строки на главной странице документации:
  - `src/pages/index.jsx`.

### Оглавление и раздел «Развертывание»
- Перестроены выпадающие разделы RU/EN sidebar:
  - `Основные документы / Core Documents`
  - `Архитектурные документы / Architecture Documents`
  - `Документы высокой доступности / High Availability Documents`
  - `Документы по модели / Model Documents`
- Аккордеон `Развертывание / Deployment` приведен к целевому порядку:
  1. `Документы по развертыванию`
  2. `Развертывание через Docker` (используется `core-docs/deploy`, без дубликата)
  3. `Развертывание в Kubernetes`
  4. `Развертывание через Terraform`
- Для раздела развертывания удален дубль Docker-страницы в `deployment-docs`.

### Production-шпаргалки по развертыванию
- Добавлены отдельные production-ориентированные документы:
  - `docs/ru/deployment-docs/index.md`
  - `docs/ru/deployment-docs/kubernetes.md`
  - `docs/ru/deployment-docs/terraform.md`
  - `docs/eng/deployment-docs/index.md`
  - `docs/eng/deployment-docs/kubernetes.md`
  - `docs/eng/deployment-docs/terraform.md`
- Обновлены Docker-гайды как основной путь в core:
  - `docs/ru/core-docs/deploy.md`
  - `docs/eng/core-docs/deploy.md`