# Deploy (EN)

Documentation baseline: `1.0.4`

## AIO Quick Start (one command)

You can start the full TARINIO stack (auto-start profile) with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/BerkutSolutions/tarinio/main/scripts/install-aio.sh | sh
```

By default, the script:
- clones/updates the repo into `/opt/tarinio`
- uses branch `main`
- starts `deploy/compose/auto-start`

After startup:
- `http://<server-ip>/login`
- `https://<server-ip>/login`

## Manual Docker Compose run

1. Review `.env.example` and create `.env`.
2. Start the profile:

```bash
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
```

## Notes

- Use `auto-start` for quick local bootstrap.
- Use `default` for a production-oriented local run (`ui` on `80`, `runtime` on `443`).
- For production, use non-default secrets and HTTPS.

