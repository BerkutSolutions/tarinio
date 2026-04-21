# Политика заметок о релизах

Эта страница относится к текущей ветке документации.

## Назначение

Release notes должны объяснять не только то, что изменилось, но и что оператор должен сделать после релиза.

## Рекомендуемые разделы каждого релиза

- `Changed`
- `Fixed`
- `Operational Impact`
- `Upgrade Notes`
- `Rollback Notes`
- `Documentation Updates`

## Что нужно выделять явно

- storage migrations;
- изменения environment variables;
- compose и image changes;
- изменения поведения HA;
- изменения benchmark или resource profile;
- новые обязательные operational checks.
