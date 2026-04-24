# Enterprise-проверка Anti-Bot

Последний полный профильный прогон: 2026-04-24.

Документ фиксирует отдельный enterprise-срез для anti-bot challenge в TARINIO `3.0.1`.

## Цель проверки

Подтвердить, что challenge-контур:

1. Срабатывает на запросы без верификационной cookie.
2. Корректно выдаёт и переиспользует anti-bot cookie.
3. Отделяет human-профили от bot-профилей по фактическому поведению клиента.

## Профили трафика (5 акторов)

- `human-regular`
- `human-hacker`
- `bot-curl`
- `bot-python-requests`
- `bot-scanner-direct-verify-no-jar`

## Критерии приёмки

1. Первый запрос без anti-bot cookie уходит в challenge-redirect.
2. Human-профиль после `/challenge/verify` и cookie persistence проходит без повторного challenge.
3. Bot-профиль без cookie persistence остаётся в challenge-петле.
4. Верификационная cookie не теряется между последовательными запросами в human-сценарии.

## Что проверяется тестом

Тест `ui/tests/e2e_antibot_profiles_test.go` валидирует:

- редирект на challenge для нового клиента;
- доступность challenge-страницы;
- корректный `/challenge/verify` redirect;
- поведение после verify (повторный запрос к исходному URL);
- наличие cookie с префиксом `waf_antibot_` в persistent jar.

## Итоговый вывод

Anti-bot механизм проходит enterprise-валидацию по разделению human/bot профилей:

- human-поток после verify стабильно возвращается к рабочему маршруту;
- bot-поток без сохранения cookie не получает устойчивого обхода;
- challenge-контракт и cookie persistence ведут себя предсказуемо.

Это подтверждает пригодность anti-bot слоя для production-like эксплуатации в составе TARINIO `3.0.1`.
