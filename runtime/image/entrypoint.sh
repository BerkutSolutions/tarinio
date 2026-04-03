#!/bin/sh
set -eu

if [ "${WAF_L4_GUARD_ENABLED:-true}" != "false" ] && [ "${WAF_L4_GUARD_ENABLED:-true}" != "0" ]; then
  /usr/local/bin/waf-runtime-l4-guard bootstrap
fi

exec /usr/local/bin/waf-runtime-launcher
