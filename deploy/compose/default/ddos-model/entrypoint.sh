#!/bin/sh
set -eu

mkdir -p /state /out
chown -R 65532:4 /state /out
chmod 0770 /state /out

exec su tarinio -s /bin/sh -c 'exec /usr/local/bin/tarinio-sentinel'
