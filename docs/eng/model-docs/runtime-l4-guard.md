# Runtime L4 Guard

This document describes the baseline L4 anti-DDoS layer of the runtime container.

## Purpose

The L4 guard protects runtime before traffic reaches:

- NGINX;
- TLS handshake;
- ModSecurity;
- the HTTP parser and higher-level policies.

This is an infrastructure layer. It does not replace WAF logic and does not depend on request content.

## What The Guard Does

During runtime bootstrap, deterministic `iptables` rules are applied that:

- limit concurrent TCP connections per source IP via `connlimit`;
- limit the rate of new TCP connections via `hashlimit`;
- apply `DROP` or `REJECT` when thresholds are exceeded.

The guard can also read adaptive output and apply temporary per-IP rules:

- `throttle`
- `drop`

## Startup Order

Normal bootstrap flow:

1. `waf-runtime-l4-guard bootstrap`
2. create or update the firewall chain
3. start `waf-runtime-launcher`
4. launcher reads `active/current.json` and starts NGINX

No separate manual step after runtime startup is required.

## Parent Chains

The guard can install its jump rule into:

- `DOCKER-USER`
- `INPUT`

`auto` is the default mode:

- if `DOCKER-USER` is present in the namespace, it is preferred;
- otherwise `INPUT` is used.

`DOCKER-USER` is generally preferable in Docker-aware deployments where traffic reaches runtime through DNAT.

## Docker, DNAT, And Destination IP

In Docker bridge mode, filtering in `DOCKER-USER` should account for the container destination IP after DNAT.

For that purpose:

- `WAF_L4_GUARD_DESTINATION_IP=<runtime-container-ip>`

If the value is not set:

- the guard tries to detect the first non-loopback IPv4 address in the current namespace.

## Protected Ports

Ports are configured through:

- `WAF_L4_GUARD_PORTS`

Default value:

- `80,443`

The list must match the real runtime exposure, otherwise some traffic will remain outside protection.

## Configuration Parameters

- `WAF_L4_GUARD_ENABLED`
  Default: `true`
- `WAF_L4_GUARD_CHAIN_MODE`
  Allowed values: `auto`, `docker-user`, `input`
- `WAF_L4_GUARD_CONN_LIMIT`
  Baseline concurrent TCP connection limit
- `WAF_L4_GUARD_RATE_PER_SECOND`
  Baseline rate of new TCP connections per second
- `WAF_L4_GUARD_RATE_BURST`
  Allowed burst for new TCP connections
- `WAF_L4_GUARD_TARGET`
  `DROP` or `REJECT`
- `WAF_L4_GUARD_DESTINATION_IP`
  Explicit runtime IPv4 address used for `DOCKER-USER`
- `WAF_L4_GUARD_ADAPTIVE_PATH`
  Path to `adaptive.json` with dynamic throttle/drop rules

## Adaptive Integration

The guard reads:

- baseline config from `l4guard/config.json`;
- adaptive output from `l4guard-adaptive/adaptive.json`.

If an adaptive entry is:

- `drop`, a direct per-IP rule is created;
- `throttle`, a dedicated hashlimit rule is created for the IP.

This results in:

- the revision defining the baseline;
- the adaptive model adding temporary escalations;
- the L4 guard performing the actual enforcement.

## Runtime Contract

The guard must not read:

- control-plane storage;
- domain objects;
- revision metadata outside the runtime bundle.

It should depend only on:

- env/runtime config;
- `iptables` availability;
- the active revision artifacts;
- the adaptive output file.

## Required Privileges

Runtime must be able to manage firewall rules in the selected namespace.

Typical requirement:

- `NET_ADMIN`

If `iptables` is unavailable:

- bootstrap should fail;
- runtime should not continue startup as if protection had been applied successfully.
