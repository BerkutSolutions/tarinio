# Secret Management

This document explains how TARINIO `3.0.1` handles secrets, what `Vault` is responsible for, which data is allowed to stay local, and what operators should verify after a fresh install or upgrade.

## Why This Matters

For production and enterprise deployments, two questions always matter:

- where secrets physically live;
- whether unsafe legacy copies remain after a migration.

In `3.0.1`, the product should no longer look like a stack where backend passwords simply live in `.env` and spread across containers without a controlled secret flow.

## Secret Classes

### Infrastructure secrets

Examples:

- `POSTGRES_PASSWORD`;
- `WAF_RUNTIME_API_TOKEN`;
- `CONTROL_PLANE_SECURITY_PEPPER`;
- bootstrap values for `OpenSearch`, `ClickHouse`, and `Vault`.

### Integration secrets

These are backend-specific credentials such as:

- the `ClickHouse` password;
- the `OpenSearch` password;
- an `OpenSearch API key`;
- future external integration tokens.

### Application-entered secrets

These are values saved by an operator through the UI and later used by the product to access external integrations.

## Secret Provider Modes

The logging subsystem supports two real provider modes.

### `encrypted_file`

This is the compatibility fallback mode:

- the secret is stored in persisted runtime settings;
- the value is encrypted before it is written;
- API responses return masked values only;
- the UI never receives the plaintext back.

This is safer than plain local storage, but it is the fallback mode rather than the preferred production posture.

### `vault`

This is the default and preferred `3.0.1` mode:

- backend integration secrets are written to `HashiCorp Vault`;
- runtime resolves them at read time;
- API and UI responses stay masked;
- local persisted settings must not keep active backend credentials in plaintext.

## What Is Stored In Vault

The current logging paths are:

- `secret/<path_prefix>/logging/clickhouse`
- `secret/<path_prefix>/logging/opensearch`

Typical fields include:

- `password`
- `api_key`

If more logging backends are added later, they should join this same managed secret flow instead of creating separate ad hoc secret storage.

## What Still Stays Local

Even in Vault mode, some bootstrap data still has to exist locally, otherwise the standalone stack cannot start.

The acceptable minimum is:

- Vault address;
- mount;
- path prefix;
- Vault bootstrap token;
- technical metadata required to read the secret.

Active backend passwords should not remain as local persistent copies after a successful Vault write.

## How Bootstrap Works In The Default Profile

In `deploy/compose/default`, Vault is no longer just "included in compose". It is prepared for real use automatically.

The normal flow is:

1. the `vault` container starts;
2. Vault is initialized;
3. Vault is unsealed;
4. the `secret/` kv-v2 mount is enabled;
5. a bootstrap token is saved into a dedicated secret volume;
6. `control-plane` and `runtime` receive read-only access to that token file.

This matters because the standalone profile is now operational out of the box instead of requiring a manual init/unseal procedure after first boot.

## Safe Migration Order

The safe order is always:

1. validate Vault access;
2. write the secret to Vault;
3. verify that it can be read back;
4. clear the local persisted backend credential copy;
5. keep API responses masked.

That order prevents the product from entering a half-broken state where the old copy is already gone but the new one is not actually usable.

## What Counts As Unsafe

The following should be treated as unsafe:

- keeping a working backend password in `.env` as the long-term source of truth;
- returning plaintext secrets to the UI;
- leaving old local copies after a successful Vault migration;
- maintaining multiple conflicting sources of truth for the same credential.

## What Counts As Normal In `3.0.1`

The intended release posture is:

- primary secret provider: `Vault`;
- local `encrypted_file`: fallback only;
- backend credentials: in Vault;
- UI: masked values only;
- bootstrap token: a local technical artifact needed to boot the standalone stack.

## How Operators Should Verify Vault

After installation or upgrade, operators should verify that:

1. `vault` is healthy;
2. Vault reports `Initialized=true` and `Sealed=false`;
3. the product stores `vault` as the active secret provider;
4. saving settings does not return plaintext backend credentials in the UI;
5. runtime can resolve the secret from Vault and use the target backend;
6. persisted runtime settings no longer contain the old open local copy.

## What To Expect During Failure

If Vault is unavailable:

- the product should not instantly lose already saved UI settings;
- but writing new secrets and safely operating integrations may degrade;
- operators should restore Vault first and then verify backend access.

If a temporary fallback is needed:

- `encrypted_file` is an acceptable temporary operational mode;
- it should not be treated as the final enterprise production posture.

## Practical Summary

The simple rule for operators is:

- integration secrets should live in `Vault`;
- only minimal bootstrap data should remain local;
- the product must not expose plaintext secrets;
- migration must follow `write -> validate -> clean local copy`.

If those conditions hold, the secret-management model in `3.0.1` is current, predictable, and safe enough for real operation.
