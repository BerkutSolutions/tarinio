# Критерии Supported Профиля (Lab)

Документ фиксирует условия перехода `deploy/lab-k8s-terraform` из статуса `experimental/lab` в `supported`.

## Обязательные гейты

1. Стабильные CI-циклы (`apply -> smoke -> teardown`) без ручной правки.
2. Стабильные smoke-проверки HA-профиля (`ha-control-plane`).
3. Выполненный security-baseline (`guardrails`, отсутствие placeholder-секретов, ротация).
4. Актуальные инструкции по upgrade, rollback и troubleshooting.
5. Синхронизация версии и changelog в активном релизном цикле.

Источник истины:

- `.work/SUPPORTED_PROFILE_CRITERIA.md`
