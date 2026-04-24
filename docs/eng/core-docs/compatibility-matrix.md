# Compatibility Matrix

This page belongs to the current documentation branch.

## Scope

This matrix defines the deployment combinations TARINIO `3.0.2` is designed, tested, and supported for.

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

- stay close to documented major versions unless newer lines are validated in lab first.

## Deployment Modes

Documented and supported patterns:

- single-node compose deployment;
- PostgreSQL-backed default deployment;
- High Availability control-plane deployment with shared PostgreSQL and Redis;
- observability-enabled High Availability lab;
- AIO-driven upgrades.

## Supported Version Channels

- `Current`: full functional and security support.
- `Stable`: bugfix and security support, no feature expansion.
- `LTS 2.0`: security and critical resilience support until April 30, 2027.

## Browser Support

The operator UI targets evergreen browsers:

- current Chrome / Chromium;
- current Microsoft Edge;
- current Firefox.

## Not Covered By This Matrix

- arbitrary Kubernetes distributions without separate validation;
- unsupported database major versions;
- custom runtime image mutations outside documented deployment paths.

## Related Documents

- [Deployment](core-docs/deploy.md)
- [Support and Lifecycle Policy](core-docs/support-lifecycle.md)
- [Sizing Guide](core-docs/sizing.md)
- [Reference Architectures](architecture-docs/reference-architectures.md)




