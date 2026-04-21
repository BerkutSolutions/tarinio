#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

probe_once() {
  docker compose exec -T toolbox sh -lc 'curl -fsS http://api-lb:8080/healthz >/dev/null'
}

wait_healthy() {
  container="$1"
  attempts="${2:-60}"
  i=1
  while [ "$i" -le "$attempts" ]; do
    status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || printf 'missing')"
    if [ "$status" = "healthy" ] || [ "$status" = "running" ]; then
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  echo "container $container did not become healthy" >&2
  exit 1
}

upgrade_one() {
  service="$1"
  container="$2"
  probe_file="$(mktemp)"
  (
    while true; do
      if probe_once; then
        printf 'ok\n' >>"$probe_file"
      else
        printf 'fail\n' >>"$probe_file"
      fi
      sleep 1
    done
  ) &
  probe_pid="$!"

  docker compose up -d --build --no-deps "$service"
  wait_healthy "$container"

  kill "$probe_pid" >/dev/null 2>&1 || true
  wait "$probe_pid" 2>/dev/null || true

  failures="$(grep -c '^fail$' "$probe_file" 2>/dev/null || true)"
  rm -f "$probe_file"
  if [ "${failures:-0}" -gt 0 ]; then
    echo "zero-downtime check failed for $service: $failures API probe failures observed" >&2
    exit 1
  fi
  echo "rolling upgrade step passed: $service"
}

docker compose --profile tools up -d toolbox
wait_healthy tarinio-ha-control-plane-a
wait_healthy tarinio-ha-control-plane-b

upgrade_one control-plane-a tarinio-ha-control-plane-a
upgrade_one control-plane-b tarinio-ha-control-plane-b

echo "Rolling control-plane upgrade passed without API downtime."
