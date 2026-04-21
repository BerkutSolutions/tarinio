#!/bin/sh
set -eu

BASE_API_URL="${BASE_API_URL:-http://api-lb:8080/healthz}"
BASE_RUNTIME_URL="${BASE_RUNTIME_URL:-http://runtime}"
BASELINE_CONCURRENCY="${BASELINE_CONCURRENCY:-2}"
BASELINE_REQUESTS_PER_WORKER="${BASELINE_REQUESTS_PER_WORKER:-20}"
RATE_LIMIT_CONCURRENCY="${RATE_LIMIT_CONCURRENCY:-4}"
RATE_LIMIT_REQUESTS_PER_WORKER="${RATE_LIMIT_REQUESTS_PER_WORKER:-20}"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

BASELINE_JSON="$(NAME=baseline-public URL="${BASE_RUNTIME_URL}/index.html" HOST_HEADER="tenant-02.ha.local" CONCURRENCY="$BASELINE_CONCURRENCY" REQUESTS_PER_WORKER="$BASELINE_REQUESTS_PER_WORKER" /tools/benchmark-http.sh)"
RATE_LIMIT_JSON="$(NAME=rate-limit-protected URL="${BASE_RUNTIME_URL}/limited.html" HOST_HEADER="tenant-01.ha.local" CONCURRENCY="$RATE_LIMIT_CONCURRENCY" REQUESTS_PER_WORKER="$RATE_LIMIT_REQUESTS_PER_WORKER" /tools/benchmark-http.sh)"
API_JSON="$(NAME=api-health URL="$BASE_API_URL" HOST_HEADER="" CONCURRENCY="4" REQUESTS_PER_WORKER="20" /tools/benchmark-http.sh)"

printf '%s\n' "$BASELINE_JSON" >"${WORKDIR}/baseline.json"
printf '%s\n' "$RATE_LIMIT_JSON" >"${WORKDIR}/rate-limit.json"
printf '%s\n' "$API_JSON" >"${WORKDIR}/api.json"

PROM_QUERY_RESULT='{}'
if curl -fsS http://prometheus:9090/api/v1/query --get --data-urlencode 'query=up' >/dev/null 2>&1; then
  PROM_QUERY_RESULT="$(curl -fsS http://prometheus:9090/api/v1/query --get --data-urlencode 'query=up')"
fi

jq -n \
  --slurpfile baseline "${WORKDIR}/baseline.json" \
  --slurpfile rate_limit "${WORKDIR}/rate-limit.json" \
  --slurpfile api "${WORKDIR}/api.json" \
  --arg generated_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --argjson prometheus_up "${PROM_QUERY_RESULT}" '
  {
    generated_at: $generated_at,
    scenarios: {
      baseline_public: $baseline[0],
      rate_limit_protected: $rate_limit[0],
      api_health: $api[0]
    },
    observations: {
      rate_limit_triggered: (($rate_limit[0].status_counts["429"] // 0) > 0),
      api_health_failures: ($api[0].status_counts["000"] // 0)
    },
    prometheus_up_snapshot: $prometheus_up
  }
'
