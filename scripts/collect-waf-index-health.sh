#!/usr/bin/env bash

if [[ -z "${BASH_VERSION:-}" ]]; then
  if command -v bash >/dev/null 2>&1; then
    exec bash "$0" "$@"
  fi
  echo "This script requires bash."
  exit 1
fi

set -u

DEPLOY_DIR="${DEPLOY_DIR:-/opt/tarinio/deploy/compose/default}"
RUNTIME_CONTAINER="${RUNTIME_CONTAINER:-tarinio-runtime}"
CONTROL_PLANE_CONTAINER="${CONTROL_PLANE_CONTAINER:-tarinio-control-plane}"
CLICKHOUSE_CONTAINER="${CLICKHOUSE_CONTAINER:-tarinio-clickhouse}"
OPENSEARCH_CONTAINER="${OPENSEARCH_CONTAINER:-tarinio-opensearch}"
WAF_CLI_BIN="${WAF_CLI_BIN:-waf-cli}"
SINCE="${SINCE:-24h}"
NON_INTERACTIVE="${NON_INTERACTIVE:-0}"

prompt_value() {
  local var_name="$1"
  local prompt_text="$2"
  local current_value="${!var_name:-}"
  if [[ -n "$current_value" || "$NON_INTERACTIVE" == "1" ]]; then
    return
  fi
  read -r -p "$prompt_text" "$var_name"
}

prompt_value SINCE "Collect diagnostics since (example: 30m, 6h, 24h): "

if [[ -z "${SINCE:-}" ]]; then
  echo "SINCE must not be empty."
  exit 1
fi

if command -v "$WAF_CLI_BIN" >/dev/null 2>&1; then
  CLI_MODE="local"
elif command -v docker-compose >/dev/null 2>&1; then
  CLI_MODE="compose"
  COMPOSE_BIN="docker-compose"
else
  CLI_MODE="compose"
  COMPOSE_BIN="docker compose"
fi

TS="$(date +%Y%m%d_%H%M%S)"
OUT_BASE_DIR="${OUT_BASE_DIR:-/tmp}"
OUT="${OUT_BASE_DIR%/}/waf-index-health-${TS}"
mkdir -p "$OUT"

run() {
  local name="$1"
  shift
  {
    echo "# cmd: $*"
    "$@"
    rc=$?
    echo
    echo "# exit_code: $rc"
  } >"$OUT/$name" 2>&1 || true
}

run_sh() {
  local name="$1"
  local cmd="$2"
  {
    echo "# cmd: $cmd"
    bash -lc "$cmd"
    rc=$?
    echo
    echo "# exit_code: $rc"
  } >"$OUT/$name" 2>&1 || true
}

run_cli() {
  local name="$1"
  shift
  if [[ "$CLI_MODE" == "local" ]]; then
    run "$name" "$WAF_CLI_BIN" --no-auth "$@"
    return
  fi
  run "$name" $COMPOSE_BIN --profile tools run --rm cli --no-auth "$@"
}

if [[ "$CLI_MODE" == "compose" && ! -d "$DEPLOY_DIR" ]]; then
  echo "DEPLOY_DIR not found: $DEPLOY_DIR"
  exit 1
fi
if [[ "$CLI_MODE" == "compose" ]]; then
  cd "$DEPLOY_DIR" || exit 1
fi

