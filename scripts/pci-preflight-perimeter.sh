#!/usr/bin/env bash

if [[ -z "${BASH_VERSION:-}" ]]; then
  if command -v bash >/dev/null 2>&1; then
    exec bash "$0" "$@"
  fi
  echo "This script requires bash."
  exit 1
fi

set -u

# PCI-like external perimeter preflight.
# Produces pass/fail summary and evidence artifacts.
#
# Optional env:
#   TARGET_HOST=localhost
#   TARGET_HTTP_PORT=80
#   TARGET_HTTPS_PORT=443
#   TARGET_RESOLVE_IP=127.0.0.1
#   EXPECTED_OPEN_PORTS="80,443"
#   PORT_PROBE_SET="22,80,443,8080,8443,9200,5432,6379"
#   EXPECTED_TCP_TIMESTAMPS=0
#   EXPECT_HSTS_PRELOAD=0
#   COMPLIANCE_POLICY_FILE=security/compliance/deprecated-controls-policy.json
#   OUT_BASE_DIR=/tmp

TARGET_HOST="${TARGET_HOST:-localhost}"
TARGET_HTTP_PORT="${TARGET_HTTP_PORT:-80}"
TARGET_HTTPS_PORT="${TARGET_HTTPS_PORT:-443}"
TARGET_RESOLVE_IP="${TARGET_RESOLVE_IP:-}"
EXPECTED_OPEN_PORTS="${EXPECTED_OPEN_PORTS:-80,443}"
PORT_PROBE_SET="${PORT_PROBE_SET:-22,80,443,8080,8443,9200,5432,6379}"
EXPECTED_TCP_TIMESTAMPS="${EXPECTED_TCP_TIMESTAMPS:-0}"
EXPECT_HSTS_PRELOAD="${EXPECT_HSTS_PRELOAD:-0}"
COMPLIANCE_POLICY_FILE="${COMPLIANCE_POLICY_FILE:-security/compliance/deprecated-controls-policy.json}"
OUT_BASE_DIR="${OUT_BASE_DIR:-/tmp}"

TS="$(date +%Y%m%d_%H%M%S)"
OUT="${OUT_BASE_DIR%/}/pci-preflight-${TS}"
mkdir -p "$OUT"

PASS=true

csv_to_lines() {
  printf "%s" "$1" | tr ',' ' ' | awk '{$1=$1; print}'
}

http_head() {
  local url="$1"
  if [[ -n "$TARGET_RESOLVE_IP" ]]; then
    curl -k -sS -I --max-time 8 --resolve "${TARGET_HOST}:${TARGET_HTTPS_PORT}:${TARGET_RESOLVE_IP}" "$url"
  else
    curl -k -sS -I --max-time 8 "$url"
  fi
}

tls_handshake_ok() {
  local mode="$1"
  local extra=()
  if [[ -n "$TARGET_RESOLVE_IP" ]]; then
    extra+=("-servername" "$TARGET_HOST")
  fi
  if timeout 10 openssl s_client -connect "${TARGET_HOST}:${TARGET_HTTPS_PORT}" "${extra[@]}" "$mode" </dev/null > /dev/null 2>&1; then
    return 0
  fi
  return 1
}

