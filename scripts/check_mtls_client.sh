#!/usr/bin/env bash
# check_mtls_client.sh — verify incoming mTLS (client → WAF).
# Generates a test CA + client cert, then checks that:
#   - requests WITHOUT cert → 400 (or 403/495)
#   - requests WITH cert    → 200 (or non-4xx)
# Usage: WAF_HOST=my.site.example bash scripts/check_mtls_client.sh

set -euo pipefail

WAF_HOST="${WAF_HOST:-localhost}"
CP_HOST="${CP_HOST:-localhost:8080}"
SITE_ID="${SITE_ID:-}"

if [ -z "$SITE_ID" ]; then
  echo "ERROR: SITE_ID is not set"
  exit 1
fi

echo "=== mTLS client check ==="
echo "WAF_HOST=$WAF_HOST  SITE_ID=$SITE_ID"

# Generate ephemeral CA
openssl req -x509 -newkey rsa:2048 -keyout /tmp/mtls_ca.key -out /tmp/mtls_ca.crt \
  -days 1 -nodes -subj "/CN=TestCA" 2>/dev/null

# Generate client cert signed by CA
openssl req -newkey rsa:2048 -keyout /tmp/mtls_client.key -out /tmp/mtls_client.csr \
  -nodes -subj "/CN=TestClient" 2>/dev/null
openssl x509 -req -in /tmp/mtls_client.csr -CA /tmp/mtls_ca.crt -CAkey /tmp/mtls_ca.key \
  -CAcreateserial -out /tmp/mtls_client.crt -days 1 2>/dev/null

# Upload CA path to profile (assumes /tmp/mtls_ca.crt accessible on WAF host; adjust for prod)
CA_PATH="/tmp/mtls_ca.crt"

echo ""
echo "--- Enabling mTLS on site $SITE_ID ..."
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d "{\"front_service\":{\"mtls_enabled\":true,\"mtls_optional\":false,\"mtls_verify_depth\":2,\"mtls_client_ca_ref\":\"${CA_PATH}\",\"mtls_pass_headers\":false}}" \
  > /dev/null

echo "Waiting 5s for revision to apply..."
sleep 5

echo ""
echo "--- Request WITHOUT client cert (expect 400/403/495):"
STATUS_NO=$(curl -sk -o /dev/null -w "%{http_code}" "https://${WAF_HOST}/")
echo "  HTTP $STATUS_NO"

echo ""
echo "--- Request WITH client cert (expect 200 or non-4xx):"
STATUS_OK=$(curl -sk --cert /tmp/mtls_client.crt --key /tmp/mtls_client.key \
  -o /dev/null -w "%{http_code}" "https://${WAF_HOST}/")
echo "  HTTP $STATUS_OK"

# Evaluate
if echo "$STATUS_NO" | grep -qE "^(400|403|495|496)$"; then
  echo "PASS: request without cert correctly rejected ($STATUS_NO)"
else
  echo "WARN: expected 400/403/495 without cert, got $STATUS_NO"
fi

if echo "$STATUS_OK" | grep -qE "^[23]"; then
  echo "PASS: request with cert accepted ($STATUS_OK)"
else
  echo "WARN: expected 2xx/3xx with cert, got $STATUS_OK"
fi

echo ""
echo "--- Disabling mTLS ..."
curl -sf -X PATCH "http://${CP_HOST}/api/easy-site-profiles/${SITE_ID}" \
  -H "Content-Type: application/json" \
  -d '{"front_service":{"mtls_enabled":false,"mtls_client_ca_ref":""}}' > /dev/null
echo "Cleanup done."
echo ""
echo "=== mTLS client check complete ==="
