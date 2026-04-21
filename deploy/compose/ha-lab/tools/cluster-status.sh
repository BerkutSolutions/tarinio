#!/bin/sh
set -eu

BASE_URL="${BASE_URL:-http://api-lb:8080}"
USERNAME="${USERNAME:-${CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME:-admin}}"
PASSWORD="${PASSWORD:-${CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD:-admin}}"
COOKIE_JAR="$(mktemp)"
trap 'rm -f "$COOKIE_JAR"' EXIT

curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}" \
  "${BASE_URL}/api/auth/login" >/dev/null

echo "Load balancer node distribution:"
for _ in 1 2 3 4 5 6; do
  curl -fsS -b "$COOKIE_JAR" "${BASE_URL}/api/app/meta" | jq -r '"  node=" + (.ha_node_id // "-") + " version=" + (.app_version // "-")'
done

echo
echo "Direct nodes:"
for NODE in control-plane-a control-plane-b; do
  curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
    -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}" \
    "http://${NODE}:8080/api/auth/login" >/dev/null
  curl -fsS -b "$COOKIE_JAR" "http://${NODE}:8080/api/app/meta" | jq --arg node "$NODE" -r '"  " + $node + " -> " + (.ha_node_id // "-")'
done
