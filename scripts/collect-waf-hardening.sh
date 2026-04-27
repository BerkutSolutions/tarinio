#!/usr/bin/env bash

if [[ -z "${BASH_VERSION:-}" ]]; then
  if command -v bash >/dev/null 2>&1; then
    exec bash "$0" "$@"
  fi
  echo "This script requires bash."
  exit 1
fi

set -u

# Hardening diagnostics collector for external scan readiness.
# Usage:
#   bash scripts/collect-waf-hardening.sh
# Optional env:
#   RUNTIME_CONTAINER=tarinio-runtime
#   DEPLOY_DIR=/opt/tarinio/deploy/compose/default
#   EXPECTED_TCP_TIMESTAMPS=0

RUNTIME_CONTAINER="${RUNTIME_CONTAINER:-tarinio-runtime}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/tarinio/deploy/compose/default}"
EXPECTED_TCP_TIMESTAMPS="${EXPECTED_TCP_TIMESTAMPS:-0}"
OUT_BASE_DIR="${OUT_BASE_DIR:-/tmp}"

TS="$(date +%Y%m%d_%H%M%S)"
OUT="${OUT_BASE_DIR%/}/waf-hardening-${TS}"
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

runtime_value=""
if command -v docker >/dev/null 2>&1; then
  run "docker_inspect_runtime.json" docker inspect "$RUNTIME_CONTAINER"
  run_sh "runtime_tcp_timestamps.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"cat /proc/sys/net/ipv4/tcp_timestamps\""
  run_sh "runtime_sysctl_tcp_timestamps.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"sysctl net.ipv4.tcp_timestamps 2>/dev/null || true\""
  run_sh "runtime_nginx_tls_hsts_grep.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -R --line-number -E 'ssl_protocols|ssl_ciphers|Strict-Transport-Security|HSTS' /etc/waf/nginx /etc/waf/tls 2>/dev/null\""
  runtime_value="$(grep -Eo '^[01]$' "$OUT/runtime_tcp_timestamps.txt" | head -n1 || true)"
fi

if [[ -r /proc/sys/net/ipv4/tcp_timestamps ]]; then
  run_sh "host_tcp_timestamps.txt" "cat /proc/sys/net/ipv4/tcp_timestamps"
fi
if command -v sysctl >/dev/null 2>&1; then
  run "host_sysctl_tcp_timestamps.txt" sysctl net.ipv4.tcp_timestamps
fi

if [[ -d "$DEPLOY_DIR" ]]; then
  if command -v docker >/dev/null 2>&1; then
    run_sh "compose_effective_config.txt" "cd '$DEPLOY_DIR' && (docker compose config 2>/dev/null || docker-compose config 2>/dev/null || true)"
  fi
fi

status="unknown"
if [[ -n "$runtime_value" && "$runtime_value" == "$EXPECTED_TCP_TIMESTAMPS" ]]; then
  status="pass"
elif [[ -n "$runtime_value" ]]; then
  status="fail"
fi

{
  echo "expected_tcp_timestamps=$EXPECTED_TCP_TIMESTAMPS"
  echo "runtime_container=$RUNTIME_CONTAINER"
  echo "runtime_tcp_timestamps=${runtime_value:-unavailable}"
  echo "status=$status"
  echo "note=pass means runtime tcp_timestamps matches the expected value"
} >"$OUT/summary.txt"

tar -C "$OUT_BASE_DIR" -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
