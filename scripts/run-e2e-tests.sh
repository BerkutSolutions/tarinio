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
#   E2E_PASS          required admin credential (or WAF_E2E_PASSWORD)
#   E2E_TIMEOUT       seconds to wait for healthcheck (default: 180)
#   E2E_FILTER        go test -run filter (default: TestE2E)
#   E2E_FRESH_ONBOARDING set to 1 to verify clean first-run onboarding and HTTPS
#   COMPOSE_CMD       docker compose command (auto-detected)
#   GO_CMD            go binary (default: go)
#   E2E_KEEP_STACK    set to 1 to skip teardown (debug)
#   E2E_BUILD_ATTEMPTS docker compose build/start attempts (default: 3)
#   E2E_BOOTSTRAP_REVISION_TIMEOUT seconds to wait for the initial active revision (default: E2E_TIMEOUT)
#   E2E_PROJECT      Docker Compose project name (default: waf-e2e)
#   E2E_BROWSER_ONLY set to 1 to run the Playwright slice instead of Go E2E
#   E2E_BROWSER_SPECS whitespace-separated Playwright spec paths for this stack
#   E2E_BROWSER_IMAGE prebuilt Playwright+Go image used by browser E2E

set -eu

REPO_ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)}"
E2E_COMPOSE_DIR="$REPO_ROOT/deploy/compose/e2e"
COMPOSE_FILE="$E2E_COMPOSE_DIR/docker-compose.yml"
E2E_PORT="${E2E_PORT:-18080}"
E2E_RT_PORT="${E2E_RT_PORT:-10080}"
E2E_RT_HTTPS_PORT="${E2E_RT_HTTPS_PORT:-10443}"
E2E_MTLS_UPSTREAM_PORT="${E2E_MTLS_UPSTREAM_PORT:-18084}"
E2E_RT_HLT_PORT="${E2E_RT_HLT_PORT:-18081}"
E2E_USER="${E2E_USER:-e2e-admin}"
E2E_PASS="${E2E_PASS:-${WAF_E2E_PASSWORD:-}}"
E2E_TIMEOUT="${E2E_TIMEOUT:-180}"
E2E_BOOTSTRAP_REVISION_TIMEOUT="${E2E_BOOTSTRAP_REVISION_TIMEOUT:-$E2E_TIMEOUT}"
E2E_FILTER="${E2E_FILTER:-TestE2E}"
E2E_PROJECT="${E2E_PROJECT:-waf-e2e}"
E2E_FRESH_ONBOARDING="${E2E_FRESH_ONBOARDING:-0}"
E2E_KEEP_STACK="${E2E_KEEP_STACK:-0}"
E2E_BROWSER_ONLY="${E2E_BROWSER_ONLY:-0}"
E2E_BROWSER_SPECS="${E2E_BROWSER_SPECS:-}"
E2E_BROWSER_IMAGE="${E2E_BROWSER_IMAGE:-tarinio-playwright-e2e:1.61.1}"
E2E_BROWSER_RUNTIME_FAULT="${E2E_BROWSER_RUNTIME_FAULT:-}"
E2E_DASHBOARD_SEED="${E2E_DASHBOARD_SEED:-auto}"
GO_CMD="${GO_CMD:-go}"
E2E_LOG_DIR="${E2E_LOG_DIR:-$REPO_ROOT/.work/logs}"
mkdir -p "$E2E_LOG_DIR"
E2E_LOG_FILE="$E2E_LOG_DIR/e2e-$(date +%Y%m%d_%H%M%S).log"
E2E_EVIDENCE_DIR="${E2E_EVIDENCE_DIR:-$E2E_LOG_DIR}"
E2E_STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
E2E_BUILD_RETRIES=0
export COMPOSE_PROJECT_NAME="$E2E_PROJECT"
export E2E_PORT E2E_RT_PORT E2E_RT_HTTPS_PORT E2E_RT_HLT_PORT E2E_PASS

