# API Positive Security

API Positive Security в TARINIO реализует принцип "разрешено только то, что явно ожидается" для API-трафика.

## Контур

- ссылка на OpenAPI-контракт сервиса (`openapi_schema_ref`);
- режим проверки (`monitor` или `block`);
- действие по умолчанию (`allow` или `deny`) для неизвестных endpoint;
- endpoint-политики с per-endpoint/per-token контролем.

## Модель endpoint-политик

Каждая политика endpoint поддерживает:

- `path`
- `methods`
- `token_ids` (заголовок `X-WAF-API-TOKEN-ID`)
- `content_types`
- `mode` (`monitor` или `block`)

## Поля easy-profile

Объект `security_api_positive`:

- `use_api_positive_security`
- `openapi_schema_ref`
- `enforcement_mode`
- `default_action`
- `endpoint_policies[]`

## Рекомендации по внедрению

1. Начинать с `monitor`.
2. Проверить логи и ложные срабатывания на реальном трафике.
3. Перевести критичные endpoint в `block`.
4. Включать `default_action=deny` только после полной инвентаризации endpoint.
