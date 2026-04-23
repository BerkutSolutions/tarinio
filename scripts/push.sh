#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
META_FILE="$ROOT_DIR/control-plane/internal/appmeta/meta.go"
MESSAGE="${1:-}"

if [[ -z "$MESSAGE" && -f "$META_FILE" ]]; then
  MESSAGE="$(sed -n 's/^var AppVersion = "\(.*\)"/v\1/p' "$META_FILE" | head -n1)"
fi

if [[ -z "$MESSAGE" ]]; then
  MESSAGE="chore: update repository"
fi

cd "$ROOT_DIR"

git add .

if git diff --cached --quiet; then
  echo "no staged changes to commit"
else
  git commit -m "$MESSAGE"
fi

git pull --rebase origin main
git push origin main

echo "push completed"