[ -n "$E2E_PASS" ] || { echo "[e2e] ERROR: E2E_PASS or WAF_E2E_PASSWORD is required" >&2; exit 1; }

if [ "$E2E_FRESH_ONBOARDING" = "1" ]; then
  export E2E_BOOTSTRAP_ADMIN_ENABLED=false
  export E2E_DEV_FAST_START_ENABLED=false
  export E2E_RUNTIME_STARTUP_BUNDLE_WAIT_SECONDS=0
fi

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

run_compose_start() {
  max_attempts="${E2E_BUILD_ATTEMPTS:-3}"
  attempt=1
  while [ "$attempt" -le "$max_attempts" ]; do
    step "Build and start containers (attempt $attempt/$max_attempts)"
    if $COMPOSE_CMD -f docker-compose.yml up -d --build >>"$E2E_LOG_FILE" 2>&1; then
      ok "Build and start containers"
      return 0
    fi
    if [ "$attempt" -lt "$max_attempts" ]; then
      E2E_BUILD_RETRIES=$attempt
      retry_delay=$((attempt * 5))
      if grep -Eq 'registry-1\.docker\.io|failed to resolve source metadata|: EOF' "$E2E_LOG_FILE"; then
        E2E_INFRASTRUCTURE_INSTABILITY=1
        warn "Docker registry request failed; retrying in ${retry_delay}s"
      else
        warn "Container build failed; retrying in ${retry_delay}s"
      fi
      $COMPOSE_CMD -f docker-compose.yml down --volumes --remove-orphans --rmi local >>"$E2E_LOG_FILE" 2>&1 || true
      sleep "$retry_delay"
    fi
    attempt=$((attempt + 1))
  done
  fail_msg "Build and start containers failed after $max_attempts attempts (log: $E2E_LOG_FILE)"
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

def is_e2e_test(name: str) -> bool:
    return name.startswith("TestE2E") or name.startswith("TestFreshOnboarding")

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
    if not is_e2e_test(test):
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

cmd = [go_cmd, "test", "-tags=e2e", "-json", "-v", "-count=1", "-timeout", "600s", "-run", flt, "./ui/tests/..."]

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

def is_e2e_test(name: str) -> bool:
    return name.startswith("TestE2E") or name.startswith("TestFreshOnboarding")

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
            if not is_e2e_test(test):
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
if skipped_tests:
    print(f"{red}[e2e] FAIL: skipped E2E tests are not accepted as proof of WAF behavior{reset}", flush=True)
    sys.exit(1)
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
WAF_E2E_MTLS_FIXTURE_URL="${WAF_E2E_MTLS_FIXTURE_URL:-http://127.0.0.1:${E2E_MTLS_UPSTREAM_PORT}}"
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
  if ! $COMPOSE_CMD -f docker-compose.yml down --volumes --remove-orphans --rmi local >>"$E2E_LOG_FILE" 2>&1; then
    fail_msg "Could not remove isolated stack $E2E_PROJECT"
    show_log_tail "$E2E_LOG_FILE" 80
    return 1
  fi
  if [ -n "$($COMPOSE_CMD -f docker-compose.yml ps -aq 2>/dev/null)" ]; then
    fail_msg "Compose resources remain after cleanup for $E2E_PROJECT"
    return 1
  fi
  STACK_DOWN_DONE=1
  ok "Stack removed"
}
trap cleanup EXIT INT TERM

# Bring up stack.
step "Starting e2e stack (control-plane=:$E2E_PORT runtime=:$E2E_RT_PORT)"
info "Detailed log: $E2E_LOG_FILE"
info "Isolated Compose project: $E2E_PROJECT"
cd "$E2E_COMPOSE_DIR"
run_quiet "Remove stale isolated stack" $COMPOSE_CMD -f docker-compose.yml down --volumes --remove-orphans --rmi local
run_compose_start
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

