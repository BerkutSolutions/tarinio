#!/usr/bin/env bash
# check_credential_stuffing.sh
# Симулирует 20 неудачных логинов за 10 секунд и проверяет что IP
# попал в sentinel state с сигналом signal_credential_stuffing.
set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"

echo "==> Simulating 20 failed login attempts..."
for i in $(seq 1 20); do
  curl -s -o /dev/null -X POST "http://${WAF_HOST}/login" \
    -d "user=test&pass=wrong${i}" || true
done

echo "==> Waiting 15s for sentinel tick..."
sleep 15

echo "==> Checking sentinel state for signal_credential_stuffing..."
RESULT=$(curl -s "http://${CP_HOST}/api/sentinel/state" | \
  jq '.entries[] | select(.reason_codes != null and (.reason_codes[] == "signal_credential_stuffing"))' 2>/dev/null || echo "")

if [ -n "$RESULT" ]; then
  echo "OK: signal_credential_stuffing found in sentinel state"
  echo "$RESULT" | jq .
else
  echo "FAIL: signal_credential_stuffing NOT found in sentinel state"
  exit 1
fi
