#!/usr/bin/env sh
set -eu
umask 077

PROFILE="${PROFILE:-default}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="${LOG_FILE:-/tmp/tarinio-rotate-secrets-${PROFILE}-${TIMESTAMP}.log}"
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
    printf "%s[%sFAIL%s] Secret rotation aborted (exit code %s). Log: %s%s\n" "$C_BOLD" "$C_RED" "$C_RESET" "$code" "$LOG_FILE" "$C_RESET" >&2
  fi
}

trap on_exit EXIT
mkdir -p "$(dirname "$LOG_FILE")"
: >"$LOG_FILE"

run_logged() {
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
  trap 'rm -f "$tmp_file"' HUP INT TERM
  : >"$tmp_file"
  chmod 600 "$tmp_file"
  if [ ! -f .env ]; then
    printf '%s=%s\n' "$key" "$value" >"$tmp_file"
    mv "$tmp_file" .env
    chmod 600 .env
    trap - HUP INT TERM
    return 0
  fi
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
  chmod 600 .env
  trap - HUP INT TERM
}

shell_quote() {
  printf "'%s'" "$(printf "%s" "$1" | sed "s/'/'\\''/g")"
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

probe_container_http() {
  service="$1"
  url="$2"
  attempts="${3:-30}"
  i=1
  while [ "$i" -le "$attempts" ]; do
    if run_logged $COMPOSE_CMD -f docker-compose.yml exec -T "$service" sh -lc "wget -qO- '$url' >/dev/null 2>&1 || curl -fsS '$url' >/dev/null 2>&1"; then
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  return 1
}

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  fail "docker compose / docker-compose is not installed"
fi

PROFILE_DIR="$INSTALL_DIR/deploy/compose/$PROFILE"
if [ ! -d "$PROFILE_DIR" ]; then
  fail "profile directory not found: $PROFILE_DIR"
fi
cd "$PROFILE_DIR"
if [ ! -f .env ]; then
  fail ".env not found in $PROFILE_DIR"
fi
if [ ! -f docker-compose.yml ]; then
  fail "docker-compose.yml not found in $PROFILE_DIR"
fi

section "TARINIO Secret Rotation"
step "Preparing secret rotation for profile $PROFILE"
ok "profile directory: $PROFILE_DIR"
ok "log file: $LOG_FILE"

postgres_user="$(read_env_value POSTGRES_USER)"
postgres_db="$(read_env_value POSTGRES_DB)"
current_postgres_password="$(read_env_value POSTGRES_PASSWORD)"
current_clickhouse_password="$(read_env_value CLICKHOUSE_PASSWORD)"
current_opensearch_password="$(read_env_value OPENSEARCH_PASSWORD)"
current_security_pepper="$(read_env_value CONTROL_PLANE_SECURITY_PEPPER)"
current_runtime_api_token="$(read_env_value WAF_RUNTIME_API_TOKEN)"

if [ -z "$postgres_user" ]; then
  postgres_user="waf"
fi
if [ -z "$postgres_db" ]; then
  postgres_db="waf"
fi

new_postgres_password="$(generate_secret 40)"
new_clickhouse_password="$(generate_secret 40)"
new_opensearch_password="$(generate_secret 40)"
new_security_pepper="$(generate_secret 64)"
new_runtime_api_token="$(generate_secret 48)"
new_postgres_dsn="postgres://${postgres_user}:${new_postgres_password}@postgres:5432/${postgres_db}?sslmode=disable"

step "Rotating secrets in .env"
write_env_value POSTGRES_PASSWORD "$new_postgres_password"
write_env_value POSTGRES_DSN "$new_postgres_dsn"
write_env_value OPENSEARCH_PASSWORD "$new_opensearch_password"
write_env_value CONTROL_PLANE_SECURITY_PEPPER "$new_security_pepper"
write_env_value WAF_RUNTIME_API_TOKEN "$new_runtime_api_token"
if grep -q '^CLICKHOUSE_PASSWORD=' .env 2>/dev/null; then
  write_env_value CLICKHOUSE_PASSWORD "$new_clickhouse_password"
fi
ok ".env updated with fresh secrets"

step "Stopping application containers before password sync"
run_logged $COMPOSE_CMD -f docker-compose.yml stop control-plane runtime ui sentinel || true
ok "application containers stopped"

step "Ensuring postgres container is up for password rotation"
run_logged $COMPOSE_CMD -f docker-compose.yml up -d postgres
ok "postgres started"

step "Applying new postgres password inside database"
recovered_password=""
for candidate in "$current_postgres_password" "waf" "change-me-strong-password"; do
  if [ -z "$candidate" ]; then
    continue
  fi
  if postgres_probe_password "$candidate" "$postgres_user" "$postgres_db"; then
    recovered_password="$candidate"
    break
  fi
done
if [ -z "$recovered_password" ]; then
  fail "could not authenticate to postgres with current .env or known legacy fallbacks"
fi
postgres_set_password "$recovered_password" "$new_postgres_password" "$postgres_user" "$postgres_db"
ok "postgres password rotated"

step "Restarting full stack with rotated secrets"
run_logged $COMPOSE_CMD -f docker-compose.yml up -d
ok "containers started"

step "Verifying health after rotation"
probe_container_http control-plane "http://127.0.0.1:8080/healthz" 45 || fail "control-plane health check failed after secret rotation"
probe_container_http runtime "http://127.0.0.1:8081/healthz" 45 || fail "runtime health check failed after secret rotation"
ok "stack health verified after secret rotation"

section "Done"
ok "secrets rotated successfully"
printf 'Profile: %s\n' "$PROFILE"
printf 'Log file: %s\n' "$LOG_FILE"
