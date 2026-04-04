# Berkut Solutions - TARINIO

<p align="center">
  <img src="ui/app/static/logo500x300.png" alt="Berkut TARINIO logo" width="240">
</p>

[English version](README.en.md)

Berkut Solutions - TARINIO - self-hosted платформа защиты и контроля веб-трафика (WAF/CRS + L4/L7 Anti-DDoS) с централизованным управлением изменениями через ревизии.

Актуальная версия: `1.0.6`

## Суть продукта

TARINIO перехватывает входящий трафик до бизнес-приложений, проверяет запросы правилами безопасности, применяет ограничения и анти-DDoS политики, а затем позволяет управляемо публиковать изменения конфигурации через цикл ревизий:

- `compile` -> `validate` -> `apply`
- откат к последней стабильной ревизии при операционных рисках

Это дает воспроизводимость изменений, контроль аудита и единый контур управления для runtime и control-plane.

## Для кого

- Команды, которым нужен self-hosted WAF-контур без внешних SaaS-зависимостей.
- Инфраструктурные и ИБ-команды, которым важны управляемые релизы правил и политик.
- Организации, где критичны трассируемость изменений и предсказуемое восстановление после инцидентов.

## Что дает компании

- Снижение риска инцидентов за счет ранней фильтрации нежелательного трафика.
- Контролируемое внедрение изменений через ревизии и централизованный audit trail.
- Единая эксплуатационная модель для WAF, TLS, политик доступа, rate-limit и Anti-DDoS.

## Функциональные возможности

- WAF/CRS инспекция и политики доступа.
- L4/L7 Anti-DDoS с настройками на уровне сайта и глобальными параметрами.
- Управление сайтами, upstream, сертификатами и TLS-конфигурациями.
- События, запросы, аудит и отчеты по ревизиям.
- UI + API + CLI (`waf-cli`) для операторских сценариев.

## Безопасность по дизайну

- Server-side zero-trust проверки прав на каждом endpoint.
- Session auth, поддержка 2FA (TOTP) и passkeys (WebAuthn).
- Self-hosted модель: данные и артефакты остаются в вашем контуре.
- Продакшн-базлайн: используйте недефолтные секреты, HTTPS и ограничение trusted proxies.

## Технический профиль

- Backend: Go.
- Runtime: NGINX + ModSecurity + OWASP CRS.
- Storage: PostgreSQL.
- Deployment: Docker / Docker Compose.

## Документация

- Общий индекс: [`docs/README.md`](docs/README.md)
- Русская документация: [`docs/ru/README.md`](docs/ru/README.md)
- English documentation: [`docs/eng/README.md`](docs/eng/README.md)
- Команды CLI: [`docs/CLI_COMMANDS.md`](docs/CLI_COMMANDS.md)

## Запуск и эксплуатация

- AIO one-command старт:
  - `curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh`
- Образ проекта:
  - `docker pull ghcr.io/berkutsolutions/tarinio:latest`
- Деплой: [`docs/ru/deploy.md`](docs/ru/deploy.md)
- Runbook (диагностика и восстановление): [`docs/ru/runbook.md`](docs/ru/runbook.md)
- Обновление и откат: [`docs/ru/upgrade.md`](docs/ru/upgrade.md)
- Compose профили: [`deploy/compose/README.md`](deploy/compose/README.md)

## Скриншоты

![Скриншот 1](ui/app/static/screen1.png)

![Скриншот 2](ui/app/static/screen2.png)

![Скриншот 3](ui/app/static/screen3.png)