if [ "$E2E_FRESH_ONBOARDING" != "1" ]; then
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

  # Health proves only that the reload listener is alive. The asynchronous
  # dev-fast-start transaction must also publish its initial revision before a
  # test starts its own compile/apply; otherwise the bootstrap can race the
  # first scenario and replace its configuration.
  step "Waiting for bootstrap runtime revision (timeout ${E2E_BOOTSTRAP_REVISION_TIMEOUT}s)"
  elapsed=0
  until $COMPOSE_CMD -f docker-compose.yml exec -T runtime sh -c '[ -s /var/lib/waf/active/current.json ]' >>"$E2E_LOG_FILE" 2>&1; do
    [ "$elapsed" -ge "$E2E_BOOTSTRAP_REVISION_TIMEOUT" ] && {
      fail_msg "bootstrap runtime revision timeout"
      $COMPOSE_CMD -f docker-compose.yml logs control-plane runtime --tail=100 >&2
      exit 1
    }
    sleep 2
    elapsed=$((elapsed + 2))
  done
  ok "Bootstrap runtime revision is ready after ${elapsed}s"
fi

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
WAF_E2E_COMPOSE_FILE="$COMPOSE_FILE"
WAF_E2E_MANAGEMENT_HOST="${WAF_E2E_MANAGEMENT_HOST:-e2e-management.test}"
WAF_E2E_AUTH_BASE_URL="${WAF_E2E_AUTH_BASE_URL:-$E2E_RUNTIME_URL}"
WAF_E2E_ANTIBOT_BASE_URL="${WAF_E2E_ANTIBOT_BASE_URL:-$E2E_RUNTIME_HTTPS_URL}"
WAF_E2E_AUTOSTART_SMART="${WAF_E2E_AUTOSTART_SMART:-1}"
WAF_E2E_L4_L7_PROTECTION="${WAF_E2E_L4_L7_PROTECTION:-1}"
WAF_E2E_FRESH_ONBOARDING="${E2E_FRESH_ONBOARDING}"
WAF_E2E_DAST_CANARY_URL="${WAF_E2E_DAST_CANARY_URL:-http://127.0.0.1:${E2E_DAST_CANARY_PORT:-18083}}"
WAF_E2E_CONTROL_PLANE_CONTAINER="$($COMPOSE_CMD -f "$COMPOSE_FILE" ps -q control-plane)"
WAF_E2E_RUNTIME_CONTAINER="$($COMPOSE_CMD -f "$COMPOSE_FILE" ps -q runtime)"
WAF_E2E_ATTACKER_CONTAINER="$($COMPOSE_CMD -f "$COMPOSE_FILE" ps -q e2e-attacker)"
WAF_E2E_L4_ATTACKER_CONTAINER="$($COMPOSE_CMD -f "$COMPOSE_FILE" ps -q e2e-l4-attacker)"
if [ -z "$WAF_E2E_CONTROL_PLANE_CONTAINER" ] || [ -z "$WAF_E2E_RUNTIME_CONTAINER" ] || [ -z "$WAF_E2E_ATTACKER_CONTAINER" ] || [ -z "$WAF_E2E_L4_ATTACKER_CONTAINER" ]; then
  fail_msg "could not resolve isolated E2E container IDs"
  exit 1
fi
WAF_E2E_ATTACKER_IP="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$WAF_E2E_ATTACKER_CONTAINER")"
WAF_E2E_L4_ATTACKER_IP="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$WAF_E2E_L4_ATTACKER_CONTAINER")"
if [ -z "$WAF_E2E_ATTACKER_IP" ] || [ -z "$WAF_E2E_L4_ATTACKER_IP" ]; then
  fail_msg "could not resolve isolated E2E attacker IP addresses"
  exit 1
