#!/usr/bin/env sh
# Runs OWASP ZAP only against the disposable Docker E2E runtime.
set -eu

MODE="${1:-baseline}"
case "$MODE" in baseline|full) ;; *) echo "usage: $0 [baseline|full]" >&2; exit 2;; esac

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${DAST_OUTPUT_DIR:-$ROOT/build/dast/$MODE}"
ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"
TARGET="${DAST_SCANNER_TARGET_URL:-http://e2e-management.test}"
HOST="${DAST_TARGET_HOST:-e2e-management.test}"
E2E_PROJECT="${E2E_PROJECT:-waf-dast-$MODE}"
if [ -n "${CI_CONCURRENT_ID:-}" ]; then
  case "$CI_CONCURRENT_ID" in
    *[!0-9]*|'') echo "[dast] ERROR: CI_CONCURRENT_ID must be numeric" >&2; exit 1 ;;
  esac
  E2E_CI_RUNNER_TOKEN="$(printf '%s' "${CI_RUNNER_SHORT_TOKEN:-runner}" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9_-' '-')"
  E2E_PROJECT="ci-${E2E_CI_RUNNER_TOKEN}-e2e-slot-${CI_CONCURRENT_ID}"
fi
mkdir -p "$OUT"
ZAP_DOCKER_NETWORK="${E2E_PROJECT}_waf-e2e-net"
zap_cidfile="$OUT/.zap-container-id"
rm -f "$zap_cidfile"

cleanup() {
  if [ -s "$zap_cidfile" ]; then
    docker rm -f "$(cat "$zap_cidfile")" >/dev/null 2>&1 || true
  fi
  rm -f "$zap_cidfile"
  COMPOSE_PROJECT_NAME="$E2E_PROJECT" docker compose -f "$ROOT/deploy/compose/e2e/docker-compose.yml" down --volumes --remove-orphans --rmi local >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

# Reuse the real bootstrap, compile/apply and readiness path used by E2E.
E2E_PROJECT="$E2E_PROJECT" E2E_KEEP_STACK=1 E2E_FILTER=TestE2ESmoke_LoginHealthcheckDashboard \
  E2E_LOG_DIR="$OUT" E2E_EVIDENCE_DIR="$OUT" sh "$ROOT/scripts/run-e2e-tests.sh" "$ROOT"

scan="zap-baseline.py"
scan_timeout="${DAST_SCAN_TIMEOUT_SECONDS:-1200}"
if [ "$MODE" = "full" ]; then
  scan="zap-full-scan.py"
  scan_timeout="${DAST_SCAN_TIMEOUT_SECONDS:-2700}"
fi

# The scanner joins only this disposable stack's Compose network. The explicit
# Host replacement still exercises the configured WAF virtual host.
zap_user_args=""
case "$(uname -s)" in
  Linux) zap_user_args="--user $(id -u):$(id -g)" ;;
esac
MSYS_NO_PATHCONV=1 timeout --preserve-status -k 30s "${scan_timeout}s" \
  docker run --rm --cidfile "$zap_cidfile" --network "$ZAP_DOCKER_NETWORK" $zap_user_args -e HOME=/tmp \
  -e JAVA_TOOL_OPTIONS=-Djava.util.prefs.userRoot=/tmp/zap-java-prefs -w /zap/wrk \
  -v "$OUT:/zap/wrk:rw" "$ZAP_IMAGE" \
  "$scan" --autooff -t "$TARGET" -m 3 -I \
  -r report.html -J report.json -w report.md -x report.xml \
  -z "-silent -dir /tmp/zap-home -config replacer.full_list(0).description=E2EHost -config replacer.full_list(0).enabled=true -config replacer.full_list(0).matchtype=REQ_HEADER -config replacer.full_list(0).matchstr=Host -config replacer.full_list(0).regex=false -config replacer.full_list(0).replacement=$HOST"

python3 "$ROOT/scripts/write-dast-evidence-report.py" --input "$OUT/report.json" --output-dir "$OUT" --mode "$MODE" --max-risk 3 --policy "$ROOT/scripts/dast-baseline-policy.json"
