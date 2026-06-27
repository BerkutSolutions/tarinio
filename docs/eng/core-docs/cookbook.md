# Operations Cookbook

## Purpose

This cookbook groups common operator tasks into short, repeatable sequences.

## Add A New Site

1. Create the site.
2. Create the upstream.
3. Bind TLS if needed.
4. Compile a revision.
5. Apply the revision.
6. Validate through `/healthcheck` and `Requests`.

## Roll Out A Large Batch Safely

1. Persist changes with `X-WAF-Auto-Apply-Disabled: true`.
2. Review the intended state.
3. Compile once.
4. Apply once.
5. Validate through `Dashboard`, `Events`, and metrics.

## Tighten CRS Carefully

1. Record the current revision.
2. Change the CRS-related policy.
3. Compile and apply.
4. Watch for false positives in `Requests` and `Events`.
5. Add exclusions only where justified.

## Investigate A Blocked Request

1. Find the request in `Requests`.
2. Correlate it with `Events`.
3. Determine whether WAF, rate-limit, or Anti-DDoS blocked it.
4. Decide whether to tune the policy or keep the block.

## Perform A Rolling Upgrade

1. Back up before upgrade.
2. Validate High Availability health.
3. Upgrade one control-plane node at a time.
4. Probe API availability during the upgrade.
5. Run strict post-upgrade smoke validation.

## Enable WebSocket Inspection

1. Open the site profile → `WebSocket` tab.
2. Enable `use_websocket_inspection`.
3. Set frame/message size limits (`max_frame_size`, `max_message_size`) for the expected traffic.
4. Add blocked patterns if needed.
5. Compile and apply the revision.
6. Monitor `Events` for WebSocket-related hits.

## Enable mTLS For A Site

Incoming mTLS (client → WAF):

1. Open the site profile → `mTLS` tab.
2. Enable `use_mtls`.
3. Set `mtls_vault_path` to the ClientCA path in Vault.
4. Enable `mtls_require_client_cert` if strict enforcement is needed.
5. Compile and apply the revision.
6. Verify that connections without a client certificate are rejected.

Outgoing mTLS (WAF → upstream):

1. Enable `use_upstream_mtls`.
2. Set `upstream_mtls_cert_vault_path` and `upstream_mtls_key_vault_path`.
3. Compile and apply the revision.
4. Confirm the upstream accepts the WAF client certificate.

## Enable Virtual Patching

1. Open the site profile → `Virtual Patches` section.
2. Add a SecRule in the `rule` field and a description in `description`.
3. Verify the rule is syntactically valid (ModSecurity SecRule syntax).
4. Compile and apply the revision.
5. Check `Events` to confirm the rule fires on expected requests.

## Configure Geo Time Windows

1. Open the site profile → `Geo Time Windows` tab.
2. Add an entry: country code (`country_code`), time range, action (`allow`/`deny`).
3. Compile and apply the revision.
4. Check `Requests` to confirm traffic from the specified regions is handled according to policy.

## Configure Credential Stuffing Detection

1. Ensure the site has `auth_endpoint_path` configured.
2. Set `auth_failure_threshold` — the number of 401 responses after which the IP is blocked.
3. Compile and apply the revision.
4. Monitor `auth_failures` signals in Sentinel and `Events`.
