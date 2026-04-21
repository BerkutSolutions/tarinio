#!/bin/sh
set -eu

BASE_URL="${BASE_URL:-http://api-lb:8080}"
USERNAME="${USERNAME:-${CONTROL_PLANE_BOOTSTRAP_ADMIN_USERNAME:-admin}}"
PASSWORD="${PASSWORD:-${CONTROL_PLANE_BOOTSTRAP_ADMIN_PASSWORD:-admin}}"
SITE_COUNT="${SITE_COUNT:-20}"
DOMAIN_SUFFIX="${DOMAIN_SUFFIX:-ha.local}"
UPSTREAM_HOST="${UPSTREAM_HOST:-demo-app}"
UPSTREAM_PORT="${UPSTREAM_PORT:-80}"
COOKIE_JAR="$(mktemp)"
WORKDIR="$(mktemp -d)"
trap 'rm -f "$COOKIE_JAR"; rm -rf "$WORKDIR"' EXIT

curl -fsS -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}" \
  "${BASE_URL}/api/auth/login" >/dev/null

wait_for_site() {
  target_id="$1"
  attempts="${2:-15}"
  current=1
  while [ "$current" -le "$attempts" ]; do
    if curl -fsS -b "$COOKIE_JAR" "${BASE_URL}/api/sites" | jq -e --arg id "$target_id" '.[] | select(.id == $id)' >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    current=$((current + 1))
  done
  return 1
}

site_exists() {
  target_id="$1"
  curl -fsS -b "$COOKIE_JAR" "${BASE_URL}/api/sites" | jq -e --arg id "$target_id" '.[] | select(.id == $id)' >/dev/null 2>&1
}

upstream_exists() {
  target_id="$1"
  curl -fsS -b "$COOKIE_JAR" "${BASE_URL}/api/upstreams" | jq -e --arg id "$target_id" '.[] | select(.id == $id)' >/dev/null 2>&1
}

rate_policy_exists() {
  target_id="$1"
  curl -fsS -b "$COOKIE_JAR" "${BASE_URL}/api/rate-limit-policies" | jq -e --arg id "$target_id" '.[] | select(.id == $id)' >/dev/null 2>&1
}

echo "Provisioning ${SITE_COUNT} HA lab services against ${BASE_URL}"

i=1
while [ "$i" -le "$SITE_COUNT" ]; do
  site_id="$(printf 'tenant-%02d' "$i")"
  host_name="${site_id}.${DOMAIN_SUFFIX}"
  upstream_id="${site_id}-upstream"

  cat >"${WORKDIR}/site.json" <<EOF
{"id":"${site_id}","primary_host":"${host_name}","enabled":true}
EOF
  site_attempt=1
  while ! site_exists "$site_id"; do
    if curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
      -X POST "${BASE_URL}/api/sites" \
      --data-binary @"${WORKDIR}/site.json" >/dev/null 2>&1; then
      :
    fi
    if wait_for_site "$site_id" 2; then
      break
    fi
    if [ "$site_attempt" -ge 6 ]; then
      echo "site ${site_id} was not persisted after retries" >&2
      exit 1
    fi
    echo "  site ${site_id}: retry ${site_attempt}"
    site_attempt=$((site_attempt + 1))
    sleep 1
  done
  echo "  site ${site_id}: ready"

  cat >"${WORKDIR}/upstream.json" <<EOF
{"id":"${upstream_id}","site_id":"${site_id}","host":"${UPSTREAM_HOST}","port":${UPSTREAM_PORT},"scheme":"http"}
EOF
  upstream_attempt=1
  while ! upstream_exists "$upstream_id"; do
    if curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
      -X POST "${BASE_URL}/api/upstreams" \
      --data-binary @"${WORKDIR}/upstream.json" >/dev/null 2>&1; then
      :
    fi
    if upstream_exists "$upstream_id"; then
      break
    fi
    if [ "$upstream_attempt" -ge 6 ]; then
      echo "upstream ${upstream_id} was not persisted after retries" >&2
      exit 1
    fi
    echo "  upstream ${upstream_id}: retry ${upstream_attempt}"
    upstream_attempt=$((upstream_attempt + 1))
    sleep 1
  done
  echo "  upstream ${upstream_id}: ready"
  i=$((i + 1))
done

cat >"${WORKDIR}/rate-limit.json" <<EOF
{"id":"tenant-01-rate","site_id":"tenant-01","enabled":true,"limits":{"requests_per_second":5,"burst":10}}
EOF
echo "  rate-limit tenant-01-rate: applying"
if rate_policy_exists "tenant-01-rate"; then
  echo "  rate-limit tenant-01-rate: already present"
else
  curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
    -X POST "${BASE_URL}/api/rate-limit-policies" \
    --data-binary @"${WORKDIR}/rate-limit.json" >/dev/null
fi

cat >"${WORKDIR}/antiddos.json" <<EOF
{
  "use_l4_guard": true,
  "chain_mode": "input",
  "conn_limit": 120,
  "rate_per_second": 240,
  "rate_burst": 480,
  "ports": [80, 443],
  "target": "DROP",
  "destination_ip": "",
  "model_enabled": true,
  "enforce_l7_rate_limit": true,
  "l7_requests_per_second": 15,
  "l7_burst": 30,
  "l7_status_code": 429,
  "model_poll_interval_seconds": 2,
  "model_decay_lambda": 0.08,
  "model_throttle_threshold": 2.5,
  "model_drop_threshold": 6.0,
  "model_hold_seconds": 60,
  "model_throttle_rate_per_second": 3,
  "model_throttle_burst": 6,
  "model_throttle_target": "REJECT",
  "model_weight_429": 1.0,
  "model_weight_403": 1.8,
  "model_weight_444": 2.2,
  "model_emergency_rps": 180,
  "model_emergency_unique_ips": 40,
  "model_emergency_per_ip_rps": 60,
  "model_weight_emergency_botnet": 6.0,
  "model_weight_emergency_single": 4.0
}
EOF
echo "  anti-ddos settings: applying"
curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -H 'X-WAF-Auto-Apply-Disabled: true' \
  -X PUT "${BASE_URL}/api/anti-ddos/settings" \
  --data-binary @"${WORKDIR}/antiddos.json" >/dev/null

echo "  revision: compile"
compile_response="$(curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' -X POST "${BASE_URL}/api/revisions/compile" -d '{}')"
revision_id="$(printf '%s' "$compile_response" | jq -r '.revision.id')"
if [ -z "$revision_id" ] || [ "$revision_id" = "null" ]; then
  echo "failed to determine revision id from compile response" >&2
  exit 1
fi
echo "  revision: apply ${revision_id}"
curl -fsS -b "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST "${BASE_URL}/api/revisions/${revision_id}/apply" -d '{}' >/dev/null

echo
echo "Provisioning completed."
echo "  Services: ${SITE_COUNT}"
echo "  Rate-limited site: tenant-01.${DOMAIN_SUFFIX}"
echo "  Revision applied: ${revision_id}"
