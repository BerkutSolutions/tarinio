#!/bin/sh
set -eu

TARGET_URL="${TARGET_URL:-http://runtime/limited.html}"
TARGET_HOST="${TARGET_HOST:-tenant-01.ha.local}"
CONCURRENCY="${CONCURRENCY:-8}"
REQUESTS_PER_WORKER="${REQUESTS_PER_WORKER:-30}"

echo "Mini DDoS simulation"
echo "  url=${TARGET_URL}"
echo "  host=${TARGET_HOST}"
echo "  concurrency=${CONCURRENCY}"
echo "  requests_per_worker=${REQUESTS_PER_WORKER}"

worker() {
  count=1
  while [ "$count" -le "$REQUESTS_PER_WORKER" ]; do
    curl -k -s -o /dev/null -w '%{http_code}\n' -H "Host: ${TARGET_HOST}" "${TARGET_URL}" || true
    count=$((count + 1))
  done
}

i=1
while [ "$i" -le "$CONCURRENCY" ]; do
  worker &
  i=$((i + 1))
done
wait

echo "Mini DDoS simulation finished."
