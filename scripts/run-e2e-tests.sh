#!/usr/bin/env sh
# run-e2e-tests.sh - spin up isolated e2e stack, run ui/tests e2e suites, tear down.
# Exits non-zero on any failure. Containers + volumes removed after run.
#
# Usage:  sh scripts/run-e2e-tests.sh [/path/to/repo]
#
# Env overrides:
#   E2E_PORT          host port for control-plane (default: 18080)
#   E2E_RT_PORT       host port for runtime HTTP (default: 10080)
#   E2E_RT_HTTPS_PORT host port for runtime HTTPS (default: 10443)
#   E2E_RT_HLT_PORT   host port for runtime health (default: 18081)
#   E2E_USER          admin username (default: e2e-admin)
#   E2E_PASS          admin credential (default: e2e-password-1234)
#   E2E_TIMEOUT       seconds to wait for healthcheck (default: 180)
#   E2E_FILTER        go test -run filter (default: TestE2E)
#   COMPOSE_CMD       docker compose command (auto-detected)
#   GO_CMD            go binary (default: go)
#   E2E_KEEP_STACK    set to 1 to skip teardown (debug)

set -eu

REPO_ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
E2E_COMPOSE_DIR="$REPO_ROOT/deploy/compose/e2e"
E2E_PORT="${E2E_PORT:-18080}"
E2E_RT_PORT="${E2E_RT_PORT:-10080}"
E2E_RT_HTTPS_PORT="${E2E_RT_HTTPS_PORT:-10443}"
E2E_RT_HLT_PORT="${E2E_RT_HLT_PORT:-18081}"
E2E_USER="${E2E_USER:-e2e-admin}"
E2E_PASS="${E2E_PASS:-e2e-password-1234}"
E2E_TIMEOUT="${E2E_TIMEOUT:-180}"
E2E_FILTER="${E2E_FILTER:-TestE2E}"
E2E_KEEP_STACK="${E2E_KEEP_STACK:-0}"
GO_CMD="${GO_CMD:-go}"
E2E_LOG_DIR="${E2E_LOG_DIR:-$REPO_ROOT/.work/logs}"
mkdir -p "$E2E_LOG_DIR"
E2E_LOG_FILE="$E2E_LOG_DIR/e2e-$(date +%Y%m%d_%H%M%S).log"

if [ -t 1 ]; then
  C_RESET="$(printf '\033[0m')"
  C_GREEN="$(printf '\033[32m')"
  C_RED="$(printf '\033[31m')"
  C_YELLOW="$(printf '\033[33m')"
  C_CYAN="$(printf '\033[36m')"
  C_GRAY="$(printf '\033[90m')"
else
  C_RESET=""; C_GREEN=""; C_RED=""; C_YELLOW=""; C_CYAN=""; C_GRAY=""
fi

step() { printf "%s[e2e] [RUN] %s%s\n" "$C_CYAN" "$1" "$C_RESET"; }
ok() { printf "%s[e2e] [OK] %s%s\n" "$C_GREEN" "$1" "$C_RESET"; }
warn() { printf "%s[e2e] [WARN] %s%s\n" "$C_YELLOW" "$1" "$C_RESET"; }
fail_msg() { printf "%s[e2e] [FAIL] %s%s\n" "$C_RED" "$1" "$C_RESET" >&2; }
info() { printf "%s[e2e]   %s%s\n" "$C_GRAY" "$1" "$C_RESET"; }

show_log_tail() {
  file="$1"
  lines="${2:-80}"
  [ -f "$file" ] || return 0
  printf "%s" "$C_RED" >&2
  tail -n "$lines" "$file" >&2 || true
  printf "%s" "$C_RESET" >&2
}

run_quiet() {
  label="$1"
  shift
  step "$label"
  if "$@" >>"$E2E_LOG_FILE" 2>&1; then
    ok "$label"
    return 0
  fi
  fail_msg "$label failed (log: $E2E_LOG_FILE)"
  show_log_tail "$E2E_LOG_FILE" 120
  return 1
}

