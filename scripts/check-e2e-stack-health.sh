#!/usr/bin/env sh
# Validates a disposable E2E Compose stack, including writable sentinel volumes.
set -eu

COMPOSE_CMD="${COMPOSE_CMD:-docker compose}"
PROFILE_DIR="${PROFILE_DIR:-$(CDPATH= cd -- "$(dirname -- "$0")/../deploy/compose/e2e" && pwd)}"
TIMEOUT_SECONDS="${E2E_HEALTH_TIMEOUT_SECONDS:-180}"

[ -n "${E2E_PASS:-${WAF_E2E_PASSWORD:-}}" ] || {
  echo "[e2e-health] E2E_PASS or WAF_E2E_PASSWORD is required" >&2
  exit 1
}
export E2E_PASS="${E2E_PASS:-$WAF_E2E_PASSWORD}"

cd "$PROFILE_DIR"

compose() {
  # shellcheck disable=SC2086
  $COMPOSE_CMD -f docker-compose.yml "$@"
}

cleanup() {
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
compose up -d --build

wait_for() {
  label="$1"
  shift
  started="$(date +%s)"
  until "$@"; do
    if [ $(( $(date +%s) - started )) -ge "$TIMEOUT_SECONDS" ]; then
      echo "[e2e-health] timeout: $label" >&2
      return 1
    fi
    sleep 2
  done
  echo "[e2e-health] ok: $label"
}

wait_for "control-plane healthz" compose exec -T control-plane sh -ec 'wget -qO- http://127.0.0.1:8080/healthz >/dev/null'
wait_for "runtime healthz" compose exec -T runtime sh -ec 'wget -qO- http://127.0.0.1:8081/healthz >/dev/null'
wait_for "sentinel adaptive output" compose exec -T tarinio-sentinel sh -ec 'test -s /out/adaptive.json'
wait_for "sentinel suggestions output" compose exec -T tarinio-sentinel sh -ec 'test -s /out/l7-suggestions.json'
compose exec -T tarinio-sentinel sh -ec 'test -w /state && test -w /out && state_probe=/state/.e2e-health-write-probe && out_probe=/out/.e2e-health-write-probe && : >"$state_probe" && : >"$out_probe" && rm -f "$state_probe" "$out_probe"'

if compose logs --no-color tarinio-sentinel | grep -Ei 'permission denied|save (state|adaptive|suggestions) output failed'; then
  echo "[e2e-health] sentinel write failure found in logs" >&2
  exit 1
fi

wait_for "UI healthcheck page" compose exec -T ui sh -ec 'wget -qO- http://127.0.0.1/healthcheck >/dev/null'
echo "[e2e-health] disposable E2E Compose stack is healthy"
