# Supported Profile Criteria (Lab)

This page summarizes when `deploy/lab-k8s-terraform` can move from `experimental/lab` to `supported`.

## Required Gates

1. Stable CI cycles (`apply -> smoke -> teardown`) without manual fixes.
2. HA profile smoke stability (`ha-control-plane`).
3. Security baseline in place (`guardrails`, non-placeholder secrets, rotation procedure).
4. Documented upgrade, rollback, and troubleshooting paths.
5. Version/changelog synchronization for the active release cycle.

Source of truth:

- `.work/SUPPORTED_PROFILE_CRITERIA.md`