summarize_go_e2e_json() {
  file="$1"
  py=""
  if command -v python3 >/dev/null 2>&1; then
    py="python3"
  elif command -v python >/dev/null 2>&1; then
    py="python"
  fi
  if [ -z "$py" ]; then
    grep '"Action":"\(pass\|fail\)"' "$file" || true
    return 0
  fi
  E2E_SUMMARY_FILE="$file" E2E_C_GREEN="$C_GREEN" E2E_C_RED="$C_RED" E2E_C_RESET="$C_RESET" "$py" - <<'PY'
import json, os, re, sys

path = os.environ["E2E_SUMMARY_FILE"]
green = os.environ.get("E2E_C_GREEN", "")
red = os.environ.get("E2E_C_RED", "")
reset = os.environ.get("E2E_C_RESET", "")

ok_marker = "[OK]"
fail_marker = "[FAIL]"
skip_marker = "[SKIP]"

def expectation(name: str) -> str:
    leaf = name.split("/")[-1]
    rules = [
        (r"MTLS_IncomingClientCert_Required", "HTTPS mTLS rejects without client cert, passes with client cert, passes after disabled"),
        (r"Returns403|Blocks_.*|Blocks_Without", "HTTP 403"),
        (r"Returns429|RateLimit_Burst|CustomLimitRules", "HTTP 429"),
        (r"Gets302|Challenge", "HTTP 302/challenge flow"),
        (r"Returns451", "HTTP 451"),
        (r"Geo", "HTTP 403"),
        (r"Allows|Passes|Bypasses|Recovery", "allowed response (not blocked)"),
        (r"BrandedHTML", "HTTP 403 + branded HTML"),
        (r"Headers|HSTS|CookieFlags", "expected headers present"),
        (r"VirtualPatches", "virtual patch block"),
        (r"Config|Parsing|WebSocket|JA3", "compiled/runtime config present"),
    ]
    for pattern, value in rules:
        if re.search(pattern, leaf):
            return value
    return "expected behavior"

def actual_for(status: str, expected: str) -> str:
    if status == "pass":
        return expected
    if status == "skip":
        return "skipped"
    return "failed"

def display_name(name: str) -> str:
    leaf = name.split("/")[-1]
    if leaf.startswith("TestE2E"):
        leaf = leaf[len("TestE2E"):]
    elif leaf.startswith("Test"):
        leaf = leaf[len("Test"):]
    return leaf.lstrip("_") or name

def suite_name(name: str) -> str:
    return f"{display_name(name)} suite"

seen = {}
order = []
outputs = {}
parents = set()
for line in open(path, encoding="utf-8", errors="replace"):
    try:
        item = json.loads(line)
    except Exception:
        continue
    test = item.get("Test") or ""
    if not test.startswith("TestE2E"):
        continue
    if "/" in test:
        parents.add(test.split("/", 1)[0])
    action = item.get("Action")
    if action == "run":
        if test not in seen:
            order.append(test)
        seen[test] = "run"
    elif action in ("pass", "fail", "skip"):
        seen[test] = action
    elif action == "output":
        outputs.setdefault(test, []).append(item.get("Output", "").strip())

for test in order:
    status = seen.get(test, "run")
    expected = expectation(test)
    actual = actual_for(status, expected)
    name = display_name(test)
    if status == "pass":
        if "/" not in test and test in parents:
            print(f"{green}[e2e] {ok_marker} {suite_name(test)} completed{reset}")
        else:
            print(f"{green}[e2e] {ok_marker} {name}: expected={expected}; actual={actual}{reset}")
    elif status == "skip":
        if "/" not in test and test in parents:
            print(f"[e2e] {skip_marker} {suite_name(test)} skipped")
        else:
            print(f"[e2e] {skip_marker} {name}: expected={expected}; actual={actual}")
    elif status == "fail":
        if "/" not in test and test in parents:
            print(f"{red}[e2e] {fail_marker} {suite_name(test)} failed{reset}")
        else:
            print(f"{red}[e2e] {fail_marker} {name}: expected={expected}; actual={actual}{reset}")
        for out in outputs.get(test, [])[-8:]:
            if out:
                print(f"{red}[e2e]   {out}{reset}")
PY
}

