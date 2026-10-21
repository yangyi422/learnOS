#!/usr/bin/env bash
# LearnOS deployment script for Linux hosts (cloud server) and WSL2.
#
# It builds the current checkout with Docker Compose, waits for the container
# health check, then verifies the served frontend and (when sqlite3 is
# available) the migrated database. The same script is used by the Windows
# wrapper scripts/deploy-local.ps1.
#
# Usage:
#   bash deploy/deploy.sh                  # build + deploy + verify
#   bash deploy/deploy.sh --pull           # git pull --ff-only first
#   bash deploy/deploy.sh --no-verify      # skip post-deploy checks
#   bash deploy/deploy.sh --with-caddy     # also start the production profile (Caddy HTTPS)
#   bash deploy/deploy.sh --help
#
# Requirements: docker with the compose plugin, curl (recommended), a .env file
# in the repository root (see .env.example).
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
cd "$ROOT"

PULL=0
VERIFY=1
WITH_CADDY=0
for arg in "$@"; do
  case "$arg" in
    --pull) PULL=1 ;;
    --no-verify) VERIFY=0 ;;
    --with-caddy) WITH_CADDY=1 ;;
    -h|--help)
      sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown option: $arg" >&2
      exit 2
      ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker not found. Local Windows users: run this through WSL (see scripts/deploy-local.ps1)." >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "error: docker compose plugin not found." >&2
  exit 1
fi
if [ ! -f .env ]; then
  echo "error: .env not found. Copy .env.example and configure it before deploying." >&2
  exit 1
fi

if [ "$PULL" = 1 ]; then
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    echo "error: --pull requires a git checkout." >&2
    exit 1
  fi
  if [ -n "$(git status --porcelain)" ]; then
    echo "error: working tree is not clean; commit/stash before --pull." >&2
    git status --short >&2
    exit 1
  fi
  echo "== git pull --ff-only =="
  git pull --ff-only
fi

read_env() {
  # Read one value from .env without executing it. Returns empty when missing.
  local key=$1
  [ -f .env ] || return 0
  grep -E "^${key}=" .env | tail -1 | cut -d= -f2- || true
}

APP_BIND_HOST=$(read_env APP_BIND_HOST)
APP_PORT=$(read_env APP_PORT)
APP_BIND_HOST=${APP_BIND_HOST:-0.0.0.0}
APP_PORT=${APP_PORT:-8888}
CURL_HOST=$APP_BIND_HOST
case "$CURL_HOST" in
  ""|0.0.0.0|"::") CURL_HOST=127.0.0.1 ;;
esac
URL="http://${CURL_HOST}:${APP_PORT}"

echo "== docker compose up -d --build =="
if [ "$WITH_CADDY" = 1 ]; then
  docker compose --profile production up -d --build
else
  docker compose up -d --build app
fi

CONTAINER_ID=$(docker compose ps -q app)
if [ -z "$CONTAINER_ID" ] && [ -n "${COMPOSE_PROJECT_NAME:-}" ]; then
  CONTAINER_ID=$(docker compose ps -q app 2>/dev/null || true)
fi
if [ -z "$CONTAINER_ID" ]; then
  echo "error: app container was not created." >&2
  docker compose logs app --tail 80 >&2 || true
  exit 1
fi

echo "== waiting for health =="
HEALTHY=0
for _ in $(seq 1 90); do
  STATUS=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$CONTAINER_ID" 2>/dev/null || echo unknown)
  if [ "$STATUS" = "healthy" ] || [ "$STATUS" = "running" ]; then
    if command -v curl >/dev/null 2>&1; then
      if curl -fsS --max-time 2 "$URL/health" >/dev/null 2>&1; then
        HEALTHY=1
        break
      fi
    else
      HEALTHY=1
      break
    fi
  fi
  sleep 1
done
if [ "$HEALTHY" != 1 ]; then
  echo "error: app did not become healthy within 90s." >&2
  docker compose ps >&2 || true
  docker compose logs app --tail 120 >&2 || true
  exit 1
fi
echo "health: $URL/health -> ok"

if [ "$VERIFY" = 1 ]; then
  echo "== runtime verification =="
  if command -v curl >/dev/null 2>&1; then
    OLD_ASSET="KVVFApzX"
    ASSET=$(curl -fsS --max-time 10 "$URL/" | grep -o 'index-[A-Za-z0-9_-]*\.js' | head -1 || true)
    echo "frontend asset: ${ASSET:-unknown} (pre-DailyFocus baseline: index-${OLD_ASSET}.js)"
    if [ -n "$ASSET" ] && curl -fsS --max-time 20 "$URL/assets/$ASSET" | grep -q '今日焦点'; then
      echo "bundle marker 今日焦点: present"
    else
      echo "warning: bundle marker 今日焦点 not found; verify the frontend build." >&2
    fi
    API_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$URL/api/v1/workspace/daily-focus?date=$(date +%F)&context=all" || true)
    echo "unauthenticated GET /api/v1/workspace/daily-focus: HTTP ${API_CODE:-unknown} (401 expected in production)"
  else
    echo "curl not found; skipping HTTP verification."
  fi

  DATA_HOST_DIR=$(read_env DATA_HOST_DIR)
  DATA_HOST_DIR=${DATA_HOST_DIR:-./data/prod}
  DB_PATH="$ROOT/$DATA_HOST_DIR/learnos.db"
  if command -v sqlite3 >/dev/null 2>&1 && [ -f "$DB_PATH" ]; then
    echo "database schema:"
    sqlite3 "$DB_PATH" "select '  schema_version='||schema_version, '  app_version='||app_version from system_metadata where id=1;"
    sqlite3 "$DB_PATH" "select '  daily_focuses='||count(*) from daily_focuses;" || true
    if [ -d "$ROOT/backups" ]; then
      echo "recent backups:"
      ls -lt "$ROOT/backups" | head -4 | sed 's/^/  /'
    fi
  else
    echo "database check skipped (sqlite3 or $DB_PATH unavailable on this host)."
  fi
fi

echo "== container status =="
docker compose ps
echo "deploy finished: $URL"
