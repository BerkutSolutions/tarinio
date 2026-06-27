#!/usr/bin/env bash
# check_ja3_block.sh — TASK-2.2 verification script
# Simulates a known bad TLS fingerprint (AES128-SHA cipher suite) and checks
# that WAF returns 403 or 444 when the JA3 is in the blacklist.
#
# Prerequisites:
#   - WAF_HOST set to the WAF address (default: localhost)
#   - The test site must have a JA3 fingerprint corresponding to AES128-SHA in its blacklist
#   - HTTPS must be enabled on the test site
#
# The JA3 fingerprint for TLS 1.2 + AES128-SHA (no extensions) is typically:
#   769,47,0-10-11,0,0 → md5 → varies by implementation
# Use this script to verify WAF blocks the known-bad fingerprint.

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"

echo "=== JA3 Block Check ==="
echo "Target: https://${WAF_HOST}/"
echo ""

# Send request with restricted TLS profile (TLS 1.2 only, single cipher)
# This produces a distinct JA3 fingerprint that can be added to the blacklist.
STATUS=$(curl -sk \
  --tls-max 1.2 \
  --ciphers 'AES128-SHA' \
  -o /dev/null \
  -w "%{http_code}" \
  "https://${WAF_HOST}/")

echo "Response status: ${STATUS}"

if [ "${STATUS}" = "403" ] || [ "${STATUS}" = "444" ]; then
  echo "PASS: WAF blocked the request with JA3 fingerprint in blacklist."
  exit 0
else
  echo "FAIL: Expected 403 or 444, got ${STATUS}."
  echo "Make sure:"
  echo "  1. HTTPS is enabled on the test site."
  echo "  2. The JA3 fingerprint for AES128-SHA/TLS1.2 is added to blacklist_ja3."
  echo "  3. A new revision has been compiled and applied."
  exit 1
fi