run_go_e2e_stream() {
  py=""
  if command -v python3 >/dev/null 2>&1; then
    py="python3"
  elif command -v python >/dev/null 2>&1; then
    py="python"
  fi
  if [ -z "$py" ]; then
    fail_msg "python is required for live e2e output"
    return 1
  fi
  E2E_TEST_LOG="$TEST_LOG" E2E_SUMMARY_OUT="${E2E_SUMMARY_OUT:-}" E2E_C_GREEN="$C_GREEN" E2E_C_RED="$C_RED" E2E_C_YELLOW="$C_YELLOW" E2E_C_CYAN="$C_CYAN" E2E_C_RESET="$C_RESET" "$py" - <<'PY'
import json, os, re, subprocess, sys

log_path = os.environ["E2E_TEST_LOG"]
summary_out = os.environ.get("E2E_SUMMARY_OUT", "")
go_cmd = os.environ.get("GO_CMD", "go")
flt = os.environ.get("E2E_FILTER", "TestE2E")
green = os.environ.get("E2E_C_GREEN", "")
red = os.environ.get("E2E_C_RED", "")
yellow = os.environ.get("E2E_C_YELLOW", "")
reset = os.environ.get("E2E_C_RESET", "")

ok_marker = "[OK]"
fail_marker = "[FAIL]"
skip_marker = "[SKIP]"

cmd = [go_cmd, "test", "-json", "-v", "-count=1", "-timeout", "600s", "-run", flt, "./ui/tests/..."]

def expectation(name: str) -> str:
    leaf = name.split("/")[-1]
    rules = [
        (r"MTLS_IncomingClientCert_Required", "HTTPS mTLS rejects without client cert, passes with client cert, passes after disabled"),
        (r"Returns403|Blocks_.*|Blocks_Without", "HTTP 403"),
        (r"Returns429|RateLimit_Burst|CustomLimitRules", "HTTP 429"),
        (r"Gets302|Challenge", "HTTP 302/challenge flow"),
        (r"Returns451", "HTTP 451"),
        (r"Geo", "HTTP 403"),
        (r"Allows|Passes|Bypasses|Recovery", "allowed response (not blocked)"),
        (r"BrandedHTML", "HTTP 403 + branded HTML"),
        (r"Headers|HSTS|CookieFlags", "expected headers present"),
        (r"VirtualPatches", "virtual patch block"),
        (r"Config|Parsing|WebSocket|JA3", "compiled/runtime config present"),
    ]
    for pattern, value in rules:
        if re.search(pattern, leaf):
            return value
    return "expected behavior"

def actual_for(status: str, expected: str) -> str:
    if status == "pass":
        return expected
    if status == "skip":
        return "skipped"
    return "failed"

def display_name(name: str) -> str:
    leaf = name.split("/")[-1]
    if leaf.startswith("TestE2E"):
        leaf = leaf[len("TestE2E"):]
    elif leaf.startswith("Test"):
        leaf = leaf[len("Test"):]
    return leaf.lstrip("_") or name

def suite_name(name: str) -> str:
    return f"{display_name(name)} suite"

outputs = {}
summary = ""
parents = set()
started_tests = set()
passed_tests = set()
skipped_tests = set()
failed_tests = set()
proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, encoding="utf-8", errors="replace", bufsize=1)
with open(log_path, "w", encoding="utf-8", errors="replace") as log:
    try:
        for line in proc.stdout:
            log.write(line)
            log.flush()
            try:
                item = json.loads(line)
            except Exception:
                continue
            test = item.get("Test") or ""
            action = item.get("Action")
            if item.get("Package") == "waf/ui/tests" and not test and action == "pass":
                elapsed = item.get("Elapsed")
                summary = f"ok waf/ui/tests {elapsed}s" if elapsed is not None else "ok waf/ui/tests"
                continue
            if not test.startswith("TestE2E"):
                continue
            if action == "run":
                started_tests.add(test)
            if "/" in test:
                parents.add(test.split("/", 1)[0])
            name = display_name(test)
            expected = expectation(test)
            if action == "output":
                out = (item.get("Output") or "").strip()
                if out:
                    outputs.setdefault(test, []).append(out)
            elif action in ("pass", "fail", "skip"):
                if action == "pass":
                    passed_tests.add(test)
                elif action == "skip":
                    skipped_tests.add(test)
                else:
                    failed_tests.add(test)
                actual = actual_for(action, expected)
                if action == "pass":
                    if "/" not in test and test in parents:
                        print(f"{green}[e2e] {ok_marker} {suite_name(test)} completed{reset}", flush=True)
                    else:
                        print(f"{green}[e2e] {ok_marker} {name}: expected={expected}; actual={actual}{reset}", flush=True)
                elif action == "skip":
                    if "/" not in test and test in parents:
                        print(f"{yellow}[e2e] {skip_marker} {suite_name(test)} skipped{reset}", flush=True)
                    else:
                        print(f"{yellow}[e2e] {skip_marker} {name}: expected={expected}; actual={actual}{reset}", flush=True)
                else:
                    if "/" not in test and test in parents:
                        print(f"{red}[e2e] {fail_marker} {suite_name(test)} failed{reset}", flush=True)
                    else:
                        print(f"{red}[e2e] {fail_marker} {name}: expected={expected}; actual={actual}{reset}", flush=True)
                    for out in outputs.get(test, [])[-8:]:
                        print(f"{red}[e2e]   {out}{reset}", flush=True)
    except KeyboardInterrupt:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except Exception:
            proc.kill()
        sys.exit(130)
