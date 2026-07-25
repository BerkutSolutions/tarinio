#!/usr/bin/env sh
# Seeds real runtime nginx access-log telemetry for Dashboard browser E2E.
set -eu
compose_file=$1
base_url=$2
username=$3
password=$4
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT INT TERM
events_file="$tmp_dir/events.jsonl"
cookie_jar="$tmp_dir/cookies.txt"
hour=0
while [ "$hour" -lt 24 ]; do
  timestamp=$(date -u -d "-$hour hours" +%Y-%m-%dT%H:%M:%SZ)
  printf '%s\n' "{\"timestamp\":\"$timestamp\",\"request_id\":\"e2e-dashboard-request-$hour\",\"client_ip\":\"198.51.100.24\",\"country\":\"RU\",\"city\":\"Moscow\",\"host\":\"dashboard-e2e.test\",\"method\":\"GET\",\"uri\":\"/catalog/$hour\",\"status\":200,\"bytes_sent\":0,\"referer\":\"\",\"user_agent\":\"WAF Dashboard E2E\",\"site\":\"dashboard-e2e-primary\",\"security_reason\":\"\",\"upstream_addr\":\"\",\"request_time\":0.001}" >>"$events_file"
  printf '%s\n' "{\"timestamp\":\"$timestamp\",\"request_id\":\"e2e-dashboard-attack-$hour\",\"client_ip\":\"203.0.113.38\",\"country\":\"DE\",\"city\":\"Frankfurt\",\"host\":\"dashboard-e2e-secondary.test\",\"method\":\"GET\",\"uri\":\"/waf-test/$hour\",\"status\":403,\"bytes_sent\":0,\"referer\":\"\",\"user_agent\":\"WAF Dashboard E2E\",\"site\":\"dashboard-e2e-secondary\",\"security_reason\":\"access_blocked\",\"upstream_addr\":\"\",\"request_time\":0.001}" >>"$events_file"
  hour=$((hour + 1))
done
docker compose -f "$compose_file" exec -T runtime sh -c 'cat >> /var/log/nginx/access.log' <"$events_file"
login_payload=$(printf '{"username":"%s","password":"%s"}' "$username" "$password")
# The dashboard seed calls the control-plane's direct loopback listener.  It is
# deliberately not the runtime HTTPS virtual host used by browser E2E, so an
# e2e-management.test Host header would be rejected by the control-plane host
# guard with HTTP 400 before authentication.
login_response="$tmp_dir/login-response.json"
if ! curl -sS -o "$login_response" -w '%{http_code}' -c "$cookie_jar" -H 'Content-Type: application/json' -d "$login_payload" "$base_url/api/auth/login" | grep -qx '200'; then
  printf 'Dashboard seed login failed: %s\n' "$(cat "$login_response" 2>/dev/null || true)" >&2
  exit 1
fi
attempt=0
while [ "$attempt" -lt 30 ]; do
  stats_response="$tmp_dir/dashboard-response.json"
  stats_status=$(curl -sS -o "$stats_response" -w '%{http_code}' -b "$cookie_jar" "$base_url/api/dashboard/stats" || true)
  stats=$(cat "$stats_response" 2>/dev/null || true)
  if [ "$stats_status" != '200' ]; then
    printf 'Dashboard seed stats status=%s body=%s\n' "$stats_status" "$stats" >&2
    exit 1
  fi
  if printf '%s' "$stats" | python3 -c '
import json
import sys

payload = json.load(sys.stdin)
required = (
    "top_attacker_ips",
    "top_attacker_countries",
    "most_attacked_urls",
    "popular_errors",
)
if len(payload.get("requests_series", [])) != 24:
    raise SystemExit(1)
if len(payload.get("request_top_sites", [])) < 2:
    raise SystemExit(1)
if not all(payload.get(key) for key in required):
    raise SystemExit(1)
'; then
    requests_response="$tmp_dir/requests-response.json"
    requests_status=$(curl -sS -o "$requests_response" -w '%{http_code}' -b "$cookie_jar" "$base_url/api/requests?limit=500" || true)
    if [ "$requests_status" = '200' ] && python3 - "$requests_response" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as source:
    rows = json.load(source)
ids = {str((row.get("entry") or {}).get("request_id") or "") for row in rows if isinstance(row, dict)}
if not any(value.startswith("e2e-dashboard-request-") for value in ids):
    raise SystemExit(1)
if not any(value.startswith("e2e-dashboard-attack-") for value in ids):
    raise SystemExit(1)
PY
    then
      exit 0
    fi
  fi
  sleep 1
  attempt=$((attempt + 1))
done
echo 'Dashboard telemetry was not aggregated within 30 seconds' >&2
exit 1
