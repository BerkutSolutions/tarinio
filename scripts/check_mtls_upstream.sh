#!/usr/bin/env bash
# check_mtls_upstream.sh — verify outgoing mTLS (WAF → upstream).
# Checks that WAF correctly presents its client certificate when proxying
# to an upstream that requires mutual TLS.
# Requires docker-compose test environment with an mTLS-capable upstream.
# Usage: WAF_HOST=localhost CP_HOST=localhost:8080 SITE_ID=my-site bash scripts/check_mtls_upstream.sh

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"
SITE_ID="${SITE_ID:-}"
TEST_PATH="${TEST_PATH:-/mtls-test}"

if [ -z "$SITE_ID" ]; then
  echo "ERROR: SITE_ID is not set"
  exit 1
fi

echo "=== Upstream mTLS check ==="
echo "WAF_HOST=$WAF_HOST  SITE_ID=$SITE_ID  TEST_PATH=$TEST_PATH"

# These paths must exist on the WAF host with valid cert/key signed by upstream's CA.
CERT_REF="${UPSTREAM_MTLS_CERT_REF:-/etc/ssl/waf-client.crt}"
KEY_REF="${UPSTREAM_MTLS_KEY_REF:-/etc/ssl/waf-client.key}"
CA_REF="${UPSTREAM_MTLS_CA_REF:-/etc/ssl/upstream-ca.crt}"

echo ""
echo "--- Enabling upstream mTLS on site $SITE_ID ..."
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "{\"upstream_routing\":{\"upstream_mtls_enabled\":true,\"upstream_mtls_cert_ref\":\"${CERT_REF}\",\"upstream_mtls_key_ref\":\"${KEY_REF}\",\"upstream_mtls_ca_ref\":\"${CA_REF}\"}}" \
  > /dev/null

echo "Waiting 5s for revision to apply..."
sleep 5

echo ""
echo "--- Sending request through WAF to mTLS upstream (expect 200):"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://${WAF_HOST}${TEST_PATH}")
echo "  HTTP $STATUS"

if echo "$STATUS" | grep -qE "^[23]"; then
  echo "PASS: WAF successfully proxied to mTLS upstream ($STATUS)"
else
  echo "WARN: expected 2xx/3xx, got $STATUS — check nginx/proxy_ssl logs"
fi

echo ""
echo "--- Disabling upstream mTLS ..."
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d '{"upstream_routing":{"upstream_mtls_enabled":false,"upstream_mtls_cert_ref":"","upstream_mtls_key_ref":"","upstream_mtls_ca_ref":""}}' > /dev/null
echo "Cleanup done."
echo ""
echo "=== Upstream mTLS check complete ==="
