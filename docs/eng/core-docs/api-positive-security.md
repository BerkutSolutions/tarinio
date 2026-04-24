# API Positive Security

API Positive Security in TARINIO enforces "allow only what is explicitly expected" for API traffic.

## Scope

- OpenAPI contract reference per service (`openapi_schema_ref`).
- Enforcement mode (`monitor` or `block`).
- Default action (`allow` or `deny`) for unknown endpoints.
- Endpoint policies with per-endpoint and per-token controls.

## Endpoint Policy Model

Each endpoint policy supports:

- `path`
- `methods`
- `token_ids` (from `X-WAF-API-TOKEN-ID`)
- `content_types`
- `mode` (`monitor` or `block`)

## Easy Profile Fields

`security_api_positive` object:

- `use_api_positive_security`
- `openapi_schema_ref`
- `enforcement_mode`
- `default_action`
- `endpoint_policies[]`

## Rollout Guidance

1. Start with `monitor` mode.
2. Validate logs and false positives on real traffic.
3. Switch selected endpoints/policies to `block`.
4. Enable `default_action=deny` only after endpoint inventory is complete.