fi
export WAF_E2E_BASE_URL WAF_E2E_USERNAME WAF_E2E_PASSWORD WAF_E2E_RUNTIME_URL WAF_E2E_RUNTIME_HTTPS_URL WAF_E2E_MTLS_FIXTURE_URL WAF_E2E_RUNTIME_HEALTH_URL WAF_E2E_RUNTIME_API_TOKEN WAF_E2E_COMPOSE_FILE WAF_E2E_MANAGEMENT_HOST WAF_E2E_AUTH_BASE_URL WAF_E2E_ANTIBOT_BASE_URL WAF_E2E_AUTOSTART_SMART WAF_E2E_L4_L7_PROTECTION WAF_E2E_FRESH_ONBOARDING WAF_E2E_RESILIENCE WAF_E2E_VERIFY_DECAY WAF_E2E_DAST_CANARY_URL WAF_E2E_ATTACKER_IP WAF_E2E_L4_ATTACKER_IP WAF_E2E_CONTROL_PLANE_CONTAINER WAF_E2E_RUNTIME_CONTAINER WAF_E2E_ATTACKER_CONTAINER WAF_E2E_L4_ATTACKER_CONTAINER GO_CMD E2E_FILTER
WAF_E2E_DISPOSABLE=1
WAF_BROWSER_BASE_URL="https://e2e-management.test:${E2E_RT_HTTPS_PORT}"
WAF_E2E_RUN_ID="${WAF_E2E_RUN_ID:-${E2E_PROJECT}-$(date +%s)}"
export WAF_E2E_DISPOSABLE WAF_BROWSER_BASE_URL WAF_E2E_RUN_ID
if [ "$E2E_DASHBOARD_SEED" = "1" ] || { [ "$E2E_BROWSER_ONLY" = "1" ] && [ "$E2E_DASHBOARD_SEED" = "auto" ] && printf '%s' "$E2E_BROWSER_SPECS" | grep -Eq '(dashboard|requests-complete|tabs)\.spec\.ts'; }; then
  step "Seeding real Dashboard telemetry"
  sh "$REPO_ROOT/scripts/seed-dashboard-telemetry.sh" "$COMPOSE_FILE" "$E2E_BASE_URL" "$E2E_USER" "$E2E_PASS" >>"$E2E_LOG_FILE" 2>&1
  ok "Dashboard telemetry is aggregated"
