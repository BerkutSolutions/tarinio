#!/usr/bin/env bash
set -euo pipefail

ARCHIVE_DIR="/archive"
ARCHIVE_FILE="${ARCHIVE_DIR}/requests.jsonl"

MAX_FILE_SIZE_MB="${REQUEST_ARCHIVE_MAX_FILE_SIZE_MB:-256}"
MAX_FILES="${REQUEST_ARCHIVE_MAX_FILES:-7}"
RETENTION_DAYS="${REQUEST_ARCHIVE_RETENTION_DAYS:-2}"

mkdir -p "${ARCHIVE_DIR}" /logs/mgmt /logs/app
touch "${ARCHIVE_FILE}"

rotate_archive() {
  local size_bytes
  local max_bytes
  size_bytes=$(wc -c < "${ARCHIVE_FILE}" || echo "0")
  max_bytes=$((MAX_FILE_SIZE_MB * 1024 * 1024))
  if [[ "${size_bytes}" -lt "${max_bytes}" ]]; then
    return
  fi

  local i
  for ((i=MAX_FILES; i>=1; i--)); do
    local src="${ARCHIVE_FILE}.${i}"
    local dst="${ARCHIVE_FILE}.$((i + 1))"
    if [[ -f "${src}" ]]; then
      mv "${src}" "${dst}"
    fi
  done
  mv "${ARCHIVE_FILE}" "${ARCHIVE_FILE}.1"
  : > "${ARCHIVE_FILE}"
}

cleanup_old_archives() {
  find "${ARCHIVE_DIR}" -type f -name "requests.jsonl*" -mtime "+${RETENTION_DAYS}" -delete || true
}

append_line() {
  local stream="$1"
  local line="$2"
  local ts
  ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  if [[ "${line}" =~ ^\{.*\}$ ]]; then
    printf '{"stream":"%s","ingested_at":"%s","entry":%s}\n' "${stream}" "${ts}" "${line}" >> "${ARCHIVE_FILE}"
  else
    local escaped
    escaped="$(printf '%s' "${line}" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')"
    printf '{"stream":"%s","ingested_at":"%s","raw":"%s"}\n' "${stream}" "${ts}" "${escaped}" >> "${ARCHIVE_FILE}"
  fi
  rotate_archive
}

follow_stream() {
  local stream="$1"
  local file="$2"
  while true; do
    if [[ ! -f "${file}" ]]; then
      sleep 1
      continue
    fi
    tail -n0 -F "${file}" | while IFS= read -r line; do
      append_line "${stream}" "${line}"
    done || true
    sleep 1
  done
}

follow_stream "mgmt" "/logs/mgmt/access.log" &
follow_stream "app" "/logs/app/access.log" &

while true; do
  sleep 3600
  cleanup_old_archives
done
