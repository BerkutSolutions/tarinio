#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
RESULTS_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/results/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

cd "$ROOT_DIR"
docker compose --profile tools --profile observability up -d --build
docker compose exec -T toolbox /tools/provision-20-services.sh >/dev/null
docker compose exec -T toolbox /tools/benchmark-pack.sh >"$RESULTS_DIR/summary.json"
echo "Benchmark summary saved to $RESULTS_DIR/summary.json"
