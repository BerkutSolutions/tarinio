# Руководство по усилению защиты

Эта страница относится к текущей ветке документации.

## Цель

Этот документ задаёт рекомендуемый production hardening baseline поверх базовых deployment-инструкций.

## Административная граница

Рекомендуемый baseline:

- ограничивать доступ к control-plane доверенными сетями;
- публиковать admin endpoints только по HTTPS;
- не выставлять admin UI напрямую в интернет без контроля;
- использовать именованные операторские аккаунты и второй фактор.

## Секреты

Рекомендуемый baseline:

- не хранить секреты в публичном репозитории;
- менять bootstrap secrets до production use;
- считать certificate exports и API tokens чувствительными артефактами;
- использовать защищённое secret store при возможности.

## Сетевая сегментация

Рекомендуемый baseline:

- отделять public ingress от administrative access;
- держать PostgreSQL и Redis только во внутренних сетях;
- использовать host firewall rules вместе с защитой TARINIO;
- публиковать только действительно нужные порты.

## Безопасность изменений

Рекомендуемый baseline:

- все значимые policy changes проводить через compile/apply;
- держать rollback-ready revisions;
- использовать batched writes для крупных provisioning changes;
- валидировать систему после каждого security-sensitive rollout.