rc = proc.wait()
completed_tests = len(passed_tests) + len(skipped_tests) + len(failed_tests)
counter_color = green if rc == 0 and len(started_tests) == completed_tests else red
print(
    f"{counter_color}[e2e] Test count: started={len(started_tests)}; passed={len(passed_tests)}; "
    f"skipped={len(skipped_tests)}; failed={len(failed_tests)}; completed={completed_tests}{reset}",
    flush=True,
)
if summary and summary_out:
    with open(summary_out, "w", encoding="utf-8") as f:
        f.write(summary + "\n")
sys.exit(rc)
PY
}

# Detect docker compose command.
if [ -z "${COMPOSE_CMD:-}" ]; then
  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
  else
    echo "[e2e] ERROR: no docker compose found" >&2
    exit 1
  fi
fi

E2E_BASE_URL="http://127.0.0.1:${E2E_PORT}"
E2E_RUNTIME_URL="http://127.0.0.1:${E2E_RT_PORT}"
E2E_RUNTIME_HTTPS_URL="https://127.0.0.1:${E2E_RT_HTTPS_PORT}"
E2E_RUNTIME_HEALTH_URL="http://127.0.0.1:${E2E_RT_HLT_PORT}"
STACK_DOWN_DONE=0

cleanup() {
  [ "$E2E_KEEP_STACK" = "1" ] && {
    warn "E2E_KEEP_STACK=1 - skipping teardown"
    info "Control-plane: $E2E_BASE_URL"
    info "Runtime:       $E2E_RUNTIME_URL"
    return
  }
  [ "$STACK_DOWN_DONE" = "1" ] && return
  step "Removing e2e stack"
  cd "$E2E_COMPOSE_DIR"
  $COMPOSE_CMD -f docker-compose.yml down --volumes --remove-orphans >>"$E2E_LOG_FILE" 2>&1 || true
  STACK_DOWN_DONE=1
  ok "Stack removed"
}
trap cleanup EXIT INT TERM

# Bring up stack.
step "Starting e2e stack (control-plane=:$E2E_PORT runtime=:$E2E_RT_PORT)"
info "Detailed log: $E2E_LOG_FILE"
cd "$E2E_COMPOSE_DIR"
$COMPOSE_CMD -f docker-compose.yml down --volumes --remove-orphans >>"$E2E_LOG_FILE" 2>&1 || true
run_quiet "Build and start containers" $COMPOSE_CMD -f docker-compose.yml up -d --build
ok "Containers are up"

# Wait for control-plane health.
step "Waiting for control-plane healthz (timeout ${E2E_TIMEOUT}s)"
elapsed=0
until curl -fsS "$E2E_BASE_URL/healthz" >/dev/null 2>&1; do
  [ "$elapsed" -ge "$E2E_TIMEOUT" ] && {
    fail_msg "control-plane healthz timeout"
    $COMPOSE_CMD -f docker-compose.yml logs --tail=80 >&2
    exit 1
  }
  sleep 2
  elapsed=$((elapsed + 2))
done
ok "Control-plane healthy after ${elapsed}s"

# Wait for bootstrap admin.
step "Waiting for bootstrap admin"
elapsed=0
until curl -fsS -X POST "$E2E_BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${E2E_USER}\",\"password\":\"${E2E_PASS}\"}" \
    >/dev/null 2>&1; do
  [ "$elapsed" -ge 30 ] && {
    fail_msg "admin login timeout"
    exit 1
  }
  sleep 2
  elapsed=$((elapsed + 2))
