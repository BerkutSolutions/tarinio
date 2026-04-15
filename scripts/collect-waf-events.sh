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
read -r -p "Client IPs (optional, multiple allowed: '1.1.1.1 2.2.2.2'): " FILTER_IP
read -r -p "Service IDs / hosts (optional, multiple allowed): " FILTER_SITE
read -r -p "HTTP status codes (optional, multiple allowed: '403 429 503'): " FILTER_STATUS

if command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_BIN="docker-compose"
else
  COMPOSE_BIN="docker compose"
fi

TS="$(date +%Y%m%d_%H%M%S)"
OUT="/tmp/waf-events-${TS}"
mkdir -p "$OUT"

normalize_multi() {
  printf "%s" "$1" | tr ',;|' '   ' | xargs
}

escape_ere() {
  printf "%s" "$1" | sed -E 's/[][(){}.^$*+?|\\-]/\\&/g'
}

build_ere_pattern() {
  local raw="$1"
  local normalized token out
  normalized="$(normalize_multi "$raw")"
  out=""
  for token in $normalized; do
    if [[ -z "$out" ]]; then
      out="$(escape_ere "$token")"
    else
      out="$out|$(escape_ere "$token")"
    fi
  done
  printf "%s" "$out"
}

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

N_FILTER_IP="$(normalize_multi "$FILTER_IP")"
N_FILTER_SITE="$(normalize_multi "$FILTER_SITE")"
N_FILTER_STATUS="$(normalize_multi "$FILTER_STATUS")"
IP_ERE="$(build_ere_pattern "$N_FILTER_IP")"
SITE_ERE="$(build_ere_pattern "$N_FILTER_SITE")"

run "app_meta.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/app/meta
run "events_all.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/events
run "access_policies.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json api GET /api/access-policies
run "antiddos.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" --json antiddos get
run "bans_list.txt" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" bans list
run "audit_ban_actions.txt" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" audit --action accesspolicy.ban --limit 200
run "audit_unban_actions.txt" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" audit --action accesspolicy.unban --limit 200

if [[ -n "$N_FILTER_SITE" ]]; then
  for site in $N_FILTER_SITE; do
    run "easy_profile_${site//[^a-zA-Z0-9_.-]/_}.json" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" easy get "$site"
  done
fi

{
  echo "# filters: ips='$N_FILTER_IP' sites='$N_FILTER_SITE' statuses='$N_FILTER_STATUS'"
  if [[ -n "$N_FILTER_IP" ]]; then
    for ip in $N_FILTER_IP; do
      echo
      echo "## ip: $ip"
      grep -nF "$ip" "$OUT/events_all.json" || true
    done
  fi
  if [[ -n "$N_FILTER_SITE" ]]; then
    for site in $N_FILTER_SITE; do
      echo
      echo "## site: $site"
      grep -nF "$site" "$OUT/events_all.json" || true
    done
  fi
  if [[ -n "$N_FILTER_STATUS" ]]; then
    for status in $N_FILTER_STATUS; do
      echo
      echo "## status: $status"
      grep -n "\"status\":${status}" "$OUT/events_all.json" || true
    done
  fi
  echo
  echo "## key-security-patterns"
  grep -nEi "request burst detected|access blocked|rate limit triggered|blocked|limit_req|too many requests|service unavailable|temporary unavailability|waf fallback|ban" "$OUT/events_all.json" || true
} >"$OUT/events_focus.txt" 2>&1

{
  echo "# filters: ips='$N_FILTER_IP' sites='$N_FILTER_SITE'"
  echo
  echo "## bans-list"
  if [[ -n "$N_FILTER_IP" || -n "$N_FILTER_SITE" ]]; then
    if [[ -n "$N_FILTER_IP" ]]; then
      for ip in $N_FILTER_IP; do
        echo
        echo "### bans for ip: $ip"
        grep -nF "$ip" "$OUT/bans_list.txt" || true
      done
    fi
    if [[ -n "$N_FILTER_SITE" ]]; then
      for site in $N_FILTER_SITE; do
        echo
        echo "### bans for site: $site"
        grep -nF "$site" "$OUT/bans_list.txt" || true
      done
    fi
  else
    cat "$OUT/bans_list.txt"
  fi

  echo
  echo "## audit-ban-actions (why banned)"
  if [[ -n "$N_FILTER_IP" || -n "$N_FILTER_SITE" ]]; then
    if [[ -n "$N_FILTER_IP" ]]; then
      for ip in $N_FILTER_IP; do
        echo
        echo "### audit ban entries for ip: $ip"
        grep -nF "$ip" "$OUT/audit_ban_actions.txt" || true
      done
    fi
    if [[ -n "$N_FILTER_SITE" ]]; then
      for site in $N_FILTER_SITE; do
        echo
        echo "### audit ban entries for site: $site"
        grep -nF "$site" "$OUT/audit_ban_actions.txt" || true
      done
    fi
  else
    cat "$OUT/audit_ban_actions.txt"
  fi

  echo
  echo "## events-ban-related (runtime reason trail)"
  if [[ -n "$N_FILTER_IP" || -n "$N_FILTER_SITE" ]]; then
    if [[ -n "$N_FILTER_IP" ]]; then
      for ip in $N_FILTER_IP; do
        echo
        echo "### events for ip: $ip"
        grep -nF "$ip" "$OUT/events_all.json" | grep -Ei "ban|blocked|burst|rate|limit|deny|403|429|444|503" || true
      done
    fi
    if [[ -n "$N_FILTER_SITE" ]]; then
      for site in $N_FILTER_SITE; do
        echo
        echo "### events for site: $site"
        grep -nF "$site" "$OUT/events_all.json" | grep -Ei "ban|blocked|burst|rate|limit|deny|403|429|444|503" || true
      done
    fi
  else
    grep -nEi "ban|blocked|burst|rate limit|limit_req|denylist|accesspolicy\\.ban|403|429|444|503" "$OUT/events_all.json" || true
  fi
} >"$OUT/bans_focus.txt" 2>&1

if [[ -n "$IP_ERE" ]]; then
  run_sh "runtime_access_bad_ip.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -E '$IP_ERE' /var/log/nginx/access.log | tail -n 2000\""
  run_sh "runtime_access_bad_ip_block_statuses.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -E '$IP_ERE' /var/log/nginx/access.log | grep -E '\\\"status\\\":(403|429|444|503)| 403 | 429 | 444 | 503 ' | tail -n 1000\""
fi

if [[ -n "$SITE_ERE" ]]; then
  run_sh "runtime_nginx_site_lookup.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -R --line-number -E 'server_name ($SITE_ERE)' /etc/waf/nginx 2>/dev/null\""
  run_sh "runtime_nginx_site_conf_focus.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"nginx -T 2>/tmp/nginxT.txt; grep -nE '($SITE_ERE)|limit_req|waf_rate_limited|blacklist_uri|deny ' /tmp/nginxT.txt | tail -n 1200\""
fi

run "runtime_logs_since_30m.log" docker logs --since=30m "$RUNTIME_CONTAINER"
run "control_plane_logs_since_30m.log" docker logs --since=30m "$CONTROL_PLANE_CONTAINER"

tar -C /tmp -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
echo "Share this archive for analysis."
