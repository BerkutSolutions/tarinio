#!/usr/bin/env sh
set -eu

repo_root="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT HUP INT TERM

mkdir -p "$fixture/ui/app" "$fixture/scripts" "$fixture/deploy/compose/default" "$fixture/private"
printf '%s\n' 'events {} http {}' >"$fixture/ui/nginx.rootless.conf"
printf '%s\n' 'server { listen 80; }' >"$fixture/ui/nginx.conf"
printf '%s\n' '<!doctype html>' >"$fixture/ui/app/index.html"
printf '%s\n' '#!/usr/bin/env sh' 'exit 0' >"$fixture/scripts/legacy.sh"
printf '%s\n' 'POSTGRES_PASSWORD=secret' >"$fixture/deploy/compose/default/.env"
printf '%s\n' 'private-test-material' >"$fixture/private/client.key"

chmod 0700 "$fixture/ui" "$fixture/ui/app" "$fixture/scripts" "$fixture/deploy" "$fixture/deploy/compose" "$fixture/deploy/compose/default"
chmod 0600 "$fixture/ui/nginx.rootless.conf" "$fixture/ui/nginx.conf" "$fixture/scripts/legacy.sh" "$fixture/private/client.key"
chmod 0666 "$fixture/ui/app/index.html"
chmod 0644 "$fixture/deploy/compose/default/.env"

sh "$repo_root/scripts/repair-installation-permissions.sh" "$fixture"

require_mode() {
  expected="$1"
  target="$2"
  actual="$(stat -c '%a' "$target")"
  if [ "$actual" != "$expected" ]; then
    echo "mode mismatch for $target: got $actual want $expected" >&2
    exit 1
  fi
}

require_mode 755 "$fixture/ui"
require_mode 755 "$fixture/ui/app"
require_mode 644 "$fixture/ui/nginx.rootless.conf"
require_mode 644 "$fixture/ui/nginx.conf"
require_mode 644 "$fixture/ui/app/index.html"
require_mode 755 "$fixture/scripts/legacy.sh"
require_mode 600 "$fixture/deploy/compose/default/.env"
require_mode 600 "$fixture/private/client.key"