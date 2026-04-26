# Enterprise-проверка защиты сервиса

Последний полный smoke-прогон: 2026-04-25 13:22:32 +03:00

Документ фиксирует воспроизводимую проверку полного защитного контура TARINIO 3.0.6.

## Область проверки

Прогон проверяет единую цепочку защиты:

1. Жесткие правила доступа (allowlist, denylist, страны) отрабатывают первыми и возвращают 403.
2. В Anti-Bot присутствует и включен по умолчанию автобан сканеров.
3. Sentinel / адаптивная Anti-DDoS модель продолжает выявлять и эскалировать всплески.
4. Нормальный и доверенный трафик не получает ложных адаптивных блокировок.
5. Для ha-lab используется синтетическая нагрузка x3, чтобы различия между профилями были заметны.

## Матрица 10 клиентов

| Роль клиента | Паттерн | Ожидаемый результат |
| --- | --- | --- |
| normal_web | обычные браузерные запросы | без adaptive-блокировок |
| normal_mobile | обычные мобильные запросы | без adaptive-блокировок |
| trusted_allowlist | доверенный источник | без adaptive-блокировок |
| api_client | высокочастотный API burst | сигнал emergency single-source |
| country_blocked | трафик из блокируемой страны | немедленные 403 |
| denylisted_ip | трафик denylist-источника | немедленные 403 |
| scanner_a | сигнатуры сканера | scanner suggestions / adaptive-pressure |
| scanner_b | сигнатуры сканера | scanner suggestions / adaptive-pressure |
| hacker_payload | payload-пробы (xss/sqli/rce) | adaptive-записи атакующего источника |
| botnet_group | распределенный флуд | сигнал emergency botnet |

## Итог

| Профиль | Масштаб нагрузки | Passed | Guards verified | Policy 403 evidence | Events | Adaptive entries | Actions | Scanner clients | Botnet unique IPs | Emergency single-source | Emergency botnet | Botnet adaptive entries | Сумма false positive по нормальным клиентам |
| --- | --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| default | x1 | True | False | True | 1200 | 145 | drop: 145 | 4 | 480 | 0 | 144 | 144 | 0 |
| ha-lab | x3 | True | False | True | 3600 | 1000 | drop: 1000 | 8 | 1440 | 0 | 0 | 999 | 0 |

## Проверки runtime-конфига

Критерии прохождения включают:

- country guard расположен раньше antibot guard;
- присутствует guard для allowlist;
- присутствует scanner autoban guard.

## Артефакты

Артефакты прогона сохраняются в:

.work/service-protection-smoke/20260425-132042

По каждому профилю:

- result.json
- adaptive.json
- l7-suggestions.json
- model-state.json
- report.md

## Как повторить

PowerShell:

./scripts/smoke-service-protection-enterprise.ps1

Запуск одного профиля:

./scripts/smoke-service-protection.ps1 -ComposeFile deploy/compose/default/docker-compose.yml -ProfileName default -OutputDir .work/service-protection-smoke/manual/default
./scripts/smoke-service-protection.ps1 -ComposeFile deploy/compose/ha-lab/docker-compose.yml -ProfileName ha-lab -OutputDir .work/service-protection-smoke/manual/ha-lab
