# Enterprise-валидация Anti-Bot

Последний полный прогон профиля: 2026-04-24.

Документ фиксирует выделенный enterprise-срез валидации поведения anti-bot challenge в TARINIO `current release`.

## Цель проверки

Подтвердить, что anti-bot challenge-контур:

1. Срабатывает на запросы без verification-cookie.
2. Корректно выдает и повторно использует anti-bot cookie-состояние.
3. Разделяет human-like и bot-like трафик по фактическому поведению клиента.

## Профили трафика (5 акторов)

- `human-regular`
- `human-hacker`
- `bot-curl`
- `bot-python-requests`
- `bot-scanner-direct-verify-no-jar`

## Критерии приемки

1. Первый запрос без anti-bot cookie перенаправляется на challenge.
2. Human-like профиль после `/challenge/verify` и сохранения cookie проходит без повторного challenge.
3. Bot-like профиль без сохранения cookie остается в challenge-петле.
4. Verification-cookie сохраняет стабильность между последовательными запросами human-профиля.

## Что проверяет тест

Тест `ui/tests/e2e_antibot_profiles_test.go` валидирует:

- challenge-redirect для нового клиента;
- доступность challenge-страницы;
- redirect-цепочку `/challenge/verify`;
- поведение после verify на исходно запрошенном URL;
- наличие cookie с префиксом `waf_antibot_` в сценариях с persistent jar.

## Итоговый вывод

Anti-bot слой проходит enterprise-проверку на разделение human/bot профилей:

- human-трафик после verify стабильно возвращается к рабочему маршруту;
- bot-трафик без сохранения cookie не получает устойчивого обхода;
- challenge-контракт и cookie-persistence ведут себя предсказуемо.

Это подтверждает production-like пригодность anti-bot контроля в составе TARINIO `current release`.
