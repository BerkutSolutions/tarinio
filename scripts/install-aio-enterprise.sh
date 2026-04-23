#!/usr/bin/env sh
set -eu

REPO_URL="${REPO_URL:-https://github.com/BerkutSolutions/tarinio.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-enterprise}"
COMPAT_CONTRACT_VERSION="${COMPAT_CONTRACT_VERSION:-2026-04-19-healthcheck-v3}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/tmp/tarinio-install-${PROFILE}-${TIMESTAMP}.log"
BACKUP_BASE_DIR="${BACKUP_BASE_DIR:-/tmp/tarinio-upgrade-backups}"
BACKUP_DIR="${BACKUP_BASE_DIR}/${PROFILE}-${TIMESTAMP}"
BACKUP_MAX_TOTAL_MB="${BACKUP_MAX_TOTAL_MB:-1024}"
BACKUP_MAX_VOLUME_MB="${BACKUP_MAX_VOLUME_MB:-512}"
BACKUP_HELPER_IMAGE="${BACKUP_HELPER_IMAGE:-busybox:1.36}"
RUN_STRICT_POST_UPGRADE_VALIDATION="${RUN_STRICT_POST_UPGRADE_VALIDATION:-0}"
CONTAINER_PROBE_ATTEMPTS="${CONTAINER_PROBE_ATTEMPTS:-45}"
HOST_PROBE_ATTEMPTS="${HOST_PROBE_ATTEMPTS:-45}"
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
  printf '\033c' >"$target" 2>/dev/null || true
  printf '\033[H\033[2J\033[3J' >"$target" 2>/dev/null || true
  if command -v tput >/dev/null 2>&1; then
    tput reset >"$target" 2>/dev/null || true
    tput clear >"$target" 2>/dev/null || true
  fi
  i=0
  while [ "$i" -lt 120 ]; do
    printf '\n' >"$target" 2>/dev/null || true
    i=$((i + 1))
  done
  printf '\033[H\033[2J\033[3J\014' >"$target" 2>/dev/null || true
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

generate_secret() {
  length="${1:-48}"
  if command -v openssl >/dev/null 2>&1; then
    bytes=$((length + 8))
    openssl rand -base64 "$bytes" 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c "$length"
    return 0
  fi
  if [ -r /dev/urandom ]; then
    tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$length"
    return 0
  fi
  date +%s | sha256sum | awk '{print $1}' | head -c "$length"
}

read_env_value() {
  key="$1"
  if [ ! -f .env ]; then
    return 0
  fi
  awk -F= -v key="$key" '$1 == key { value=$0; sub("^[^=]*=", "", value); print value }' .env | tail -n 1
}

write_env_value() {
  key="$1"
  value="$2"
  tmp_file=".env.tmp.$$"
  awk -F= -v key="$key" -v value="$value" '
    BEGIN { written = 0 }
    $1 == key {
      print key "=" value
      written = 1
      next
    }
    { print }
    END {
      if (!written) {
        print key "=" value
      }
    }
  ' .env >"$tmp_file"
  mv "$tmp_file" .env
}

