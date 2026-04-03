# Deploy (EN)

Documentation baseline: `1.0.0`

## Docker Compose

Docker Compose is the recommended path for local and single-node deployments.

1. Review `.env.example` and create `.env` (secrets, `TZ`, DB parameters).
2. Start the compose stack from `deploy/compose/`.
3. Open the UI and complete admin bootstrap.

## Notes

- Runtime must stay offline (no CDN/online runtime dependencies;.
- For production, set secrets explicitly and enable HTTPS.
