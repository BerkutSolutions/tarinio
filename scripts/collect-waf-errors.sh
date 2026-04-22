#!/usr/bin/env bash

if [[ -z "${BASH_VERSION:-}" ]]; then
  if command -v bash >/dev/null 2>&1; then
    exec bash "$0" "$@"
  fi
  echo "This script requires bash."
  exit 1
fi

set -u

# Interactive/runtime-friendly collector focused on runtime/control-plane errors.
# Usage:
#   bash scripts/collect-waf-errors.sh
# Optional env:
#   SINCE=24h
#   RUNTIME_CONTAINER=tarinio-runtime
#   CONTROL_PLANE_CONTAINER=tarinio-control-plane
#   DDOS_MODEL_CONTAINER=tarinio-ddos-model
#   UI_CONTAINER=tarinio-ui
#   FILTER_HOST=example.com
#   FILTER_IP=203.0.113.10 198.51.100.20

SINCE="${SINCE:-24h}"
RUNTIME_CONTAINER="${RUNTIME_CONTAINER:-tarinio-runtime}"
CONTROL_PLANE_CONTAINER="${CONTROL_PLANE_CONTAINER:-tarinio-control-plane}"
DDOS_MODEL_CONTAINER="${DDOS_MODEL_CONTAINER:-tarinio-ddos-model}"
UI_CONTAINER="${UI_CONTAINER:-tarinio-ui}"
FILTER_HOST="${FILTER_HOST:-}"
FILTER_IP="${FILTER_IP:-}"
FILTER_URI="${FILTER_URI:-}"
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

prompt_value SINCE "Collect logs since (example: 30m, 6h, 24h): "
prompt_value FILTER_HOST "Filter host (optional, example: example.com): "
prompt_value FILTER_IP "Filter client IPs (optional, multiple allowed): "
prompt_value FILTER_URI "Filter request URIs / paths (optional, multiple allowed): "

TS="$(date +%Y%m%d_%H%M%S)"
OUT_BASE_DIR="${OUT_BASE_DIR:-/tmp}"
OUT="${OUT_BASE_DIR%/}/waf-errors-${TS}"
mkdir -p "$OUT"

normalize_multi() {
  printf "%s" "$1" | tr ',;|' '   ' | tr -d "\"'" | awk '{$1=$1; print}'
}

redact_sensitive() {
  printf "%s" "$1"
}

run() {
  local name="$1"
  shift
  local cmd_str="$*"
  cmd_str="$(redact_sensitive "$cmd_str")"
  {
    echo "# cmd: $cmd_str"
    "$@"
    rc=$?
    echo
    echo "# exit_code: $rc"
  } >"$OUT/$name" 2>&1 || true
}

N_FILTER_IP="$(normalize_multi "$FILTER_IP")"
N_FILTER_URI="$(normalize_multi "$FILTER_URI")"

run "runtime_logs_since.log" docker logs --since="$SINCE" "$RUNTIME_CONTAINER"
run "control_plane_logs_since.log" docker logs --since="$SINCE" "$CONTROL_PLANE_CONTAINER"
run "ddos_model_logs_since.log" docker logs --since="$SINCE" "$DDOS_MODEL_CONTAINER"
run "ui_logs_since.log" docker logs --since="$SINCE" "$UI_CONTAINER"
run "runtime_access_recent.log" docker exec "$RUNTIME_CONTAINER" sh -lc "tail -n 12000 /var/log/nginx/access.log"

{
  echo "# runtime focus"
  echo "since=$SINCE"
  echo "filter_host=$FILTER_HOST"
  echo "filter_ip=$N_FILTER_IP"
  echo "filter_uri=$N_FILTER_URI"
  echo
  grep -nEi " \\[error\\] | \\[warn\\] | \\[notice\\] |ModSecurity-nginx|variables_hash|acme-challenge|signal process started|open\\(|failed \\(2: No such file or directory\\)" "$OUT/runtime_logs_since.log" || true
} >"$OUT/runtime_errors_focus.txt" 2>&1