tcp_open() {
  local host="$1"
  local port="$2"
  if timeout 2 bash -c ">/dev/tcp/${host}/${port}" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

mark_fail() {
  PASS=false
}

{
  echo "target_host=$TARGET_HOST"
  echo "target_http_port=$TARGET_HTTP_PORT"
  echo "target_https_port=$TARGET_HTTPS_PORT"
  echo "target_resolve_ip=${TARGET_RESOLVE_IP:-none}"
  echo "expected_open_ports=$EXPECTED_OPEN_PORTS"
  echo "port_probe_set=$PORT_PROBE_SET"
  echo "expected_tcp_timestamps=$EXPECTED_TCP_TIMESTAMPS"
  echo "expect_hsts_preload=$EXPECT_HSTS_PRELOAD"
  echo "compliance_policy_file=$COMPLIANCE_POLICY_FILE"
} > "$OUT/context.txt"

# 1) Port surface checks
expected_open_norm="$(csv_to_lines "$EXPECTED_OPEN_PORTS")"
probe_ports_norm="$(csv_to_lines "$PORT_PROBE_SET")"
{
  echo "# Port checks"
  echo "Expected open ports: $expected_open_norm"
  echo "Probed ports: $probe_ports_norm"
  echo
  for p in $probe_ports_norm; do
    if tcp_open "$TARGET_HOST" "$p"; then
      echo "open:$p"
    else
      echo "closed:$p"
    fi
  done
} > "$OUT/ports-check.txt"

for p in $expected_open_norm; do
  if ! grep -q "^open:${p}$" "$OUT/ports-check.txt"; then
    echo "missing_expected_open_port:$p" >> "$OUT/ports-check.txt"
    mark_fail
  fi
done

for p in $probe_ports_norm; do
  if grep -q "^open:${p}$" "$OUT/ports-check.txt" && ! grep -Eq "(^| )${p}($| )" <<< "$expected_open_norm"; then
    echo "unexpected_open_port:$p" >> "$OUT/ports-check.txt"
    mark_fail
  fi
done

# 2) TCP timestamps check
runtime_tcp_ts="unavailable"
if [[ -r /proc/sys/net/ipv4/tcp_timestamps ]]; then
  runtime_tcp_ts="$(cat /proc/sys/net/ipv4/tcp_timestamps 2>/dev/null || true)"
fi
{
  echo "# TCP timestamps"
  echo "expected=$EXPECTED_TCP_TIMESTAMPS"
  echo "actual=$runtime_tcp_ts"
} > "$OUT/tcp-timestamps-check.txt"
if [[ "$runtime_tcp_ts" != "$EXPECTED_TCP_TIMESTAMPS" ]]; then
  echo "tcp_timestamps_mismatch" >> "$OUT/tcp-timestamps-check.txt"
  mark_fail
fi

# 3) TLS protocols and weak cipher negative tests
tls12_ok=false
tls13_ok=false
tls10_blocked=false
tls11_blocked=false
weak_cipher_blocked=false

if tls_handshake_ok "-tls1_2"; then tls12_ok=true; else mark_fail; fi
if tls_handshake_ok "-tls1_3"; then tls13_ok=true; else mark_fail; fi
if tls_handshake_ok "-tls1"; then tls10_blocked=false; mark_fail; else tls10_blocked=true; fi
if tls_handshake_ok "-tls1_1"; then tls11_blocked=false; mark_fail; else tls11_blocked=true; fi

if timeout 10 openssl s_client -connect "${TARGET_HOST}:${TARGET_HTTPS_PORT}" -cipher AES128-SHA -tls1_2 </dev/null >/dev/null 2>&1; then
  weak_cipher_blocked=false
  mark_fail
else
  weak_cipher_blocked=true
fi

{
  echo "# TLS checks"
  echo "tls1_2_ok=$tls12_ok"
  echo "tls1_3_ok=$tls13_ok"
  echo "tls1_0_blocked=$tls10_blocked"
  echo "tls1_1_blocked=$tls11_blocked"
  echo "weak_cipher_blocked=$weak_cipher_blocked"
} > "$OUT/tls-check.txt"

# 4) Header checks (HSTS)
hsts_ok=false
hsts_preload_ok=false
hsts_raw_file="$OUT/hsts-headers.txt"
if http_head "https://${TARGET_HOST}:${TARGET_HTTPS_PORT}/" > "$hsts_raw_file" 2>&1; then
  if grep -Eiq '^strict-transport-security:' "$hsts_raw_file"; then
    if grep -Eiq '^strict-transport-security:.*max-age=[0-9]+' "$hsts_raw_file"; then
      hsts_ok=true
    fi
    if [[ "$EXPECT_HSTS_PRELOAD" == "1" ]]; then
      if grep -Eiq '^strict-transport-security:.*preload' "$hsts_raw_file"; then
        hsts_preload_ok=true
      else
        mark_fail
      fi
    else
      hsts_preload_ok=true
    fi
  else
    mark_fail
  fi
else
  mark_fail
fi
if [[ "$hsts_ok" != "true" ]]; then
  mark_fail
fi

{
  echo "# Header checks"
  echo "hsts_present_with_max_age=$hsts_ok"
  echo "hsts_preload_check=$hsts_preload_ok"
} > "$OUT/headers-check.txt"

# 5) Compliance policy for deprecated checks
policy_ok=false
{
  echo "# Deprecated control policy checks"
  echo "policy_file=$COMPLIANCE_POLICY_FILE"
  if [[ -f "$COMPLIANCE_POLICY_FILE" ]]; then
    cp "$COMPLIANCE_POLICY_FILE" "$OUT/deprecated-controls-policy.json"
    if grep -q '"control_id"[[:space:]]*:[[:space:]]*"HPKP_HEADER"' "$COMPLIANCE_POLICY_FILE" \
      && grep -q '"status"[[:space:]]*:[[:space:]]*"deprecated_not_applicable"' "$COMPLIANCE_POLICY_FILE"; then
      echo "hpkp_policy_present=true"
      policy_ok=true
    else
      echo "hpkp_policy_present=false"
    fi
  else
    echo "policy_file_missing=true"
  fi
} > "$OUT/deprecated-controls-check.txt"
if [[ "$policy_ok" != "true" ]]; then
  mark_fail
fi

overall="pass"
if [[ "$PASS" != "true" ]]; then
  overall="fail"
fi

cat > "$OUT/summary.json" <<EOF
{
  "generated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "target_host": "$TARGET_HOST",
  "target_http_port": $TARGET_HTTP_PORT,
  "target_https_port": $TARGET_HTTPS_PORT,
  "overall": "$overall",
  "checks": {
    "ports": "$(if grep -q 'unexpected_open_port\|missing_expected_open_port' "$OUT/ports-check.txt"; then echo "fail"; else echo "pass"; fi)",
    "tcp_timestamps": "$(if grep -q 'tcp_timestamps_mismatch' "$OUT/tcp-timestamps-check.txt"; then echo "fail"; else echo "pass"; fi)",
    "tls": "$(if [[ "$tls12_ok" == "true" && "$tls13_ok" == "true" && "$tls10_blocked" == "true" && "$tls11_blocked" == "true" && "$weak_cipher_blocked" == "true" ]]; then echo "pass"; else echo "fail"; fi)",
    "headers_hsts": "$(if [[ "$hsts_ok" == "true" && "$hsts_preload_ok" == "true" ]]; then echo "pass"; else echo "fail"; fi)",
    "deprecated_controls_policy": "$(if [[ "$policy_ok" == "true" ]]; then echo "pass"; else echo "fail"; fi)"
  }
}
EOF

{
  echo "PCI perimeter preflight result: $overall"
  echo "artifact_dir=$OUT"
  echo
  echo "Artifacts:"
  echo "  context.txt"
  echo "  ports-check.txt"
  echo "  tcp-timestamps-check.txt"
  echo "  tls-check.txt"
  echo "  hsts-headers.txt"
  echo "  headers-check.txt"
  echo "  deprecated-controls-check.txt"
  echo "  summary.json"
} > "$OUT/summary.txt"

echo "$OUT" > "$OUT_BASE_DIR/.last-pci-preflight-artifact"
cat "$OUT/summary.txt"

if [[ "$overall" != "pass" ]]; then
  exit 1
fi
