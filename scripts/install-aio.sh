#!/usr/bin/env sh
set -eu

REPO_URL="${REPO_URL:-https://github.com/BerkutSolutions/tarinio.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-default}"
COMPAT_CONTRACT_VERSION="${COMPAT_CONTRACT_VERSION:-2026-04-08-healthcheck-v2}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/tmp/tarinio-install-${PROFILE}-${TIMESTAMP}.log"
BACKUP_BASE_DIR="${BACKUP_BASE_DIR:-/tmp/tarinio-upgrade-backups}"
BACKUP_DIR="${BACKUP_BASE_DIR}/${PROFILE}-${TIMESTAMP}"
BACKUP_MAX_TOTAL_MB="${BACKUP_MAX_TOTAL_MB:-1024}"
BACKUP_MAX_VOLUME_MB="${BACKUP_MAX_VOLUME_MB:-512}"
BACKUP_HELPER_IMAGE="${BACKUP_HELPER_IMAGE:-busybox:1.36}"
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

reset_screen() {
  target=""
  if [ -w /dev/tty ]; then
    target="/dev/tty"
  elif [ -t 1 ]; then
    target="/dev/stdout"
  fi
  if [ -z "$target" ]; then
    return
  fi
  if command -v tput >/dev/null 2>&1; then
    tput clear >"$target" 2>/dev/null || true
  fi
  printf '\033[H\033[2J\033[3J\014' >"$target"
}

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

safe_snapshot() {
  mkdir -p "$BACKUP_DIR"
  if [ -f "$INSTALL_DIR/deploy/compose/$PROFILE/.env" ]; then
    cp "$INSTALL_DIR/deploy/compose/$PROFILE/.env" "$BACKUP_DIR/.env"
  fi
  if [ -f "$INSTALL_DIR/deploy/compose/$PROFILE/docker-compose.yml" ]; then
    cp "$INSTALL_DIR/deploy/compose/$PROFILE/docker-compose.yml" "$BACKUP_DIR/docker-compose.yml"
  fi
  if [ -f "$INSTALL_DIR/control-plane/internal/appmeta/meta.go" ]; then
    cp "$INSTALL_DIR/control-plane/internal/appmeta/meta.go" "$BACKUP_DIR/meta.go"
  fi
  ok "safe snapshot saved: $BACKUP_DIR"
}

compose_capture() {
  if [ "${COMPOSE_LEGACY:-0}" -eq 1 ]; then
    docker-compose "$@" 2>>"$LOG_FILE"
  else
    docker compose "$@" 2>>"$LOG_FILE"
  fi
}

should_backup_volume() {
  name="$1"
  case "$name" in
    *runtime-data*|*control-plane-data*|*certificates-data*|*postgres-data*|*ddos-model-state*|*request-archive-data*|*l4-adaptive*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

safe_data_backup() {
  profile_dir="$INSTALL_DIR/deploy/compose/$PROFILE"
  compose_file="$profile_dir/docker-compose.yml"
  if [ ! -f "$compose_file" ]; then
    warn "docker-compose.yml not found for profile $PROFILE, skipping volume backup"
    return 0
  fi

  mkdir -p "$BACKUP_DIR/volumes"
  manifest="$BACKUP_DIR/backup-manifest.txt"
  : >"$manifest"

  volumes="$(cd "$profile_dir" && compose_capture -f docker-compose.yml config --volumes || true)"
  if [ -z "$volumes" ]; then
    warn "no named volumes found in compose config, skipping volume backup"
    return 0
  fi

  total_estimated_mb=0
  saved_count=0
  skipped_count=0

  for volume in $volumes; do
    if ! should_backup_volume "$volume"; then
      continue
    fi

    if ! docker volume inspect "$volume" >/dev/null 2>&1; then
      skipped_count=$((skipped_count + 1))
      printf "skip %s: volume not found\n" "$volume" >>"$manifest"
      continue
    fi

    estimated_mb="$(docker run --rm -v "${volume}:/data:ro" "$BACKUP_HELPER_IMAGE" sh -lc "du -sm /data 2>/dev/null | awk '{print \$1}'" 2>>"$LOG_FILE" || true)"
    estimated_mb="$(printf "%s" "$estimated_mb" | tr -d '\r' | awk 'NF{print $1; exit}')"
    case "$estimated_mb" in
      ''|*[!0-9]*) estimated_mb=0 ;;
    esac

    if [ "$estimated_mb" -gt "$BACKUP_MAX_VOLUME_MB" ]; then
      skipped_count=$((skipped_count + 1))
      warn "skip volume $volume: estimated ${estimated_mb}MB > per-volume limit ${BACKUP_MAX_VOLUME_MB}MB"
      printf "skip %s: estimated %sMB > per-volume limit %sMB\n" "$volume" "$estimated_mb" "$BACKUP_MAX_VOLUME_MB" >>"$manifest"
      continue
    fi

    projected_total=$((total_estimated_mb + estimated_mb))
    if [ "$projected_total" -gt "$BACKUP_MAX_TOTAL_MB" ]; then
      skipped_count=$((skipped_count + 1))
      warn "skip volume $volume: total backup limit ${BACKUP_MAX_TOTAL_MB}MB would be exceeded"
      printf "skip %s: total limit %sMB would be exceeded (current %sMB + %sMB)\n" "$volume" "$BACKUP_MAX_TOTAL_MB" "$total_estimated_mb" "$estimated_mb" >>"$manifest"
      continue
    fi

    safe_name="$(printf "%s" "$volume" | tr '/:' '__')"
    target_file="$BACKUP_DIR/volumes/${safe_name}.tar.gz"
    if docker run --rm -v "${volume}:/data:ro" -v "$BACKUP_DIR/volumes:/backup" "$BACKUP_HELPER_IMAGE" sh -lc "cd /data && tar -czf '/backup/${safe_name}.tar.gz' ." >>"$LOG_FILE" 2>&1; then
      total_estimated_mb="$projected_total"
      saved_count=$((saved_count + 1))
      ok "backup saved: $volume -> $target_file (~${estimated_mb}MB)"
      printf "saved %s: %s (~%sMB)\n" "$volume" "$target_file" "$estimated_mb" >>"$manifest"
    else
      skipped_count=$((skipped_count + 1))
      warn "failed to backup volume $volume (see log: $LOG_FILE)"
      printf "skip %s: backup command failed\n" "$volume" >>"$manifest"
    fi
  done

  ok "lightweight volume backup finished: saved=$saved_count skipped=$skipped_count estimated_total<=${total_estimated_mb}MB"
  ok "backup manifest: $manifest"
}

