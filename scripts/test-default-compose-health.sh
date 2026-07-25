#!/usr/bin/env sh
# Validates default Compose via a copied, disposable profile. Production resources stay untouched.
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
PROFILE_DIR="${DEFAULT_COMPOSE_DIR:-$ROOT_DIR/deploy/compose/default}"
TMP_DIR="$(mktemp -d)"
PROJECT="default-health-${CI_PIPELINE_ID:-local}-$$"

cleanup() {
  docker compose -p "$PROJECT" -f "$PROFILE_DIR/docker-compose.yml" -f "$TMP_DIR/health-override.yml" down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

cat >"$TMP_DIR/health-override.yml" <<'YAML'
services:
  control-plane:
    container_name: ${WAF_STACK_NAME}-control-plane
  runtime:
    container_name: ${WAF_STACK_NAME}-runtime
    ports: !reset []
  postgres:
    container_name: ${WAF_STACK_NAME}-postgres
  opensearch:
    container_name: ${WAF_STACK_NAME}-opensearch
  vault:
    container_name: ${WAF_STACK_NAME}-vault
  ui:
    container_name: ${WAF_STACK_NAME}-ui
  tarinio-sentinel:
    container_name: ${WAF_STACK_NAME}-sentinel
YAML
COMPOSE="docker compose -p $PROJECT -f $PROFILE_DIR/docker-compose.yml -f $TMP_DIR/health-override.yml"

export WAF_STACK_NAME="$PROJECT"
export WAF_RUNTIME_API_TOKEN="${WAF_RUNTIME_API_TOKEN:-default-health-runtime-token}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-default-health-postgres-password}"
export CONTROL_PLANE_SECURITY_PEPPER="${CONTROL_PLANE_SECURITY_PEPPER:-default-health-security-pepper}"
$COMPOSE config -q
$COMPOSE up -d --build --pull never

wait_for() {
  label="$1"
  shift
  started="$(date +%s)"
  until "$@"; do
    [ $(( $(date +%s) - started )) -lt "${DEFAULT_HEALTH_TIMEOUT_SECONDS:-240}" ] || { echo "[default-health-test] timeout: $label" >&2; exit 1; }
    sleep 2
  done
  echo "[default-health-test] ok: $label"
}

wait_for "control-plane health" $COMPOSE exec -T control-plane sh -ec 'wget -qO- http://127.0.0.1:8080/healthz >/dev/null'
wait_for "runtime health" $COMPOSE exec -T runtime sh -ec 'wget -qO- http://127.0.0.1:8081/healthz >/dev/null'
wait_for "sentinel adaptive output" $COMPOSE exec -T tarinio-sentinel sh -ec 'test -s /out/adaptive.json'
wait_for "sentinel suggestions output" $COMPOSE exec -T tarinio-sentinel sh -ec 'test -s /out/l7-suggestions.json'
wait_for "sentinel unprivileged process" $COMPOSE exec -T tarinio-sentinel sh -ec 'awk "\$1 == \"Uid:\" { exit \$2 == 65532 ? 0 : 1 }" /proc/1/status'
wait_for "sentinel writable volumes" $COMPOSE exec -T --user 65532:4 tarinio-sentinel sh -ec 'test -w /state && test -w /out && : >/state/.health-probe && : >/out/.health-probe && rm -f /state/.health-probe /out/.health-probe'
wait_for "sentinel health" $COMPOSE exec -T tarinio-sentinel sh -ec 'test -s /out/adaptive.json'
if $COMPOSE logs --no-color tarinio-sentinel | grep -Ei 'permission denied|save (state|adaptive|suggestions) output failed'; then
  echo "[default-health-test] sentinel write errors found" >&2
  exit 1
fi
echo "[default-health-test] isolated default profile is healthy"