{
  echo "# control-plane focus"
  echo "since=$SINCE"
  echo
  grep -nEi "runtime security collector failed|deadline exceeded|panic|fatal|error|warn" "$OUT/control_plane_logs_since.log" || true
} >"$OUT/control_plane_errors_focus.txt" 2>&1

{
  echo "# per host/ip focus"
  if [[ -n "$FILTER_HOST" ]]; then
    echo
    echo "## runtime host: $FILTER_HOST"
    grep -nF "$FILTER_HOST" "$OUT/runtime_logs_since.log" || true
    echo
    echo "## control-plane host: $FILTER_HOST"
    grep -nF "$FILTER_HOST" "$OUT/control_plane_logs_since.log" || true
  fi
  if [[ -n "$N_FILTER_IP" ]]; then
    for ip in $N_FILTER_IP; do
      echo
      echo "## runtime ip: $ip"
      grep -nF "$ip" "$OUT/runtime_logs_since.log" || true
      echo
      echo "## control-plane ip: $ip"
      grep -nF "$ip" "$OUT/control_plane_logs_since.log" || true
    done
  fi
  if [[ -n "$N_FILTER_URI" ]]; then
    for uri in $N_FILTER_URI; do
      echo
      echo "## runtime uri: $uri"
      grep -nF "$uri" "$OUT/runtime_logs_since.log" || true
      echo
      echo "## access uri: $uri"
      grep -nF "$uri" "$OUT/runtime_access_recent.log" || true
      echo
      echo "## control-plane uri: $uri"
      grep -nF "$uri" "$OUT/control_plane_logs_since.log" || true
    done
  fi
} >"$OUT/filters_focus.txt" 2>&1

{
  echo "# control-plane request access logs"
  grep -nF '"component":"control-plane.http"' "$OUT/control_plane_logs_since.log" || true
} >"$OUT/control_plane_requests_focus.txt" 2>&1

{
  echo "# runtime request status summary"
  grep -Eo '\"status\":[0-9]+' "$OUT/runtime_access_recent.log" | sort | uniq -c | sort -nr || true
  echo
  echo "# interesting request patterns"
  grep -nEi '\"uri\":\"/(api(/[0-9]+)?/(envelope|store|minidump)|api/2/envelope)\"|buffered to a temporary file|limit_req|too many requests|temporary unavailability' "$OUT/runtime_logs_since.log" || true
} >"$OUT/request_focus.txt" 2>&1

{
  runtime_error_count="$(grep -Eci " \\[error\\] " "$OUT/runtime_logs_since.log" || true)"
  runtime_warn_count="$(grep -Eci " \\[warn\\] " "$OUT/runtime_logs_since.log" || true)"
  runtime_notice_count="$(grep -Eci " \\[notice\\] " "$OUT/runtime_logs_since.log" || true)"
  acme_missing_count="$(grep -Eci "acme-challenge|open\\(" "$OUT/runtime_logs_since.log" || true)"
  hash_warn_count="$(grep -Eci "variables_hash" "$OUT/runtime_logs_since.log" || true)"
  collector_timeout_count="$(grep -Eci "runtime security collector failed|deadline exceeded" "$OUT/control_plane_logs_since.log" || true)"

  echo "since=$SINCE"
  echo "runtime_container=$RUNTIME_CONTAINER"
  echo "control_plane_container=$CONTROL_PLANE_CONTAINER"
  echo "runtime.error.count=$runtime_error_count"
  echo "runtime.warn.count=$runtime_warn_count"
  echo "runtime.notice.count=$runtime_notice_count"
  echo "runtime.acme_missing.count=$acme_missing_count"
  echo "runtime.variables_hash_warn.count=$hash_warn_count"
  echo "control_plane.collector_timeout.count=$collector_timeout_count"
} >"$OUT/summary.txt" 2>&1

tar -C "$OUT_BASE_DIR" -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
echo "Share this archive for analysis."
