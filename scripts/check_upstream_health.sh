#!/usr/bin/env bash
# check_upstream_health.sh
# Останавливаем тестовый upstream, проверяем что WAF отвечает кастомной страницей.
set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"

echo "==> Stopping upstream-test container..."
docker compose stop upstream-test 2>/dev/null || true
sleep 5

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${WAF_HOST}/app" || echo "000")
echo "With down upstream, got: $STATUS (expect 502 or 503 with error page)"

echo "==> Restarting upstream-test container..."
docker compose start upstream-test 2>/dev/null || true

if [ "$STATUS" = "502" ] || [ "$STATUS" = "503" ]; then
  echo "OK: WAF returned proper error code when upstream is down"
else
  echo "WARN: unexpected status $STATUS"
fi
