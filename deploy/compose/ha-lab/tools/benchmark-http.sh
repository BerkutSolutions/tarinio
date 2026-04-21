#!/bin/sh
set -eu

NAME="${NAME:-scenario}"
URL="${URL:?URL is required}"
HOST_HEADER="${HOST_HEADER:-}"
CONCURRENCY="${CONCURRENCY:-4}"
REQUESTS_PER_WORKER="${REQUESTS_PER_WORKER:-25}"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

seq 1 "$CONCURRENCY" | xargs -I{} -P "$CONCURRENCY" sh -c '
  set -eu
  URL="$1"
  HOST_HEADER="$2"
  REQUESTS_PER_WORKER="$3"
  WORKDIR="$4"
  worker_id="$5"
  raw_file="${WORKDIR}/worker-${worker_id}.jsonl"
  i=1
  while [ "$i" -le "$REQUESTS_PER_WORKER" ]; do
    if [ -n "$HOST_HEADER" ]; then
      result="$(curl -sS --max-time 5 -o /dev/null -w "%{http_code} %{time_total}" -H "Host: ${HOST_HEADER}" "$URL" 2>/dev/null || printf "000 0")"
    else
      result="$(curl -sS --max-time 5 -o /dev/null -w "%{http_code} %{time_total}" "$URL" 2>/dev/null || printf "000 0")"
    fi
    status="$(printf "%s" "$result" | awk "{print \$1}")"
    duration_s="$(printf "%s" "$result" | awk "{print \$2}")"
    duration_ms="$(awk "BEGIN { printf \"%.3f\", (${duration_s:-0} * 1000) }")"
    printf "{\"status\":\"%s\",\"duration_ms\":%s}\n" "${status:-000}" "${duration_ms:-0}" >>"$raw_file"
    i=$((i + 1))
  done
' sh "$URL" "$HOST_HEADER" "$REQUESTS_PER_WORKER" "$WORKDIR" {}

cat "${WORKDIR}"/worker-*.jsonl >"${WORKDIR}/all.jsonl"

jq -s --arg name "$NAME" --arg url "$URL" --arg host "$HOST_HEADER" '
  def percentile(p):
    if length == 0 then 0
    else (sort | .[((length * p) | ceil) - 1])
    end;
  {
    scenario: $name,
    url: $url,
    host_header: $host,
    total_requests: length,
    status_counts: (group_by(.status) | map({key: (.[0].status|tostring), value: length}) | from_entries),
    avg_duration_ms: (if length == 0 then 0 else (map(.duration_ms) | add / length) end),
    min_duration_ms: (if length == 0 then 0 else (map(.duration_ms) | min) end),
    max_duration_ms: (if length == 0 then 0 else (map(.duration_ms) | max) end),
    p50_duration_ms: (map(.duration_ms) | percentile(0.50)),
    p95_duration_ms: (map(.duration_ms) | percentile(0.95)),
    p99_duration_ms: (map(.duration_ms) | percentile(0.99))
  }
' "${WORKDIR}/all.jsonl"
