#!/usr/bin/env sh
# Runs OWASP ZAP only against the disposable Docker E2E runtime.
set -eu

MODE="${1:-baseline}"
case "$MODE" in baseline|full) ;; *) echo "usage: $0 [baseline|full]" >&2; exit 2;; esac

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${DAST_OUTPUT_DIR:-$ROOT/build/dast/$MODE}"
ZAP_IMAGE="${ZAP_IMAGE:-ghcr.io/zaproxy/zaproxy:stable}"
TARGET="${DAST_TARGET_URL:-http://127.0.0.1:10080}"
HOST="${DAST_TARGET_HOST:-e2e-management.test}"
mkdir -p "$OUT"

cleanup() {
  docker compose -f "$ROOT/deploy/compose/e2e/docker-compose.yml" down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

# Reuse the real bootstrap, compile/apply and readiness path used by E2E.
E2E_KEEP_STACK=1 E2E_FILTER=TestE2ESmoke_LoginHealthcheckDashboard \
  E2E_LOG_DIR="$OUT" E2E_EVIDENCE_DIR="$OUT" sh "$ROOT/scripts/run-e2e-tests.sh" "$ROOT"

scan="zap-baseline.py"
[ "$MODE" = "full" ] && scan="zap-full-scan.py"

# The explicit Host replacement reaches the configured WAF virtual host while
# Docker host networking keeps the scanner isolated from every non-E2E network.
docker run --rm --network host -v "$OUT:/zap/wrk:rw" "$ZAP_IMAGE" \
  "$scan" -t "$TARGET" -m 3 -I \
  -r report.html -J report.json -w report.md -x report.xml \
  -z "-config replacer.full_list(0).description=E2EHost -config replacer.full_list(0).enabled=true -config replacer.full_list(0).matchtype=REQ_HEADER -config replacer.full_list(0).matchstr=Host -config replacer.full_list(0).regex=false -config replacer.full_list(0).replacement=$HOST"

python3 "$ROOT/scripts/write-dast-evidence-report.py" --input "$OUT/report.json" --output-dir "$OUT" --mode "$MODE" --max-risk 3
