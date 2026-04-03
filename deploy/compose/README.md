# Compose Profiles

This directory contains two isolated compose profiles:

- `default/` - main local stack
- `testpage/` - one-command profile with two WAF HTTPS entries:
  - `https://localhost:8080` -> WAF/UI
  - `https://localhost:8081` -> protected test page

Run from repository root:

```powershell
docker compose -f deploy/compose/default/docker-compose.yml up -d --build
cd deploy/compose/testpage
docker compose up -d --build
```

Run from `deploy/compose` directory:

```powershell
cd deploy/compose
docker compose -f .\default\docker-compose.yml up -d --build
cd .\testpage
docker compose up -d --build
```

Profile docs:

- `deploy/compose/default/README.md`
- `deploy/compose/testpage/README.md`
