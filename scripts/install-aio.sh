#!/usr/bin/env sh
set -eu

REPO_URL="${REPO_URL:-https://github.com/BerkutSolutions/tarinio.git}"
INSTALL_DIR="${INSTALL_DIR:-/opt/tarinio}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-default}"

if command -v docker >/dev/null 2>&1; then
  :
else
  echo "docker is not installed" >&2
  exit 1
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  echo "docker compose / docker-compose is not installed" >&2
  exit 1
fi

if [ -d "$INSTALL_DIR/.git" ]; then
  git -C "$INSTALL_DIR" fetch --all --tags
  git -C "$INSTALL_DIR" checkout "$BRANCH"
  git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH"
else
  git clone --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
fi

cd "$INSTALL_DIR/deploy/compose/$PROFILE"

if [ ! -f .env ]; then
  cp .env.example .env
fi

echo "Starting TARINIO profile: $PROFILE"
$COMPOSE_CMD -f docker-compose.yml up -d --build

echo
echo "TARINIO is starting."
echo "HTTP:  http://<server-ip>/login"
echo "HTTPS: https://<server-ip>/login"
echo
echo "Follow logs with:"
echo "  cd $INSTALL_DIR/deploy/compose/$PROFILE"
echo "  $COMPOSE_CMD -f docker-compose.yml logs -f"
