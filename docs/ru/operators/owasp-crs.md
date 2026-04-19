# Эксплуатация OWASP CRS

Дата: `2026-04-04`

Документ описывает, как TARINIO управляет версиями OWASP CRS и поведением CRS на уровне сервисов.

## Модель обновления

Поведение runtime:
- при первом старте runtime может один раз подтянуть последнюю версию CRS;
- после этого CRS не обновляется “тихо” просто от открытия страницы;
- дальнейшие обновления выполняются оператором вручную или по расписанию.

Опциональный режим:
- оператор может включить почасовую автопроверку и автообновление;
- если режим выключен, runtime остаётся на текущей активной версии CRS.

## API и управление

Основные endpoints:
- `GET /api/owasp-crs/status`
- `POST /api/owasp-crs/check-updates`
- `POST /api/owasp-crs/update`

Особенности:
- `check-updates` неразрушающий, может использоваться как dry-run;
- `update` подтягивает актуальный релиз и инициирует runtime reload;
- `update` также принимает переключение hourly auto-update.

## Поведение на уровне сервиса

Easy profile управляет CRS через:
- `use_modsecurity`
- `use_modsecurity_crs_plugins`
- `use_modsecurity_custom_configuration`
- `custom_configuration.path`
- `custom_configuration.content`

Значения по умолчанию:
- если ModSecurity включён, CRS обычно тоже включён;
- custom configuration подключается только по явному флагу.

Режимы:
- `block` -> `SecRuleEngine On`
- `monitor` -> `SecRuleEngine DetectionOnly`
- `transparent` -> `SecRuleEngine Off`

## Проверка после обновления

Используйте:
- `deploy/compose/default/test-xss.ps1`

Сценарий проверки должен подтвердить:
- что CRS действительно загружен;
- что XSS smoke-пробы отрабатывают корректно;
- что блокировки не стали массово ложноположительными.
