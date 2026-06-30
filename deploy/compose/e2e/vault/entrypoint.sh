#!/bin/sh
set -eu

LOCAL_ADDR="${VAULT_LOCAL_ADDR:-http://127.0.0.1:8200}"
CONFIG_FILE="/tmp/tarinio-vault.hcl"
BOOTSTRAP_DIR="${VAULT_BOOTSTRAP_DIR:-/vault/bootstrap}"
INIT_FILE="${BOOTSTRAP_DIR}/init.txt"
UNSEAL_FILE="${BOOTSTRAP_DIR}/unseal-key"
ROOT_TOKEN_FILE="${BOOTSTRAP_DIR}/root-token"
MOUNT_NAME="$(printf "%s" "${VAULT_MOUNT:-secret}" | tr -d '\r' | sed 's#^/*##; s#/*$##')"

mkdir -p /vault/file "$BOOTSTRAP_DIR"
chmod 700 "$BOOTSTRAP_DIR"
printf '%s\n' "${VAULT_LOCAL_CONFIG:-}" >"$CONFIG_FILE"

vault server -config="$CONFIG_FILE" &
VAULT_PID=$!

cleanup() {
  kill "$VAULT_PID" >/dev/null 2>&1 || true
}
trap cleanup INT TERM

wait_for_api() {
  i=0
  while [ "$i" -lt 60 ]; do
    set +e
    vault status -address="$LOCAL_ADDR" >/dev/null 2>&1
    code=$?
    set -e
    if [ "$code" = "0" ] || [ "$code" = "1" ] || [ "$code" = "2" ]; then
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  echo "vault bootstrap: API did not become reachable" >&2
  return 1
}

status_code() {
  set +e
  vault status -address="$LOCAL_ADDR" >/dev/null 2>&1
  code=$?
  set -e
  echo "$code"
}

status_field() {
  field="$1"
  set +e
  output="$(vault status -address="$LOCAL_ADDR" 2>/dev/null)"
  set -e
  printf '%s\n' "$output" | awk -v key="$field" '$1 == key { print $2; exit }'
}

extract_init_value() {
  key="$1"
  file="$2"
  awk -F': ' -v key="$key" '$1 == key { print $2; exit }' "$file"
}

ensure_initialized() {
  initialized="$(status_field Initialized)"
  if [ "$initialized" = "false" ] && [ ! -s "$UNSEAL_FILE" ]; then
    echo "vault bootstrap: initializing storage" >&2
    vault operator init -address="$LOCAL_ADDR" -key-shares=1 -key-threshold=1 >"$INIT_FILE"
    extract_init_value "Unseal Key 1" "$INIT_FILE" >"$UNSEAL_FILE"
    extract_init_value "Initial Root Token" "$INIT_FILE" >"$ROOT_TOKEN_FILE"
    chmod 600 "$INIT_FILE" "$UNSEAL_FILE" "$ROOT_TOKEN_FILE"
  fi
}

ensure_unsealed() {
  initialized="$(status_field Initialized)"
  sealed="$(status_field Sealed)"
  if [ "$initialized" = "true" ] && [ "$sealed" = "true" ] && [ -s "$UNSEAL_FILE" ]; then
    echo "vault bootstrap: unsealing server" >&2
    vault operator unseal -address="$LOCAL_ADDR" "$(cat "$UNSEAL_FILE")" >/dev/null
  fi
}

ensure_mount() {
  if [ -z "$MOUNT_NAME" ] || [ ! -s "$ROOT_TOKEN_FILE" ]; then
    return 0
  fi
  export VAULT_TOKEN="$(cat "$ROOT_TOKEN_FILE")"
  if vault secrets list -address="$LOCAL_ADDR" 2>/dev/null | awk '{print $1}' | grep -qx "${MOUNT_NAME}/"; then
    return 0
  fi
  echo "vault bootstrap: enabling kv-v2 mount ${MOUNT_NAME}" >&2
  vault secrets enable -address="$LOCAL_ADDR" -path="$MOUNT_NAME" -version=2 kv >/dev/null
}

wait_for_api
ensure_initialized
ensure_unsealed
ensure_mount

wait "$VAULT_PID"
