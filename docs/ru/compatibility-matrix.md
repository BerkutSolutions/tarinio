# Матрица совместимости

Эта страница относится к текущей ветке документации.

## Назначение

Эта матрица описывает окружения и сочетания компонентов, вокруг которых TARINIO проектируется и проверяется.

## Платформа выполнения

Поддерживаемый базовый сценарий:

- Linux-хосты для production deployment;
- Docker Engine с Docker Compose plugin;
- Debian-based container images для `control-plane` и bundled PostgreSQL;
- runtime на базе NGINX + ModSecurity.

## Сервисы данных

Поддерживаемый продуктовый baseline:

- PostgreSQL `15`
- Redis `7`

Практическое правило:

- держаться близко к документированным major version, если новые версии не были отдельно проверены в lab.

## Режимы развёртывания

Документированные и поддерживаемые сценарии:

- single-node compose deployment;
- default deployment с PostgreSQL-backed storage;
- HA control-plane с общими PostgreSQL и Redis;
- HA lab с observability profile;
- AIO-driven upgrades.

## Браузеры

UI рассчитан на актуальные evergreen browsers:

- текущий Chrome / Chromium;
- текущий Microsoft Edge;
- текущий Firefox.

## Связанные документы

- [Развёртывание](deploy.md)
- [Sizing Guide](sizing.md)
- [Референсные архитектуры](reference-architectures.md)
