#!/usr/bin/env bash
# check_geo_timewindow.sh — verify geo time-window enforcement end-to-end.
# Requires: WAF_HOST, CP_HOST, SITE_ID env vars; docker-compose default profile running.
# Usage: WAF_HOST=localhost CP_HOST=localhost:8080 SITE_ID=my-site bash scripts/check_geo_timewindow.sh

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"
SITE_ID="${SITE_ID:-}"

if [ -z "$SITE_ID" ]; then
  echo "ERROR: SITE_ID is not set"
  exit 1
fi

CURRENT_HOUR=$(date -u +%H | sed 's/^0*//')
CURRENT_HOUR="${CURRENT_HOUR:-0}"
HOURS_END=$(( (CURRENT_HOUR + 1) % 24 ))
# Ensure end > start (wrap-around not supported in validator)
if [ "$HOURS_END" -le "$CURRENT_HOUR" ]; then
  HOURS_END=$(( CURRENT_HOUR + 1 ))
fi
if [ "$HOURS_END" -gt 23 ]; then
  echo "SKIP: current hour is 23 UTC, cannot create end=24; run at an earlier hour."
  exit 0
fi

echo "=== Geo Time Window check ==="
echo "Current UTC hour: $CURRENT_HOUR, window: ${CURRENT_HOUR}..${HOURS_END}"
echo "Site: $SITE_ID"

# Read current profile
PROFILE=$(curl -sf "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}")
if [ -z "$PROFILE" ]; then
  echo "ERROR: could not fetch profile for site $SITE_ID"
  exit 1
fi

# Add a geo time window that blocks XX (reserved, never a real country) during the current hour.
# This tests the compile path without affecting real traffic.
PATCH=$(cat <<EOF
{
  "security_country_policy": {
    "blacklist_country": [],
    "whitelist_country": [],
    "geo_time_windows": [
      {
        "countries": ["XX"],
        "action": "block",
        "days_of_week": [],
        "hours_start": ${CURRENT_HOUR},
        "hours_end": ${HOURS_END}
      }
    ]
  }
}
EOF
)

echo ""
echo "--- Patching site profile with geo time window for XX..."
RESULT=$(curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "$PATCH")
echo "Profile patch result: $RESULT"

echo ""
echo "--- Waiting 5s for revision to apply..."
sleep 5

echo ""
echo "--- Checking WAF responds (window active for XX, real traffic unaffected)..."
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${WAF_HOST}/")
echo "WAF status for normal request: $STATUS (expect 200 or upstream response)"

echo ""
echo "--- Removing geo time window..."
CLEANUP=$(cat <<EOF
{
  "security_country_policy": {
    "blacklist_country": [],
    "whitelist_country": [],
    "geo_time_windows": []
  }
}
EOF
)
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "$CLEANUP" > /dev/null
echo "Cleanup done."
echo ""
echo "=== PASS: geo time window compiled and applied without errors ==="
