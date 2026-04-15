#!/usr/bin/env bash
set -u

# Interactive diagnostics collector focused on security events and ban/rate-limit issues.
# Usage:
#   bash scripts/collect-waf-events.sh
# Optional env:
#   DEPLOY_DIR=/opt/tarinio/deploy/compose/default
#   RUNTIME_CONTAINER=tarinio-runtime
#   CONTROL_PLANE_CONTAINER=tarinio-control-plane

DEPLOY_DIR="${DEPLOY_DIR:-/opt/tarinio/deploy/compose/default}"
RUNTIME_CONTAINER="${RUNTIME_CONTAINER:-tarinio-runtime}"
CONTROL_PLANE_CONTAINER="${CONTROL_PLANE_CONTAINER:-tarinio-control-plane}"

read -r -p "WAF username: " WAF_USER
read -r -s -p "WAF password: " WAF_PASS
echo
read -r -p "Client IP (optional): " FILTER_IP
read -r -p "Service ID / host (optional): " FILTER_SITE
read -r -p "HTTP status code (optional, e.g. 429): " FILTER_STATUS

if command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_BIN="docker-compose"
else
  COMPOSE_BIN="docker compose"
fi

TS="$(date +%Y%m%d_%H%M%S)"
OUT="/tmp/waf-events-${TS}"
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

if [[ ! -d "$DEPLOY_DIR" ]]; then
  echo "DEPLOY_DIR not found: $DEPLOY_DIR"
  exit 1
fi
cd "$DEPLOY_DIR" || exit 1

run "app_meta.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/app/meta
run "events_all.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/events
run "access_policies.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/access-policies
run "antiddos.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json antiddos get
run "bans_list.txt" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" bans list

if [[ -n "${FILTER_SITE}" ]]; then
  run "easy_profile.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" easy get "$FILTER_SITE"
fi

if [[ -n "${FILTER_IP}" || -n "${FILTER_SITE}" || -n "${FILTER_STATUS}" ]]; then
  {
    echo "# filters: ip='${FILTER_IP}' site='${FILTER_SITE}' status='${FILTER_STATUS}'"
    if [[ -n "${FILTER_IP}" ]]; then
      grep -n "$FILTER_IP" "$OUT/events_all.json" || true
    fi
    if [[ -n "${FILTER_SITE}" ]]; then
      grep -n "$FILTER_SITE" "$OUT/events_all.json" || true
    fi
    if [[ -n "${FILTER_STATUS}" ]]; then
      grep -n "\"status\":${FILTER_STATUS}" "$OUT/events_all.json" || true
    fi
    grep -n "request burst detected (not blocked)\|request burst detected on path (not blocked)\|access blocked\|rate limit triggered" "$OUT/events_all.json" || true
  } >"$OUT/events_focus.txt" 2>&1
else
  run_sh "events_focus.txt" "grep -n 'request burst detected (not blocked)\\|request burst detected on path (not blocked)\\|access blocked\\|rate limit triggered' '$OUT/events_all.json' || true"
fi

if [[ -n "${FILTER_IP}" ]]; then
  run_sh "runtime_access_bad_ip.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep '$FILTER_IP' /var/log/nginx/access.log | tail -n 1000\""
  run_sh "runtime_access_bad_ip_block_statuses.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep '$FILTER_IP' /var/log/nginx/access.log | grep -E '\\\"status\\\":(403|429|444|503)| 403 | 429 | 444 | 503 ' | tail -n 500\""
fi

if [[ -n "${FILTER_SITE}" ]]; then
  run_sh "runtime_nginx_site_lookup.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -R --line-number 'server_name $FILTER_SITE' /etc/waf/nginx 2>/dev/null\""
  run_sh "runtime_nginx_site_conf_focus.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"nginx -T 2>/tmp/nginxT.txt; grep -n '$FILTER_SITE\\|limit_req\\|waf_rate_limited\\|blacklist_uri\\|deny ' /tmp/nginxT.txt | tail -n 800\""
fi

run "runtime_logs_since_30m.log" docker logs --since=30m "$RUNTIME_CONTAINER"
run "control_plane_logs_since_30m.log" docker logs --since=30m "$CONTROL_PLANE_CONTAINER"

tar -C /tmp -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
echo "Share this archive for analysis."
