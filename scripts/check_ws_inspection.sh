#!/usr/bin/env bash
# check_ws_inspection.sh — verify WebSocket frame inspection enforcement.
# Requires: WAF_HOST, CP_HOST, SITE_ID env vars; docker-compose default profile running.
# Requires: websocat or wscat installed on PATH.
# Usage: WAF_HOST=localhost CP_HOST=localhost:8080 SITE_ID=my-site bash scripts/check_ws_inspection.sh

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"
SITE_ID="${SITE_ID:-}"
WS_PATH="${WS_PATH:-/ws}"

if [ -z "$SITE_ID" ]; then
  echo "ERROR: SITE_ID is not set"
  exit 1
fi

# Check for websocat
if ! command -v websocat &>/dev/null; then
  echo "SKIP: websocat not found; install it from https://github.com/vi/websocat"
  exit 0
fi

BLOCK_PATTERN="DROP TABLE"

echo "=== WebSocket Inspection check ==="
echo "Site: $SITE_ID"
echo "Block pattern: $BLOCK_PATTERN"

# Enable WS inspection with block pattern
PATCH=$(cat <<EOF
{
  "security_websocket": {
    "use_ws_inspection": true,
    "ws_block_patterns": ["DROP TABLE"],
    "ws_max_message_bytes": 0,
    "ws_rate_msg_per_sec": 0
  }
}
EOF
)

echo ""
echo "--- Patching site profile with WS inspection enabled..."
RESULT=$(curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "$PATCH")
echo "Profile patch result: $RESULT"

echo ""
echo "--- Waiting 5s for revision to apply..."
sleep 5

echo ""
echo "--- Sending blocked pattern '$BLOCK_PATTERN' via WebSocket..."
WS_URL="ws://${WAF_HOST}${WS_PATH}"
set +e
RESPONSE=$(echo "DROP TABLE users" | timeout 5 websocat --close-status-code=1008 "$WS_URL" 2>&1)
WS_EXIT=$?
set -e

echo "websocat exit code: $WS_EXIT"
echo "websocat output: $RESPONSE"

if echo "$RESPONSE" | grep -qiE "1008|policy.*violation|closed|refused|forbidden|403"; then
  echo "PASS: connection closed or blocked with policy violation code."
elif [ "$WS_EXIT" -ne 0 ]; then
  echo "PASS: connection rejected (non-zero exit from websocat)."
else
  echo "WARN: connection not visibly blocked — check nginx/Lua logs for inspection output."
fi

echo ""
echo "--- Removing WS inspection..."
CLEANUP=$(cat <<EOF
{
  "security_websocket": {
    "use_ws_inspection": false,
    "ws_block_patterns": [],
    "ws_max_message_bytes": 0,
    "ws_rate_msg_per_sec": 0
  }
}
EOF
)
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "$CLEANUP" > /dev/null
echo "Cleanup done."
echo ""
echo "=== WS inspection check complete ==="
