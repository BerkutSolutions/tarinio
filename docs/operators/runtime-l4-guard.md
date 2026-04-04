# Runtime L4 Guard

This document defines the MVP L4 anti-DDoS baseline for the runtime container.

## Purpose

The L4 guard protects the runtime before traffic reaches NGINX, TLS parsing, or ModSecurity processing.

The guard is an infrastructure layer. It is not part of WAF rule logic.

## What It Does

The bootstrap applies deterministic `iptables` rules that:

- limit concurrent TCP connections per source IP with `connlimit`
- limit new TCP connection rate per source IP with `hashlimit`
- drop or reject excess traffic before NGINX handles it

This baseline mitigates high-concurrency floods and high-rate connection floods, including keep-alive abuse where one source tries to pin too many open sockets and force expensive HTTP/TLS work.

## Canonical Bootstrap Flow

The runtime image starts through `runtime/image/entrypoint.sh`.

Bootstrap order:

1. run `waf-runtime-l4-guard bootstrap`
2. apply or refresh the deterministic firewall chain
3. start `waf-runtime-launcher`
4. launcher reads `active/current.json` and starts NGINX

No manual post-start firewall step is required.

## Chain Placement

The guard supports these parent chains:

- `DOCKER-USER`
- `INPUT`

`WAF_L4_GUARD_CHAIN_MODE=auto` is the default.

Behavior:

- if `DOCKER-USER` exists in the current namespace, the guard uses it
- otherwise it falls back to `INPUT`

`DOCKER-USER` is preferred when the runtime firewall is applied from a Docker-aware network namespace and traffic reaches the container through DNAT.

## Docker and DNAT

For Docker bridge networking, filtering in `DOCKER-USER` should target the runtime container destination IP after DNAT.

Use:

- `WAF_L4_GUARD_DESTINATION_IP=<runtime-container-ip>`

When this variable is set, the jump rule in `DOCKER-USER` is bound to that destination IP and the configured protected ports.

If the variable is not set, the guard falls back to the first non-loopback IPv4 address visible in the current namespace.

## Protected Ports

Protected ports are defined by:

- `WAF_L4_GUARD_PORTS`

Default:

- `80,443`

The same canonical ports must be used consistently across publish rules and runtime exposure.

## Environment Variables

- `WAF_L4_GUARD_ENABLED`
  Default: `true`
- `WAF_L4_GUARD_CHAIN_MODE`
  Allowed: `auto`, `docker-user`, `input`
- `WAF_L4_GUARD_CONN_LIMIT`
  Default: `50`
- `WAF_L4_GUARD_RATE_PER_SECOND`
  Default: `30`
- `WAF_L4_GUARD_RATE_BURST`
  Default: `60`
- `WAF_L4_GUARD_PORTS`
  Default: `80,443`
- `WAF_L4_GUARD_TARGET`
  Allowed: `DROP`, `REJECT`
  Default: `DROP`
- `WAF_L4_GUARD_DESTINATION_IP`
  Optional IPv4 destination for `DOCKER-USER` placement

## Runtime Contract

The guard does not read control-plane state, bundle contents, or domain entities.

It depends only on:

- fixed runtime env configuration
- fixed runtime port exposure
- kernel packet filtering availability

It must run before NGINX starts, but it does not change launcher logic or bundle layout.

## Required Privileges

The runtime environment must allow firewall rule management for the selected namespace.

Typical requirement:

- `NET_ADMIN`

If the process cannot manage `iptables`, bootstrap fails and the runtime container does not continue as if protection were active.


