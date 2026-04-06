#!/usr/bin/env sh
set -eu

REPO_URL="${REPO_URL:-https://github.com/BerkutSolutions/tarinio.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-default}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/tmp/tarinio-install-${PROFILE}-${TIMESTAMP}.log"
FAILED=0

if [ -t 1 ]; then
  C_RESET="$(printf '\033[0m')"
  C_BOLD="$(printf '\033[1m')"
  C_BLUE="$(printf '\033[34m')"
  C_GREEN="$(printf '\033[32m')"
  C_YELLOW="$(printf '\033[33m')"
  C_RED="$(printf '\033[31m')"
else
  C_RESET=""
  C_BOLD=""
  C_BLUE=""
  C_GREEN=""
  C_YELLOW=""
  C_RED=""
fi

section() {
  echo
  printf "%s%s== %s ==%s\n" "$C_BOLD" "$C_BLUE" "$1" "$C_RESET"
}

step() {
  printf "%s[%s..%s] %s\n" "$C_BOLD" "$C_BLUE" "$C_RESET" "$1"
}

ok() {
  printf "%s[%sOK%s] %s\n" "$C_BOLD" "$C_GREEN" "$C_RESET" "$1"
}

warn() {
  printf "%s[%sWARN%s] %s\n" "$C_BOLD" "$C_YELLOW" "$C_RESET" "$1"
}

fail() {
  FAILED=1
  printf "%s[%sFAIL%s] %s\n" "$C_BOLD" "$C_RED" "$C_RESET" "$1" >&2
  exit 1
}

on_exit() {
  code=$?
  if [ "$code" -ne 0 ] && [ "$FAILED" -eq 0 ]; then
    printf "%s[%sFAIL%s] Installation aborted (exit code %s). Log: %s%s\n" "$C_BOLD" "$C_RED" "$C_RESET" "$code" "$LOG_FILE" "$C_RESET" >&2
    if [ -f "$LOG_FILE" ]; then
      echo "Last 30 log lines:" >&2
      tail -n 30 "$LOG_FILE" >&2 || true
    fi
  fi
}

run_logged() {
  # shellcheck disable=SC2068
  "$@" >>"$LOG_FILE" 2>&1
}

extract_version() {
  file="$1"
  if [ ! -f "$file" ]; then
    printf "unknown"
    return
  fi
  version="$(sed -n 's/.*AppVersion = "\(.*\)".*/\1/p' "$file" | head -n 1)"
  if [ -n "$version" ]; then
    printf "%s" "$version"
  else
    printf "unknown"
  fi
}

trap on_exit EXIT

mkdir -p "$(dirname "$LOG_FILE")"
: >"$LOG_FILE"

section "TARINIO AIO Installer"
step "Checking required tools"

if command -v docker >/dev/null 2>&1; then
  ok "docker detected"
else
  fail "docker is not installed"
fi

if command -v git >/dev/null 2>&1; then
  ok "git detected"
else
  fail "git is not installed"
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
  COMPOSE_LEGACY=0
  ok "using docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
  COMPOSE_LEGACY=1
  warn "using legacy docker-compose"
else
  fail "docker compose / docker-compose is not installed"
fi

section "Repository"
if [ -d "$INSTALL_DIR/.git" ]; then
  step "Updating existing repository in $INSTALL_DIR"
  run_logged git -C "$INSTALL_DIR" fetch --all --tags
  run_logged git -C "$INSTALL_DIR" checkout "$BRANCH"
  run_logged git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
  ok "repository updated to branch $BRANCH"
else
  step "Cloning repository to $INSTALL_DIR"
  run_logged git clone --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
  ok "repository cloned"
fi

TARGET_VERSION="$(extract_version "$INSTALL_DIR/control-plane/internal/appmeta/meta.go")"
TARGET_COMMIT="$(git -C "$INSTALL_DIR" rev-parse --short HEAD 2>/dev/null || printf "unknown")"
section "Install Plan"
ok "target version: $TARGET_VERSION"
ok "branch/commit: $BRANCH ($TARGET_COMMIT)"
ok "profile: $PROFILE"
ok "install path: $INSTALL_DIR"
ok "detailed log: $LOG_FILE"

section "Profile Preparation"
cd "$INSTALL_DIR/deploy/compose/$PROFILE"
ok "selected profile: $PROFILE"

if [ ! -f .env ]; then
  step "Creating .env from .env.example"
  cp .env.example .env
  ok ".env created"
else
  ok ".env already exists"
fi

section "Build And Start"
EXISTING_CONTAINERS="$($COMPOSE_CMD -f docker-compose.yml ps -aq 2>/dev/null || true)"
if [ -n "$EXISTING_CONTAINERS" ]; then
  if [ "$COMPOSE_LEGACY" -eq 1 ]; then
    step "Legacy docker-compose detected: removing old project containers to avoid recreate bug (without volumes)"
    if run_logged $COMPOSE_CMD -f docker-compose.yml down --remove-orphans; then
      ok "existing containers removed (volumes preserved)"
    else
      warn "failed to remove old containers, continuing with rebuild"
    fi
  else
    step "Existing project containers detected, stopping gracefully (without volumes)"
    if run_logged $COMPOSE_CMD -f docker-compose.yml stop; then
      ok "existing containers stopped"
    else
      warn "failed to stop some containers, continuing with rebuild"
    fi
  fi
else
  ok "no existing containers from this profile"
fi

step "Building images (details are written to log)"
run_logged $COMPOSE_CMD -f docker-compose.yml build
ok "images built"

step "Starting containers"
run_logged $COMPOSE_CMD -f docker-compose.yml up -d
ok "containers started"

step "Checking container status"
run_logged $COMPOSE_CMD -f docker-compose.yml ps
ok "status collected"

section "Done"
echo "TARINIO is starting."
echo "Installed version: $TARGET_VERSION"
echo "Initial setup UI (temporary): http://<server-ip>:8080/login"
echo "After onboarding: https://<your-domain>/login"
echo "WAF HTTP:  http://<server-ip>/"
echo "WAF HTTPS: https://<server-ip>/"
echo "Installer log: $LOG_FILE"
echo
echo "Follow logs with:"
echo "  cd $INSTALL_DIR/deploy/compose/$PROFILE"
echo "  $COMPOSE_CMD -f docker-compose.yml logs -f"
