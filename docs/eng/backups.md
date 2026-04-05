# Backups (EN)

Documentation baseline: `1.0.13`

## Goal

Define a minimal safe backup path for single-node deployments.

## What to back up

- Database (PostgreSQL volume)
- volumes containing revision artifacts and runtime state (if used)
- `.env` (outside the repo, store securely)

## Important

Backups are required before upgrades (see `docs/eng/upgrade.md`).