run_cli "app_meta.json" --json api GET /api/app/meta
run_cli "runtime_settings.json" --json api GET /api/settings/runtime
run_cli "requests_api_sample.json" --json api GET /api/requests?limit=5
run_cli "events_api_sample.json" --json api GET /api/events?limit=5
run "runtime_logs_since.log" docker logs --since="$SINCE" "$RUNTIME_CONTAINER"
run "control_plane_logs_since.log" docker logs --since="$SINCE" "$CONTROL_PLANE_CONTAINER"
run "opensearch_logs_since.log" docker logs --since="$SINCE" "$OPENSEARCH_CONTAINER"
run "clickhouse_logs_since.log" docker logs --since="$SINCE" "$CLICKHOUSE_CONTAINER"
run_sh "runtime_request_indexes.json" "docker exec '$RUNTIME_CONTAINER' sh -lc \"wget --header='X-WAF-Runtime-Token: ${WAF_RUNTIME_API_TOKEN:-default-runtime-shared-token}' -qO- 'http://127.0.0.1:8081/requests/indexes?limit=50&offset=0'\""
run_sh "runtime_request_probe_today.json" "docker exec '$RUNTIME_CONTAINER' sh -lc \"wget --header='X-WAF-Runtime-Token: ${WAF_RUNTIME_API_TOKEN:-default-runtime-shared-token}' -qO- 'http://127.0.0.1:8081/requests/probe?probe=1'\""
run_sh "runtime_request_probe_yesterday.json" "docker exec '$RUNTIME_CONTAINER' sh -lc \"wget --header='X-WAF-Runtime-Token: ${WAF_RUNTIME_API_TOKEN:-default-runtime-shared-token}' -qO- 'http://127.0.0.1:8081/requests/probe?probe=1&day=$(date -u -d '1 day ago' +%F 2>/dev/null || date -u -v-1d +%F 2>/dev/null || printf '')'\""
run_sh "runtime_access_tail.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"tail -n 8000 /var/log/nginx/access.log\""
run_sh "runtime_nginx_conf_requests.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"nginx -T 2>/tmp/nginxT.txt; grep -nE 'requests/indexes|requests/probe|error_page 504|proxy_intercept_errors|fallback' /tmp/nginxT.txt\""
run_sh "opensearch_health.txt" "timeout 15 docker exec '$OPENSEARCH_CONTAINER' bash -lc \"timeout 8 bash -c 'exec 3<>/dev/tcp/127.0.0.1/9200; printf \\\"GET /_cluster/health HTTP/1.0\\r\\nHost: 127.0.0.1\\r\\nConnection: close\\r\\n\\r\\n\\\" >&3; timeout 8 cat <&3'\""
run_sh "opensearch_indices.txt" "timeout 15 docker exec '$OPENSEARCH_CONTAINER' bash -lc \"timeout 8 bash -c 'exec 3<>/dev/tcp/127.0.0.1/9200; printf \\\"GET /_cat/indices?v HTTP/1.0\\r\\nHost: 127.0.0.1\\r\\nConnection: close\\r\\n\\r\\n\\\" >&3; timeout 8 cat <&3'\""
run_sh "opensearch_aliases.txt" "timeout 15 docker exec '$OPENSEARCH_CONTAINER' bash -lc \"timeout 8 bash -c 'exec 3<>/dev/tcp/127.0.0.1/9200; printf \\\"GET /_cat/aliases?v HTTP/1.0\\r\\nHost: 127.0.0.1\\r\\nConnection: close\\r\\n\\r\\n\\\" >&3; timeout 8 cat <&3'\""
run_sh "clickhouse_ping.txt" "docker exec '$CLICKHOUSE_CONTAINER' sh -lc \"wget -qO- http://127.0.0.1:8123/ping\""
run_sh "clickhouse_tables.tsv" "docker exec '$CLICKHOUSE_CONTAINER' sh -lc \"clickhouse-client --user \\\"\\${CLICKHOUSE_USER:-default}\\\" --password \\\"\\${CLICKHOUSE_PASSWORD:-}\\\" --query \\\"SELECT database, name, total_rows, total_bytes FROM system.tables WHERE database NOT IN ('system','information_schema','INFORMATION_SCHEMA') FORMAT TSVWithNames\\\" 2>/dev/null || true\""
run_sh "clickhouse_request_logs_count.txt" "docker exec '$CLICKHOUSE_CONTAINER' sh -lc \"clickhouse-client --user \\\"\\${CLICKHOUSE_USER:-default}\\\" --password \\\"\\${CLICKHOUSE_PASSWORD:-}\\\" --query \\\"SELECT count() FROM waf_logs.request_logs\\\" 2>/dev/null || true\""

{
  echo "# index health summary"
  echo "since=$SINCE"
  echo
  echo "## runtime /requests/indexes"
  grep -nE '\"date\"|\"file\"|\"lines\"|\"size_bytes\"|\"updated_at\"|\"error\"' "$OUT/runtime_request_indexes.json" || true
  echo
  echo "## runtime /requests probe"
  grep -nE 'HTTP/|\"ok\"|\"error\"|\"status\"' "$OUT/runtime_request_probe_today.json" "$OUT/runtime_request_probe_yesterday.json" || true
  echo
  echo "## request API sample"
  grep -nE 'HTTP/|\"host\"|\"uri\"|\"status\"|\"site_id\"' "$OUT/requests_api_sample.json" | head -n 40 || true
  echo
  echo "## event API sample"
  grep -nE 'HTTP/|\"site_id\"|\"kind\"|\"status\"|\"action\"' "$OUT/events_api_sample.json" | head -n 40 || true
  echo
  echo "## runtime/control-plane log focus"
  grep -nEi 'requests/indexes|requests/probe|request archive|opensearch|clickhouse|timeout|timed out|deadline exceeded|fallback|504' "$OUT/runtime_logs_since.log" "$OUT/control_plane_logs_since.log" || true
  echo
  echo "## storage backend log focus"
  grep -nEi 'error|warn|timeout|timed out|read-only|disk|shard|mapping|merge|exception' "$OUT/opensearch_logs_since.log" "$OUT/clickhouse_logs_since.log" || true
  echo
  echo "## access log focus for dashboard and request endpoints"
  grep -nEi '\"uri\":\"/api/(dashboard/stats|requests|events)\"|\"status\":504|\"status\":502' "$OUT/runtime_access_tail.log" || true
} >"$OUT/index_health_summary.txt" 2>&1

tar -C "$OUT_BASE_DIR" -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