fi
TEST_SUMMARY_FILE="$(mktemp)"
if [ "$E2E_BROWSER_ONLY" = "1" ]; then
  [ -n "$E2E_BROWSER_SPECS" ] || { fail_msg "E2E_BROWSER_SPECS is required in browser-only mode"; exit 1; }
  docker image inspect "$E2E_BROWSER_IMAGE" >/dev/null 2>&1 || { fail_msg "Browser image is missing: $E2E_BROWSER_IMAGE"; exit 1; }
  if [ "$E2E_BROWSER_RUNTIME_FAULT" = "requests-backend" ]; then
    step "Authenticating before Requests backend-failure injection"
    docker run --rm --network host --user "$(id -u):$(id -g)" \
      -e HOME=/tmp -e CI -e WAF_E2E_DISPOSABLE -e WAF_BROWSER_BASE_URL \
      -e WAF_E2E_USERNAME -e WAF_E2E_PASSWORD -e WAF_E2E_RUN_ID \
      -v "$REPO_ROOT:/workspace" -w /workspace/e2e/browser "$E2E_BROWSER_IMAGE" \
      bash -lc 'npm ci --prefer-offline --no-audit --fund=false && npx playwright test --project=setup' >>"$E2E_LOG_FILE" 2>&1
    WAF_BROWSER_FAULT_SYNC_FILE="$E2E_LOG_DIR/requests-backend-fault"
    WAF_BROWSER_FAULT_SYNC_CONTAINER_FILE="/workspace/build/e2e/$(basename "$E2E_LOG_DIR")/requests-backend-fault"
    export WAF_BROWSER_FAULT_SYNC_FILE WAF_BROWSER_FAULT_SYNC_CONTAINER_FILE
    rm -f "$WAF_BROWSER_FAULT_SYNC_FILE".*
    (
      attempt=0
      while [ ! -f "$WAF_BROWSER_FAULT_SYNC_FILE.desktop.ready" ]; do
        attempt=$((attempt + 1))
        [ "$attempt" -lt 90 ] || { printf '%s\n' 'browser fault readiness signal timed out' > "$WAF_BROWSER_FAULT_SYNC_FILE.error"; exit 1; }
        sleep 1
      done
      $COMPOSE_CMD -f "$COMPOSE_FILE" pause runtime >>"$E2E_LOG_FILE" 2>&1 || { printf '%s\n' 'docker compose pause runtime failed' > "$WAF_BROWSER_FAULT_SYNC_FILE.error"; exit 1; }
      : > "$WAF_BROWSER_FAULT_SYNC_FILE.paused"
    ) &
    E2E_BROWSER_FAULT_PID=$!
  fi
  step "Running browser tests on isolated stack: $E2E_BROWSER_SPECS"
  docker run --rm --network host --user "$(id -u):$(id -g)" \
    -e HOME=/tmp -e CI -e CI_COMMIT_SHA -e CI_PIPELINE_URL \
    -e WAF_E2E_DISPOSABLE -e WAF_BROWSER_BASE_URL -e WAF_E2E_RUNTIME_URL \
    -e WAF_E2E_USERNAME -e WAF_E2E_PASSWORD -e WAF_E2E_RUN_ID \
    -e WAF_BROWSER_RESULTS_FILE -e WAF_BROWSER_OUTPUT_DIR -e WAF_BROWSER_JUNIT_FILE -e WAF_BROWSER_FAULT_SYNC_CONTAINER_FILE \
    -e WAF_BROWSER_WORKERS="${E2E_BROWSER_WORKERS:-1}" -e E2E_BROWSER_SPECS -e E2E_BROWSER_RUNTIME_FAULT \
    -v "$REPO_ROOT:/workspace" -w /workspace/e2e/browser "$E2E_BROWSER_IMAGE" \
    bash -lc 'npm ci --prefer-offline --no-audit --fund=false && if [ "$E2E_BROWSER_RUNTIME_FAULT" = "requests-backend" ]; then npm test -- --project=desktop --project=mobile --no-deps $E2E_BROWSER_SPECS; else npm test -- $E2E_BROWSER_SPECS; fi' >"$TEST_LOG" 2>&1 || TEST_EXIT=$?
  TEST_SUMMARY="browser:$E2E_BROWSER_SPECS"
  if [ "$E2E_BROWSER_RUNTIME_FAULT" = "requests-backend" ]; then
    if ! wait "$E2E_BROWSER_FAULT_PID"; then
      fail_msg "Could not pause isolated runtime"
      [ -f "$WAF_BROWSER_FAULT_SYNC_FILE.error" ] && cat "$WAF_BROWSER_FAULT_SYNC_FILE.error" >&2
      TEST_EXIT="${TEST_EXIT:-1}"
    elif [ -f "$WAF_BROWSER_FAULT_SYNC_FILE.paused" ]; then
      $COMPOSE_CMD -f "$COMPOSE_FILE" unpause runtime >>"$E2E_LOG_FILE" 2>&1 || { fail_msg "Could not unpause isolated runtime"; TEST_EXIT="${TEST_EXIT:-1}"; }
    fi
  fi
else
  E2E_SUMMARY_OUT="$TEST_SUMMARY_FILE" run_go_e2e_stream || TEST_EXIT=$?
  TEST_SUMMARY="$(cat "$TEST_SUMMARY_FILE" 2>/dev/null || true)"
fi
cat "$TEST_LOG" >>"$E2E_LOG_FILE"
rm -f "$TEST_SUMMARY_FILE"

redact_e2e_artifacts() {
  python3 "$REPO_ROOT/scripts/redact-e2e-artifact.py" "$TEST_LOG" || true
  python3 "$REPO_ROOT/scripts/redact-e2e-artifact.py" "$E2E_LOG_FILE" || true
}

