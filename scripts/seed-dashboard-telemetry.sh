#!/usr/bin/env sh
# Seeds real runtime nginx access-log telemetry for Dashboard browser E2E.
set -eu
compose_file=$1
base_url=$2
username=$3
password=$4
management_host=${5:-e2e-management.test}
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
login_payload=$(printf '{\"username\":\"%s\",\"password\":\"%s\"}' "$username" "$password")
curl -fsS -c "$cookie_jar" -H 'Content-Type: application/json' -d "$login_payload" "$base_url/api/auth/login" >/dev/null
attempt=0
while [ "$attempt" -lt 30 ]; do
  stats=$(curl -fsS -b "$cookie_jar" -H "Host: $management_host" "$base_url/api/dashboard/stats" || true)
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
    exit 0
  fi
  sleep 1
  attempt=$((attempt + 1))
done
echo 'Dashboard telemetry was not aggregated within 30 seconds' >&2
exit 1
