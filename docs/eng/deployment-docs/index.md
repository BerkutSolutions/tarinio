---
sidebar_position: 1
---

# Deployment Documents

This section is a production-oriented deployment cheatsheet for TARINIO v1.3.5+.

## Included Documents

- [Docker Deployment](docker.md)
- [Kubernetes Deployment](kubernetes.md)
- [Terraform Deployment](terraform.md)

## Section Principles

- production-first guidance (not lab-only);
- dual-shell command examples: `sh` and `PowerShell`;
- concise operator-oriented runbook format.

## What's New In v1.3.x

- `vault` is now a mandatory stack component — stores mTLS certificates;
- `tarinio-sentinel` service added (DDoS model, JA3, credential stuffing detection);
- runtime requires `NET_ADMIN` + `NET_RAW` capabilities;
- new volumes: `waf-certificates-data`, `waf-l4-adaptive`, `waf-sentinel-state`;
- new site profile fields: `mtls_*`, `upstream_mtls_*`, `security_websocket`, `geo_time_windows`, `virtual_patches`, `blacklist_ja3`, `http_strict_parsing`, `health_check_*`.
