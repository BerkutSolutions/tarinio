#!/usr/bin/env sh
set -eu

REPO_URL="${REPO_URL:-https://github.com/BerkutSolutions/tarinio.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-default}"

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
  printf "%s[%sFAIL%s] %s\n" "$C_BOLD" "$C_RED" "$C_RESET" "$1" >&2
  exit 1
}

section "TARINIO AIO Installer"
step "Checking required tools"

if command -v docker >/dev/null 2>&1; then
  ok "docker detected"
else
  fail "docker is not installed"
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
  ok "using docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
  warn "using legacy docker-compose"
else
  fail "docker compose / docker-compose is not installed"
fi

section "Repository"
if [ -d "$INSTALL_DIR/.git" ]; then
  step "Updating existing repository in $INSTALL_DIR"
  git -C "$INSTALL_DIR" fetch --all --tags
  git -C "$INSTALL_DIR" checkout "$BRANCH"
  git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
  ok "repository updated to branch $BRANCH"
else
  step "Cloning repository to $INSTALL_DIR"
  git clone --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
  ok "repository cloned"
fi

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
step "Starting TARINIO profile: $PROFILE"
$COMPOSE_CMD -f docker-compose.yml up -d --build
ok "containers started"

section "Done"
echo "TARINIO is starting."
echo "Initial setup UI (temporary): http://<server-ip>:8080/login"
echo "After onboarding: https://<your-domain>/login"
echo "WAF HTTP:  http://<server-ip>/"
echo "WAF HTTPS: https://<server-ip>/"
echo
echo "Follow logs with:"
echo "  cd $INSTALL_DIR/deploy/compose/$PROFILE"
echo "  $COMPOSE_CMD -f docker-compose.yml logs -f"
