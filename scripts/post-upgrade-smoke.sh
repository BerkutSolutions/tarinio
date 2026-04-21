#!/usr/bin/env sh
set -eu

COMPOSE_CMD="${COMPOSE_CMD:-docker compose}"
PROFILE_DIR="${PROFILE_DIR:-$(pwd)}"
CONTROL_PLANE_SERVICE="${CONTROL_PLANE_SERVICE:-control-plane}"
RUNTIME_SERVICE="${RUNTIME_SERVICE:-runtime}"
HOST_BASE_URL="${HOST_BASE_URL:-http://127.0.0.1:8080}"
RUNTIME_API_TOKEN="${WAF_RUNTIME_API_TOKEN:-}"
CONTROL_PLANE_METRICS_TOKEN="${CONTROL_PLANE_METRICS_TOKEN:-}"
RUNTIME_METRICS_TOKEN="${WAF_RUNTIME_METRICS_TOKEN:-}"

cd "$PROFILE_DIR"

probe_exec() {
  # shellcheck disable=SC2086
  $COMPOSE_CMD -f docker-compose.yml exec -T "$1" sh -lc "$2"
}

probe_host() {
  url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS "$url" >/dev/null
    return
  fi
  wget -qO- "$url" >/dev/null
}

echo "Strict post-upgrade smoke: control-plane health"
probe_exec "$CONTROL_PLANE_SERVICE" "wget -qO- http://127.0.0.1:8080/healthz >/dev/null"

echo "Strict post-upgrade smoke: setup status"
probe_exec "$CONTROL_PLANE_SERVICE" "wget -qO- http://127.0.0.1:8080/api/setup/status >/dev/null"

echo "Strict post-upgrade smoke: runtime health"
probe_exec "$RUNTIME_SERVICE" "wget -qO- http://127.0.0.1:8081/healthz >/dev/null"

if [ -n "$RUNTIME_API_TOKEN" ]; then
  echo "Strict post-upgrade smoke: runtime ready"
  probe_exec "$RUNTIME_SERVICE" "wget -qO- --header='X-WAF-Runtime-Token: $RUNTIME_API_TOKEN' http://127.0.0.1:8081/readyz >/dev/null"
fi

echo "Strict post-upgrade smoke: host healthcheck"
probe_host "$HOST_BASE_URL/healthcheck"

if [ -n "$CONTROL_PLANE_METRICS_TOKEN" ]; then
  echo "Strict post-upgrade smoke: control-plane metrics"
  probe_exec "$CONTROL_PLANE_SERVICE" "wget -qO- 'http://127.0.0.1:8080/metrics?token=$CONTROL_PLANE_METRICS_TOKEN' >/dev/null"
fi

if [ -n "$RUNTIME_METRICS_TOKEN" ]; then
  echo "Strict post-upgrade smoke: runtime metrics"
  probe_exec "$RUNTIME_SERVICE" "wget -qO- 'http://127.0.0.1:8081/metrics?token=$RUNTIME_METRICS_TOKEN' >/dev/null"
fi

echo "Strict post-upgrade smoke passed."
