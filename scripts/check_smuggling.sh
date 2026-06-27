#!/usr/bin/env bash
# check_smuggling.sh — проверка HTTP Request Smuggling / Desync Protection (TASK-1.1)
#
# Использование:
#   WAF_HOST=waf.example.com bash scripts/check_smuggling.sh
#
# Требования: curl >= 7.x
# Тестовая среда: deploy/compose/default

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
WAF_PORT="${WAF_PORT:-80}"
PASS=0
FAIL=0

ok()   { echo "[PASS] $*"; PASS=$((PASS+1)); }
fail() { echo "[FAIL] $*"; FAIL=$((FAIL+1)); }

echo "=== HTTP Smuggling Hardening Check ==="
echo "Target: http://$WAF_HOST:$WAF_PORT"
echo ""

# --- Тест 1: CL+TE запрос (Content-Length + Transfer-Encoding) ---
# Легитимный WAF должен отклонить или игнорировать Transfer-Encoding при наличии Content-Length.
STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  --max-time 5 \
  -H "Content-Length: 6" \
  -H "Transfer-Encoding: chunked" \
  --data-binary $'0\r\n\r\n' \
  "http://$WAF_HOST:$WAF_PORT/" 2>/dev/null || echo "000")

if [[ "$STATUS" == "400" || "$STATUS" == "444" || "$STATUS" == "408" || "$STATUS" == "000" ]]; then
  ok "CL+TE desync attempt rejected (status=$STATUS)"
else
  fail "CL+TE desync attempt returned unexpected status=$STATUS (expected 400/444)"
fi

# --- Тест 2: Заголовок с подчёркиванием (underscores_in_headers off) ---
# Когда underscores_in_headers off — nginx отбрасывает такие заголовки (не 400, но не пробрасывает).
# Проверяем что сервер отвечает (не падает).
STATUS2=$(curl -s -o /dev/null -w "%{http_code}" \
  --max-time 5 \
  -H "X_Custom_Header: test" \
  "http://$WAF_HOST:$WAF_PORT/" 2>/dev/null || echo "000")

if [[ "$STATUS2" != "000" && "$STATUS2" != "500" ]]; then
  ok "Underscore header request handled gracefully (status=$STATUS2)"
else
  fail "Underscore header request failed unexpectedly (status=$STATUS2)"
fi

# --- Тест 3: Transfer-Encoding заголовок не проброшен к upstream ---
# Отправляем TE заголовок — проверяем что WAF не вернул 502 из-за конфликта.
STATUS3=$(curl -s -o /dev/null -w "%{http_code}" \
  --max-time 5 \
  -H "Transfer-Encoding: identity" \
  "http://$WAF_HOST:$WAF_PORT/" 2>/dev/null || echo "000")

if [[ "$STATUS3" != "502" && "$STATUS3" != "000" ]]; then
  ok "Transfer-Encoding identity handled (status=$STATUS3)"
else
  fail "Transfer-Encoding identity caused upstream error (status=$STATUS3)"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="

if [[ $FAIL -gt 0 ]]; then
  echo "WARNING: Some checks failed. Ensure HttpStrictParsing is enabled for the site."
  exit 1
fi
exit 0
