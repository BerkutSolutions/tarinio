#!/usr/bin/env bash
# check_sentinel_signals.sh
# Симулирует серию запросов для триггера antibot fail и bad behavior.
# Требует что antibot включён на тестовом сайте.
set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"

echo "==> Simulating 10 requests without antibot cookie (expect 403)..."
for i in $(seq 1 10); do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Cookie: " "http://${WAF_HOST}/") || true
  echo "  request $i: $STATUS"
done

echo "==> Waiting 15s for sentinel tick..."
sleep 15

echo "==> Checking sentinel state for signal_antibot_fail..."
ANTIBOT=$(curl -s "http://${CP_HOST}/api/sentinel/state" | \
  jq '.entries[] | select(.reason_codes != null and (.reason_codes[] == "signal_antibot_fail"))' 2>/dev/null || echo "")

if [ -n "$ANTIBOT" ]; then
  echo "OK: signal_antibot_fail found"
  echo "$ANTIBOT" | jq .
else
  echo "WARN: signal_antibot_fail not found (antibot may not be enabled)"
fi

echo "==> Checking sentinel state for signal_bad_behavior..."
BADBEH=$(curl -s "http://${CP_HOST}/api/sentinel/state" | \
  jq '.entries[] | select(.reason_codes != null and (.reason_codes[] == "signal_bad_behavior"))' 2>/dev/null || echo "")

if [ -n "$BADBEH" ]; then
  echo "OK: signal_bad_behavior found"
  echo "$BADBEH" | jq .
else
  echo "WARN: signal_bad_behavior not found (bad_behavior zone may not be triggered)"
fi
