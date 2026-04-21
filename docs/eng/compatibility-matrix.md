# Compatibility Matrix

This page belongs to the current documentation branch.

## Scope

This matrix describes the deployment combinations TARINIO is designed and tested around.

## Runtime Platform

Supported baseline:

- Linux hosts for production deployment;
- Docker Engine with Docker Compose plugin;
- Debian-based container images for `control-plane` and bundled PostgreSQL;
- NGINX + ModSecurity runtime.

## Data Services

Supported product-aligned services:

- PostgreSQL `15`
- Redis `7`

Recommended rule:

- keep close to the documented major versions unless you validate newer ones in lab first.

## Deployment Modes

Documented and supported patterns:

- single-node compose deployment;
- PostgreSQL-backed default deployment;
- HA control-plane deployment with shared PostgreSQL and Redis;
- observability-enabled HA lab;
- AIO-driven upgrades.

## Related Documents

- [Deployment](deploy.md)
- [Sizing Guide](sizing.md)
- [Reference Architectures](reference-architectures.md)

## Browser Support

The operator UI is intended for current evergreen browsers:

- current Chrome / Chromium;
- current Microsoft Edge;
- current Firefox.

## Environment Assumptions

Expected production conditions:

- persistent volumes for data services and runtime artifacts;
- explicit TLS and secret management;
- stable internal networking between control-plane, runtime, PostgreSQL, and Redis.

## Not A Compatibility Promise For

This matrix should not be read as a support promise for:

- arbitrary Kubernetes distributions;
- unsupported database majors;
- non-Docker local emulation layers;
- heavily modified custom runtime images.
