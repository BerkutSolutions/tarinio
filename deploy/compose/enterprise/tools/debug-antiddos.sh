#!/bin/sh
set -eu

BASE_URL="${BASE_URL:-http://api-lb:8080}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-admin}"
COOKIE_JAR="$(mktemp)"
WORKDIR="$(mktemp -d)"
trap 'rm -f "$COOKIE_JAR"; rm -rf "$WORKDIR"' EXIT

curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}" \
  "${BASE_URL}/api/auth/login" >/dev/null

cat >"${WORKDIR}/antiddos.json" <<'EOF'
{
  "use_l4_guard": true,
  "chain_mode": "input",
  "conn_limit": 120,
  "rate_per_second": 240,
  "rate_burst": 480,
  "ports": [80, 443],
  "target": "DROP",
  "destination_ip": "",
  "model_enabled": true,
  "enforce_l7_rate_limit": true,
  "l7_requests_per_second": 15,
  "l7_burst": 30,
  "l7_status_code": 429,
  "model_poll_interval_seconds": 2,
  "model_decay_lambda": 0.08,
  "model_throttle_threshold": 2.5,
  "model_drop_threshold": 6.0,
  "model_hold_seconds": 60,
  "model_throttle_rate_per_second": 3,
  "model_throttle_burst": 6,
  "model_throttle_target": "REJECT",
  "model_weight_429": 1.0,
  "model_weight_403": 1.8,
  "model_weight_444": 2.2,
  "model_emergency_rps": 180,
  "model_emergency_unique_ips": 40,
  "model_emergency_per_ip_rps": 60,
  "model_weight_emergency_botnet": 6.0,
  "model_weight_emergency_single": 4.0
}
EOF

exec curl -sS -i -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -H 'X-WAF-Auto-Apply-Disabled: true' \
  -X PUT "${BASE_URL}/api/anti-ddos/settings" \
  --data-binary @"${WORKDIR}/antiddos.json"
