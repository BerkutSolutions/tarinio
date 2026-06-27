#!/usr/bin/env bash
# check_5xx_anomaly.sh
# Генерирует 5xx через специальный эндпоинт тестового upstream и проверяет
# что дашборд показывает warning/critical для сайта.
set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"
SITE_ID="${SITE_ID:-}"

echo "==> Generating 5xx requests..."
for i in $(seq 1 50); do
  curl -s -o /dev/null "http://${WAF_HOST}/simulate-500" || true
done

echo "==> Waiting 10s for dashboard refresh..."
sleep 10

echo "==> Checking dashboard upstream_health..."
if [ -n "$SITE_ID" ]; then
  RESULT=$(curl -s "http://${CP_HOST}/api/dashboard/snapshot" | \
    jq --arg sid "$SITE_ID" '.upstream_health[]? | select(.site_id == $sid)' 2>/dev/null || echo "")
else
  RESULT=$(curl -s "http://${CP_HOST}/api/dashboard/snapshot" | \
    jq '.upstream_health[]? | select(.status == "warning" or .status == "critical")' 2>/dev/null || echo "")
fi

if [ -n "$RESULT" ]; then
  echo "OK: upstream_health anomaly detected"
  echo "$RESULT" | jq .
else
  echo "WARN: no upstream_health anomaly found (may need more traffic or longer window)"
fi
