# Deploy Lab Workspace

Назначение: `experimental/lab` контур развертывания TARINIO.

Структура:

- `docker`
- `k8s`
- `terraform`
- `toolchain`
- `ci`
- `docs`
- `scripts`

Перед началом работ обязательно прочитать:

- `.work/DEPLOY_SCOPE.md`
- `.work/SUPPORTED_PROFILE_CRITERIA.md`
- `.work/TASKS2.md`

## Что уже есть

- baseline bootstrap/preflight;
- Kubernetes single-node lab manifests + smoke;
- profiles: `ha-control-plane`, `opensearch-clickhouse`, `guardrails`;
- Terraform orchestrator (`init/plan/apply/destroy`);
- CI cycle (`run-lab-cycle.ps1`);
- RU/EN runbook.

## Release No-Trace Tests

Для release-check цикла добавлены отдельные no-trace preflight тесты:

- `deploy/lab-k8s-terraform/scripts/test-k8s-no-trace.ps1`
- `deploy/lab-k8s-terraform/scripts/test-terraform-no-trace.ps1`

Требование: оба теста не должны оставлять следов в системе/репозитории.