is_placeholder_secret() {
  value="$(printf "%s" "$1" | tr -d '\r')"
  case "$value" in
    ""|change-me*|default|changeme|please-change*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

ensure_secure_env_defaults() {
  changed=0

  postgres_user="$(read_env_value POSTGRES_USER)"
  postgres_db="$(read_env_value POSTGRES_DB)"
  postgres_password="$(read_env_value POSTGRES_PASSWORD)"
  clickhouse_password="$(read_env_value CLICKHOUSE_PASSWORD)"
  opensearch_password="$(read_env_value OPENSEARCH_PASSWORD)"
  security_pepper="$(read_env_value CONTROL_PLANE_SECURITY_PEPPER)"
  postgres_dsn="$(read_env_value POSTGRES_DSN)"

  if [ -z "$postgres_user" ]; then
    postgres_user="waf"
    write_env_value POSTGRES_USER "$postgres_user"
    changed=1
  fi
  if [ -z "$postgres_db" ]; then
    postgres_db="waf"
    write_env_value POSTGRES_DB "$postgres_db"
    changed=1
  fi
  if is_placeholder_secret "$postgres_password"; then
    postgres_password="$(generate_secret 40)"
    write_env_value POSTGRES_PASSWORD "$postgres_password"
    changed=1
    ok "generated secure POSTGRES_PASSWORD"
  fi
  if is_placeholder_secret "$clickhouse_password"; then
    clickhouse_password="$(generate_secret 40)"
    write_env_value CLICKHOUSE_PASSWORD "$clickhouse_password"
    changed=1
    ok "generated secure CLICKHOUSE_PASSWORD"
  fi
  if is_placeholder_secret "$opensearch_password"; then
    opensearch_password="$(generate_secret 40)"
    write_env_value OPENSEARCH_PASSWORD "$opensearch_password"
    changed=1
    ok "generated secure OPENSEARCH_PASSWORD"
  fi
  if is_placeholder_secret "$security_pepper"; then
    security_pepper="$(generate_secret 64)"
    write_env_value CONTROL_PLANE_SECURITY_PEPPER "$security_pepper"
    changed=1
    ok "generated secure CONTROL_PLANE_SECURITY_PEPPER"
  fi

  target_postgres_dsn="postgres://${postgres_user}:${postgres_password}@postgres:5432/${postgres_db}?sslmode=disable"
  if [ -z "$postgres_dsn" ] || printf "%s" "$postgres_dsn" | grep -q "change-me"; then
    write_env_value POSTGRES_DSN "$target_postgres_dsn"
    changed=1
    ok "normalized POSTGRES_DSN for current credentials"
  fi

  if [ "$changed" -eq 1 ]; then
    ok ".env security defaults normalized"
  else
    ok ".env security defaults already set"
  fi
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

shell_quote() {
  printf "'%s'" "$(printf "%s" "$1" | sed "s/'/'\\\\''/g")"
}

postgres_dsn_password() {
  dsn="$1"
  if [ -z "$dsn" ]; then
    return 0
  fi
  printf "%s" "$dsn" | sed -n 's#^postgres://[^:]*:\([^@]*\)@.*#\1#p' | head -n 1
}

postgres_probe_password() {
  password="$1"
  user="$2"
  db="$3"
  pwd_q="$(shell_quote "$password")"
  user_q="$(shell_quote "$user")"
  db_q="$(shell_quote "$db")"
  cmd="PGPASSWORD=$pwd_q psql -h 127.0.0.1 -U $user_q -d $db_q -v ON_ERROR_STOP=1 -tAc 'select 1' >/dev/null 2>&1"
  run_logged $COMPOSE_CMD -f docker-compose.yml exec -T postgres sh -lc "$cmd"
}

postgres_set_password() {
  current_password="$1"
  desired_password="$2"
  user="$3"
  db="$4"
  desired_sql="$(printf "%s" "$desired_password" | sed "s/'/''/g")"
  sql="ALTER ROLE \"$user\" WITH PASSWORD '$desired_sql';"
  current_q="$(shell_quote "$current_password")"
  user_q="$(shell_quote "$user")"
  db_q="$(shell_quote "$db")"
  sql_q="$(shell_quote "$sql")"
  cmd="PGPASSWORD=$current_q psql -h 127.0.0.1 -U $user_q -d $db_q -v ON_ERROR_STOP=1 -tAc $sql_q"
  run_logged $COMPOSE_CMD -f docker-compose.yml exec -T postgres sh -lc "$cmd"
}

ensure_postgres_password_alignment() {
  postgres_user="$(read_env_value POSTGRES_USER)"
  postgres_db="$(read_env_value POSTGRES_DB)"
  desired_password="$(read_env_value POSTGRES_PASSWORD)"
  postgres_dsn="$(read_env_value POSTGRES_DSN)"
  dsn_password="$(postgres_dsn_password "$postgres_dsn")"

  if [ -z "$postgres_user" ]; then
    postgres_user="waf"
  fi
  if [ -z "$postgres_db" ]; then
    postgres_db="waf"
  fi
  if [ -z "$desired_password" ]; then
    desired_password="waf"
  fi

  if postgres_probe_password "$desired_password" "$postgres_user" "$postgres_db"; then
    ok "postgres credentials verified"
    return 0
  fi

  recovered_password=""
  for candidate in "$dsn_password" "waf" "change-me-strong-password"; do
    if [ -z "$candidate" ] || [ "$candidate" = "$desired_password" ]; then
      continue
    fi
    if postgres_probe_password "$candidate" "$postgres_user" "$postgres_db"; then
      recovered_password="$candidate"
      break
    fi
  done

  if [ -z "$recovered_password" ]; then
    warn "postgres credentials probe failed for configured password and known legacy fallbacks"
    return 0
  fi

  warn "postgres password mismatch detected; aligning database user password with current .env"
  postgres_set_password "$recovered_password" "$desired_password" "$postgres_user" "$postgres_db"
  ok "postgres user password updated to match current .env"

  target_postgres_dsn="postgres://${postgres_user}:${desired_password}@postgres:5432/${postgres_db}?sslmode=disable"
  write_env_value POSTGRES_DSN "$target_postgres_dsn"
  ok "POSTGRES_DSN normalized after postgres password recovery"

  if run_logged $COMPOSE_CMD -f docker-compose.yml restart control-plane; then
    ok "control-plane restarted after postgres credential recovery"
  else
    warn "failed to restart control-plane after postgres credential recovery"
  fi
}

should_backup_volume() {
  name="$1"
  case "$name" in
    *runtime-data*|*control-plane-data*|*certificates-data*|*postgres-data*|*clickhouse-data*|*opensearch-data*|*vault-data*|*vault-bootstrap-data*|*ddos-model-state*|*request-archive-data*|*l4-adaptive*)
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
  attempts="${2:-30}"
  i=1
  while [ "$i" -le "$attempts" ]; do
    if command -v curl >/dev/null 2>&1; then
      if run_logged curl -fsS -I "$url"; then
        ok "host responded: $url"
        return 0
      fi
    elif command -v wget >/dev/null 2>&1; then
      if run_logged wget --server-response --spider "$url"; then
        ok "host responded: $url"
        return 0
      fi
    else
      fail "curl/wget is required for host probe: $url"
    fi
    sleep 2
    i=$((i + 1))
  done
  fail "host probe failed for $url after ${attempts} attempts"
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
step "Normalizing runtime secrets and secure defaults"
ensure_secure_env_defaults

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

step "Verifying Postgres credentials"
ensure_postgres_password_alignment

step "Checking container status"
run_logged $COMPOSE_CMD -f docker-compose.yml ps
ok "status collected"

section "Post-Upgrade Health Gate"
step "Probing control-plane health endpoint"
probe_container_http control-plane "http://127.0.0.1:8080/healthz" "$CONTAINER_PROBE_ATTEMPTS"
step "Probing runtime health endpoint"
probe_container_http runtime "http://127.0.0.1:8081/healthz" "$CONTAINER_PROBE_ATTEMPTS"
step "Probing public runtime gateway route"
probe_host_http "http://127.0.0.1/" "$HOST_PROBE_ATTEMPTS"
ok "post-upgrade health gate passed"

if [ "$RUN_STRICT_POST_UPGRADE_VALIDATION" = "1" ] || [ "$RUN_STRICT_POST_UPGRADE_VALIDATION" = "true" ]; then
  section "Strict Post-Upgrade Validation"
  step "Running scripts/post-upgrade-smoke.sh"
  run_logged env "PROFILE_DIR=$(pwd)" "COMPOSE_CMD=$COMPOSE_CMD" sh "$INSTALL_DIR/scripts/post-upgrade-smoke.sh"
  ok "strict post-upgrade validation passed"
fi

section "Done"
printf '%s\n' "TARINIO is starting."
printf 'Installed version: %s\n' "$TARGET_VERSION"
printf '%s\n' 'Initial setup: http://<server-ip>/'
printf '%s\n' 'After successful onboarding: https://<your-domain>/login'
printf '%s\n' 'WAF HTTP:  http://<server-ip>/'
printf '%s\n' 'WAF HTTPS: https://<server-ip>/'
printf 'Installer log: %s\n' "$LOG_FILE"
printf 'Backup dir: %s\n' "$BACKUP_DIR"
printf 'Backup limits: per-volume=%sMB, total=%sMB\n' "$BACKUP_MAX_VOLUME_MB" "$BACKUP_MAX_TOTAL_MB"
printf 'Compatibility contract: %s\n' "$COMPAT_CONTRACT_VERSION"
printf '\n'
printf '%s\n' 'Follow logs with:'
printf '  cd %s/deploy/compose/%s\n' "$INSTALL_DIR" "$PROFILE"
printf '  %s -f docker-compose.yml logs -f\n' "$COMPOSE_CMD"