write_e2e_stability() {
  status="$1"
  failure_class="${2:-}"
  mkdir -p "$E2E_EVIDENCE_DIR"
  E2E_STABILITY_PATH="$E2E_EVIDENCE_DIR/e2e-stability.json" \
  E2E_STARTED_AT="$E2E_STARTED_AT" E2E_BUILD_RETRIES="$E2E_BUILD_RETRIES" E2E_FILTER="$E2E_FILTER" \
  E2E_STABILITY_STATUS="$status" E2E_STABILITY_FAILURE="$failure_class" \
  E2E_STABILITY_FINISHED="$(date -u +%Y-%m-%dT%H:%M:%SZ)" python3 - <<'PY'
import json, os
from pathlib import Path

Path(os.environ["E2E_STABILITY_PATH"]).write_text(json.dumps({
    "schema_version": 1,
    "suite": os.environ.get("E2E_FILTER", "TestE2E"),
    "started_at": os.environ["E2E_STARTED_AT"],
    "finished_at": os.environ["E2E_STABILITY_FINISHED"],
    "test_attempts": 1,
    "build_retries": int(os.environ.get("E2E_BUILD_RETRIES", "0")),
    "status": os.environ["E2E_STABILITY_STATUS"],
    "failure_class": os.environ["E2E_STABILITY_FAILURE"] or None,
    "flaky": int(os.environ.get("E2E_BUILD_RETRIES", "0")) > 0,
}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

capture_e2e_runtime_diagnostics() {
  {
    printf '\n[e2e] Runtime diagnostics captured before teardown\n'
    $COMPOSE_CMD -f "$COMPOSE_FILE" ps
    $COMPOSE_CMD -f "$COMPOSE_FILE" logs --no-color --timestamps --tail=400 runtime control-plane tarinio-sentinel
  } >>"$E2E_LOG_FILE" 2>&1 || true
}

if [ "$TEST_EXIT" -ne 0 ]; then
  capture_e2e_runtime_diagnostics
  redact_e2e_artifacts
  if [ "$E2E_BROWSER_ONLY" = "1" ] && [ -n "${WAF_BROWSER_RESULTS_FILE:-}" ] && [ -f "$REPO_ROOT/e2e/browser/$WAF_BROWSER_RESULTS_FILE" ]; then
    node "$REPO_ROOT/e2e/browser/scripts/write-evidence.mjs" "$REPO_ROOT/e2e/browser/$WAF_BROWSER_RESULTS_FILE" "$E2E_EVIDENCE_DIR" "$E2E_PROJECT" || true
  else
    python3 "$REPO_ROOT/scripts/write-e2e-evidence-report.py" --log "$TEST_LOG" --runtime-log "$E2E_LOG_FILE" --output-dir "$E2E_EVIDENCE_DIR" --suite "$E2E_FILTER" || true
  fi
  write_e2e_stability failed test
  fail_msg "Tests failed (exit $TEST_EXIT). Expected/actual details from Go output:"
  show_log_tail "$TEST_LOG" 160
  rm -f "$TEST_LOG"
  exit "$TEST_EXIT"
fi
redact_e2e_artifacts
if [ "$E2E_BROWSER_ONLY" = "1" ]; then
  node "$REPO_ROOT/e2e/browser/scripts/write-evidence.mjs" "$REPO_ROOT/e2e/browser/$WAF_BROWSER_RESULTS_FILE" "$E2E_EVIDENCE_DIR" "$E2E_PROJECT"
else
  python3 "$REPO_ROOT/scripts/write-e2e-evidence-report.py" --log "$TEST_LOG" --runtime-log "$E2E_LOG_FILE" --output-dir "$E2E_EVIDENCE_DIR" --suite "$E2E_FILTER"
fi
write_e2e_stability passed
rm -f "$TEST_LOG"
ok "Tests passed: ${TEST_SUMMARY:-$E2E_FILTER}"
ok "All e2e checks passed"