probe_container_http() {
  service="$1"
  url="$2"
  attempts="${3:-30}"
  i=1
  while [ "$i" -le "$attempts" ]; do
    if run_logged $COMPOSE_CMD -f docker-compose.yml exec -T "$service" sh -lc "wget -qO- '$url' >/dev/null 2>&1 || curl -fsS '$url' >/dev/null 2>&1"; then
      ok "$service responded: $url"
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  fail "$service health probe failed for $url after ${attempts} attempts"
}

probe_host_http() {
  url="$1"
  if command -v curl >/dev/null 2>&1; then
    run_logged curl -fsS -I "$url"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    run_logged wget -qO- "$url"
    return
  fi
  fail "curl/wget is required for host probe: $url"
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

extract_latest_changelog_version() {
  file="$1"
  if [ ! -f "$file" ]; then
    printf "unknown"
    return
  fi
  version="$(sed -n 's/^## \[\([^]]*\)\].*/\1/p' "$file" | head -n 1)"
  if [ -n "$version" ]; then
    printf "%s" "$version"
  else
    printf "unknown"
  fi
}

trap on_exit EXIT

mkdir -p "$(dirname "$LOG_FILE")"
: >"$LOG_FILE"
reset_screen()

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
CURRENT_VERSION="unknown"
CURRENT_COMMIT="-"
if [ -f "$INSTALL_DIR/control-plane/internal/appmeta/meta.go" ]; then
  CURRENT_VERSION="$(extract_version "$INSTALL_DIR/control-plane/internal/appmeta/meta.go")"
fi
if [ -d "$INSTALL_DIR/.git" ]; then
  CURRENT_COMMIT="$(git -C "$INSTALL_DIR" rev-parse --short HEAD 2>/dev/null || printf "unknown")"
  section "Safety Snapshot"
  step "Saving pre-upgrade snapshot for rollback"
  safe_snapshot
  step "Saving lightweight data backup to /tmp (size-limited)"
  safe_data_backup

  step "Updating existing repository in $INSTALL_DIR"
  run_logged git -C "$INSTALL_DIR" fetch --all --tags
  run_logged git -C "$INSTALL_DIR" checkout "$BRANCH"
  run_logged git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
  ok "repository updated to branch $BRANCH"
else
  step "Cloning repository to $INSTALL_DIR"
  run_logged git clone --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
  ok "repository cloned"
  if [ "$CURRENT_VERSION" = "unknown" ]; then
    CURRENT_VERSION="not_installed"
  fi
fi

TARGET_VERSION="$(extract_version "$INSTALL_DIR/control-plane/internal/appmeta/meta.go")"
TARGET_COMMIT="$(git -C "$INSTALL_DIR" rev-parse --short HEAD 2>/dev/null || printf "unknown")"
TARGET_CHANGELOG_VERSION="$(extract_latest_changelog_version "$INSTALL_DIR/CHANGELOG.md")"
section "Install Plan"
ok "current version/commit: $CURRENT_VERSION ($CURRENT_COMMIT)"
ok "target version: $TARGET_VERSION"
ok "target changelog head: $TARGET_CHANGELOG_VERSION"
ok "branch/commit: $BRANCH ($TARGET_COMMIT)"
ok "profile: $PROFILE"
ok "install path: $INSTALL_DIR"
ok "detailed log: $LOG_FILE"
ok "compat contract: $COMPAT_CONTRACT_VERSION"

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

section "Post-Upgrade Health Gate"
step "Probing control-plane health endpoint"
probe_container_http control-plane "http://127.0.0.1:8080/healthz" 45
step "Probing runtime health endpoint"
probe_container_http runtime "http://127.0.0.1:8081/healthz" 45
step "Probing UI healthcheck page route"
probe_host_http "http://127.0.0.1:8080/healthcheck"
ok "post-upgrade health gate passed"

section "Done"
echo "TARINIO is starting."
echo "Installed version: $TARGET_VERSION"
echo "Initial setup UI (temporary): http://<server-ip>:8080/login"
echo "After onboarding: https://<your-domain>/login"
echo "WAF HTTP:  http://<server-ip>/"
echo "WAF HTTPS: https://<server-ip>/"
echo "Installer log: $LOG_FILE"
echo "Backup dir: $BACKUP_DIR"
echo "Backup limits: per-volume=${BACKUP_MAX_VOLUME_MB}MB, total=${BACKUP_MAX_TOTAL_MB}MB"
echo "Compatibility contract: $COMPAT_CONTRACT_VERSION"
echo
echo "Follow logs with:"
echo "  cd $INSTALL_DIR/deploy/compose/$PROFILE"
echo "  $COMPOSE_CMD -f docker-compose.yml logs -f"
