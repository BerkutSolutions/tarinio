#!/usr/bin/env bash

if [[ -z "${BASH_VERSION:-}" ]]; then
  if command -v bash >/dev/null 2>&1; then
    exec bash "$0" "$@"
  fi
  echo "This script requires bash."
  exit 1
fi

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
WAF_CLI_BIN="${WAF_CLI_BIN:-waf-cli}"
NON_INTERACTIVE="${NON_INTERACTIVE:-0}"

prompt_value() {
  local var_name="$1"
  local prompt_text="$2"
  local secret="${3:-0}"
  local current_value="${!var_name:-}"

  if [[ -n "$current_value" ]]; then
    return
  fi
  if [[ "$NON_INTERACTIVE" == "1" ]]; then
    return
  fi
  if [[ "$secret" == "1" ]]; then
    read -r -s -p "$prompt_text" "$var_name"
    echo
    return
  fi
  read -r -p "$prompt_text" "$var_name"
}

prompt_value WAF_USER "WAF username: "
prompt_value WAF_PASS "WAF password: " 1
prompt_value FILTER_IP "Client IPs (optional, multiple allowed: '1.1.1.1 2.2.2.2'): "
prompt_value FILTER_SITE "Target site IDs / hosts (optional, multiple allowed, e.g. 'site-a.example.com api.example.com'): "
prompt_value FILTER_STATUS "HTTP status codes (optional, multiple allowed: '403 429 503'): "

if [[ -z "${WAF_USER:-}" || -z "${WAF_PASS:-}" ]]; then
  echo "WAF_USER and WAF_PASS are required."
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
OUT="${OUT_BASE_DIR%/}/waf-events-${TS}"
mkdir -p "$OUT"

normalize_multi() {
  printf "%s" "$1" | tr ',;|' '   ' | tr -d "\"'" | awk '{$1=$1; print}'
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

redact_sensitive() {
  local text="$1"
  if [[ -n "${WAF_PASS:-}" ]]; then
    text="${text//${WAF_PASS}/***}"
  fi
  printf "%s" "$text"
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

run_sh() {
  local name="$1"
  local cmd="$2"
  local safe_cmd
  safe_cmd="$(redact_sensitive "$cmd")"
  {
    echo "# cmd: $safe_cmd"
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
    run "$name" "$WAF_CLI_BIN" --username "$WAF_USER" --password "$WAF_PASS" "$@"
    return
  fi
  run "$name" $COMPOSE_BIN --profile tools run --rm cli --username "$WAF_USER" --password "$WAF_PASS" "$@"
}

if [[ "$CLI_MODE" == "compose" && ! -d "$DEPLOY_DIR" ]]; then
  echo "DEPLOY_DIR not found: $DEPLOY_DIR"
  exit 1
fi
if [[ "$CLI_MODE" == "compose" ]]; then
  cd "$DEPLOY_DIR" || exit 1
fi

N_FILTER_IP="$(normalize_multi "$FILTER_IP")"
N_FILTER_SITE="$(normalize_multi "$FILTER_SITE")"
N_FILTER_STATUS="$(normalize_multi "$FILTER_STATUS")"
IP_ERE="$(build_ere_pattern "$N_FILTER_IP")"
SITE_ERE="$(build_ere_pattern "$N_FILTER_SITE")"

run_cli "app_meta.json" --json api GET /api/app/meta
run_cli "sites.json" --json api GET /api/sites
run_cli "events_all.json" --json api GET /api/events
run_cli "access_policies.json" --json api GET /api/access-policies
run_cli "antiddos.json" --json antiddos get
run_cli "bans_list.txt" bans list
run_cli "audit_ban_actions.txt" audit --action accesspolicy.ban --limit 200
run_cli "audit_unban_actions.txt" audit --action accesspolicy.unban --limit 200

if [[ -n "$N_FILTER_SITE" ]]; then
  for site in $N_FILTER_SITE; do
    run_cli "easy_profile_${site//[^a-zA-Z0-9_.-]/_}.json" easy get "$site"
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
  if [[ -n "$N_FILTER_SITE" ]]; then
    for site in $N_FILTER_SITE; do
      echo
      echo "### site: $site"
      grep -nF "$site" "$OUT/events_all.json" | grep -Ei "request burst detected|access blocked|rate limit triggered|blocked|limit_req|too many requests|service unavailable|temporary unavailability|waf fallback|ban" || true
    done
  else
    grep -nEi "request burst detected|access blocked|rate limit triggered|blocked|limit_req|too many requests|service unavailable|temporary unavailability|waf fallback|ban" "$OUT/events_all.json" || true
  fi
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
  run_sh "runtime_access_site.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -E '\\\"host\\\":\\\"($SITE_ERE)\\\"' /var/log/nginx/access.log | tail -n 4000\""
  run_sh "runtime_access_site_block_statuses.log" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -E '\\\"host\\\":\\\"($SITE_ERE)\\\"' /var/log/nginx/access.log | grep -E '\\\"status\\\":(403|429|444|500|502|503|504)' | tail -n 2000\""
  run_sh "runtime_nginx_site_lookup.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"grep -R --line-number -E 'server_name ($SITE_ERE)' /etc/waf/nginx 2>/dev/null\""
  run_sh "runtime_nginx_site_conf_focus.txt" "docker exec '$RUNTIME_CONTAINER' sh -lc \"nginx -T 2>/tmp/nginxT.txt; grep -nE '($SITE_ERE)|limit_req|waf_rate_limited|blacklist_uri|deny ' /tmp/nginxT.txt | tail -n 1200\""
fi

{
  echo "# site diagnostics summary"
  echo "filters.sites=$N_FILTER_SITE"
  if [[ -n "$N_FILTER_SITE" ]]; then
    for site in $N_FILTER_SITE; do
      echo
      echo "## $site"
      echo "events_all.matches=$(grep -cF "$site" "$OUT/events_all.json" || true)"
      echo "sites.matches=$(grep -cF "$site" "$OUT/sites.json" || true)"
      echo "runtime_access.matches=$(grep -cE "\\\"host\\\":\\\"$site\\\"" "$OUT/runtime_access_site.log" || true)"
      echo "runtime_nginx_server_name.matches=$(grep -cF "$site" "$OUT/runtime_nginx_site_lookup.txt" || true)"
      echo "easy_profile_status=$(grep -Eo 'HTTP [0-9]+' "$OUT/easy_profile_${site//[^a-zA-Z0-9_.-]/_}.json" | tail -n1 || true)"
    done
  fi
} >"$OUT/site_diagnostics_summary.txt" 2>&1

run "runtime_logs_since_30m.log" docker logs --since=30m "$RUNTIME_CONTAINER"
run "control_plane_logs_since_30m.log" docker logs --since=30m "$CONTROL_PLANE_CONTAINER"

tar -C "$OUT_BASE_DIR" -czf "${OUT}.tar.gz" "$(basename "$OUT")"

echo
echo "Done."
echo "Collected directory: $OUT"
echo "Archive: ${OUT}.tar.gz"
echo "Share this archive for analysis."
