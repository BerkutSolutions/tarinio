# Compliance Mapping

## Purpose

This page helps security and compliance teams understand how TARINIO v1.3.5 capabilities align with common control themes.

## Important Clarification

This is a functional mapping, not a claim of formal certification.

## OWASP-Oriented Alignment

TARINIO contributes to:

- request inspection and policy enforcement (WAF/CRS, ModSecurity);
- traffic visibility and event correlation;
- safer operational rollout of security changes via revision workflow;
- rate limiting and hostile traffic suppression;
- credential stuffing detection (OWASP ASVS 2.2 — monitoring 401 responses on auth paths);
- TLS fingerprint blocking via JA3/JA4 blacklist (reducing automated attack surface);
- virtual patches for temporary mitigation of known vulnerabilities;
- WebSocket traffic inspection;
- HTTP Request Smuggling protection via strict parsing;
- mutual TLS authentication (incoming and outgoing mTLS).

## Operational Security Alignment

TARINIO helps teams support controls around:

- least-privilege administration;
- auditability of changes (revision log, apply history);
- backup and recovery planning;
- protected administrative access;
- observable security operations;
- secret management via Vault (mTLS certs, logging credentials);
- geo-based access policies with time windows (Geo Time Windows).

## Governance Value

TARINIO documentation and product workflows are especially useful where teams need:

- evidence of controlled change management;
- evidence of rollback readiness;
- evidence of monitored ingress protections;
- evidence of operator runbooks and recovery procedures.