done
ok "Admin ready"

# Wait for runtime health.
step "Waiting for runtime healthz (timeout ${E2E_TIMEOUT}s)"
elapsed=0
until curl -fsS "$E2E_RUNTIME_HEALTH_URL/healthz" >/dev/null 2>&1; do
  [ "$elapsed" -ge "$E2E_TIMEOUT" ] && {
    fail_msg "runtime healthz timeout"
    $COMPOSE_CMD -f docker-compose.yml logs runtime --tail=50 >&2
    exit 1
  }
  sleep 2
  elapsed=$((elapsed + 2))
done
ok "Runtime healthy after ${elapsed}s"

# Initial compile+apply so runtime has a valid revision.
step "Compiling and applying initial revision"
CP_COOKIE_JAR="$(mktemp)"
curl -sS -c "$CP_COOKIE_JAR" -X POST "$E2E_BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${E2E_USER}\",\"password\":\"${E2E_PASS}\"}" >/dev/null
COMPILE_OUT="$(curl -sS -b "$CP_COOKIE_JAR" -X POST "$E2E_BASE_URL/api/revisions/compile" \
  -H "Content-Type: application/json" -d '{}')"
printf '%s\n' "[compile] $COMPILE_OUT" >>"$E2E_LOG_FILE"
REV_ID="$(printf '%s' "$COMPILE_OUT" | grep -o '"revision_id":"[^"]*"' | cut -d'"' -f4)"
if [ -z "$REV_ID" ]; then
  REV_ID="$(printf '%s' "$COMPILE_OUT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)"
fi
if [ -n "$REV_ID" ]; then
  sleep 3
  APPLY_OUT="$(curl -sS -b "$CP_COOKIE_JAR" -X POST "$E2E_BASE_URL/api/revisions/$REV_ID/apply" \
    -H "Content-Type: application/json" -d '{}' 2>&1 || true)"
  printf '%s\n' "[apply] $APPLY_OUT" >>"$E2E_LOG_FILE"
  sleep 5
fi
rm -f "$CP_COOKIE_JAR"
ok "Initial revision ready (id=$REV_ID)"

# Run tests.
step "Running tests: $E2E_FILTER"
cd "$REPO_ROOT"
TEST_EXIT=0
TEST_LOG="$(mktemp)"
WAF_E2E_BASE_URL="$E2E_BASE_URL"
WAF_E2E_USERNAME="$E2E_USER"
WAF_E2E_PASSWORD="$E2E_PASS"
WAF_E2E_RUNTIME_URL="$E2E_RUNTIME_URL"
WAF_E2E_RUNTIME_HTTPS_URL="$E2E_RUNTIME_HTTPS_URL"
WAF_E2E_RUNTIME_HEALTH_URL="$E2E_RUNTIME_HEALTH_URL"
WAF_E2E_RUNTIME_API_TOKEN="e2e-test-runtime-token"
WAF_E2E_MANAGEMENT_HOST="${WAF_E2E_MANAGEMENT_HOST:-e2e-management.test}"
export WAF_E2E_BASE_URL WAF_E2E_USERNAME WAF_E2E_PASSWORD WAF_E2E_RUNTIME_URL WAF_E2E_RUNTIME_HTTPS_URL WAF_E2E_RUNTIME_HEALTH_URL WAF_E2E_RUNTIME_API_TOKEN WAF_E2E_MANAGEMENT_HOST GO_CMD E2E_FILTER
TEST_SUMMARY_FILE="$(mktemp)"
E2E_SUMMARY_OUT="$TEST_SUMMARY_FILE" run_go_e2e_stream || TEST_EXIT=$?
cat "$TEST_LOG" >>"$E2E_LOG_FILE"
TEST_SUMMARY="$(cat "$TEST_SUMMARY_FILE" 2>/dev/null || true)"
rm -f "$TEST_SUMMARY_FILE"

if [ "$TEST_EXIT" -ne 0 ]; then
  fail_msg "Tests failed (exit $TEST_EXIT). Expected/actual details from Go output:"
  show_log_tail "$TEST_LOG" 160
  rm -f "$TEST_LOG"
  exit "$TEST_EXIT"
fi
rm -f "$TEST_LOG"
ok "Tests passed: ${TEST_SUMMARY:-$E2E_FILTER}"
ok "All e2e checks passed"
