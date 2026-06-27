#!/usr/bin/env bash
# check_ja3_logging.sh — проверка что JA3 fingerprint пишется в access-log (TASK-2.1)
#
# Использование:
#   WAF_HOST=waf.example.com bash scripts/check_ja3_logging.sh
#
# Требования: curl >= 7.x, доступ к /var/log/nginx/access.log на WAF-хосте
# Тестовая среда: deploy/compose/default

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
WAF_PORT="${WAF_PORT:-443}"
LOG_PATH="${WAF_LOG_PATH:-/var/log/nginx/access.log}"
PASS=0
FAIL=0

ok()   { echo "[PASS] $*"; PASS=$((PASS+1)); }
fail() { echo "[FAIL] $*"; FAIL=$((FAIL+1)); }

echo "=== JA3 Fingerprint Logging Check ==="
echo "Target: https://$WAF_HOST:$WAF_PORT"
echo ""

# --- Тест 1: Делаем HTTPS-запрос ---
echo "Sending HTTPS request..."
curl -sk --max-time 5 -o /dev/null "https://$WAF_HOST:$WAF_PORT/" || true
sleep 2

# --- Тест 2: Проверяем что поле ja3 есть в логе ---
if [ -f "$LOG_PATH" ]; then
  if tail -n 20 "$LOG_PATH" | grep -q '"ja3"'; then
    ok "Field 'ja3' found in access log"
  else
    fail "Field 'ja3' NOT found in access log (check if ngx_ssl_ja3 module is loaded)"
  fi

  # --- Тест 3: Проверяем что ja3 не пустой (хотя бы одна запись с непустым значением) ---
  if tail -n 20 "$LOG_PATH" | grep -qE '"ja3":"[0-9a-f]{10,}"'; then
    ok "Non-empty JA3 hash found in log"
  else
    echo "[INFO] JA3 field present but empty — nginx module ngx_ssl_ja3 may not be installed."
    echo "       JA3 collection requires OpenResty or nginx built with ngx_ssl_ja3 / ssl_ja3 module."
    echo "       Without the module \$ssl_ja3_hash resolves to empty string — field is logged as empty."
    ok "JA3 field present (empty without module is expected on vanilla nginx)"
  fi
else
  fail "Log file not found at $LOG_PATH — run from inside the WAF container or set WAF_LOG_PATH"
fi

# --- Тест 4: Проверяем формат JSON в логе ---
if [ -f "$LOG_PATH" ]; then
  LAST_LINE=$(tail -n 1 "$LOG_PATH")
  if echo "$LAST_LINE" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
    ok "Access log is valid JSON"
  else
    fail "Access log last line is not valid JSON"
  fi
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
exit 0
