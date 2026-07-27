#!/usr/bin/env sh
set -eu

install_root="${1:-}"
if [ -z "$install_root" ] || [ ! -d "$install_root" ]; then
  echo "usage: repair-installation-permissions.sh <installation-root>" >&2
  exit 2
fi

chmod u+rwx,go-w,go+rx "$install_root"
for subtree in cmd compiler control-plane deploy docs e2e internal runtime scripts tools ui; do
  path="$install_root/$subtree"
  [ -d "$path" ] || continue
  find "$path" -type d -exec chmod u+rwx,go-w,go+rx {} +
  find "$path" -type f \
    ! -name '.env' ! -name '*.pem' ! -name '*.key' ! -name '*.crt' ! -name '*.p12' \
    -exec chmod u+rw,go-w,go+r {} +
done

for config in \
  "$install_root/ui/nginx.conf" \
  "$install_root/ui/nginx.rootless.conf" \
  "$install_root/ui/nginx.testpage.conf"; do
  [ -f "$config" ] && chmod 0644 "$config"
done

for script_root in "$install_root/scripts" "$install_root/deploy"; do
  [ -d "$script_root" ] || continue
  find "$script_root" -type f -name '*.sh' -exec chmod 0755 {} +
done

if [ -d "$install_root/deploy/compose" ]; then
  find "$install_root/deploy/compose" -type f -name '.env' -exec chmod 0600 {} +
fi