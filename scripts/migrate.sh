#!/usr/bin/env bash
set -euo pipefail

SERVICE=${1:-}
DIRECTION=${2:-up}
VERSION=${3:-}

if [[ -z "$SERVICE" ]]; then
  echo "Usage: migrate.sh <service> <up|down|force> [version]"
  exit 1
fi

# Map service name to DB DSN env var
case "$SERVICE" in
  auth-svc)      DSN="${AUTH_DB_DSN:-postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable}" ;;
  workspace-svc) DSN="${WORKSPACE_DB_DSN:-postgres://postgres:postgres@localhost:5433/workspace_db?sslmode=disable}" ;;
  label-svc)     DSN="${LABEL_DB_DSN:-postgres://postgres:postgres@localhost:5434/label_db?sslmode=disable}" ;;
  agent-svc)     DSN="${AGENT_DB_DSN:-postgres://postgres:postgres@localhost:5435/agent_db?sslmode=disable}" ;;
  print-svc)     DSN="${PRINT_DB_DSN:-postgres://postgres:postgres@localhost:5436/print_db?sslmode=disable}" ;;
  *)             echo "Unknown service: $SERVICE"; exit 1 ;;
esac

MIGRATIONS_DIR="$(cd "$(dirname "$0")/.." && pwd)/migrations/$SERVICE"

if [[ ! -d "$MIGRATIONS_DIR" ]]; then
  echo "No migrations directory found for $SERVICE at $MIGRATIONS_DIR"
  exit 1
fi

echo "Running '$DIRECTION' migration for $SERVICE..."

case "$DIRECTION" in
  up)    migrate -path "$MIGRATIONS_DIR" -database "$DSN" up ;;
  down)  migrate -path "$MIGRATIONS_DIR" -database "$DSN" down 1 ;;
  force) migrate -path "$MIGRATIONS_DIR" -database "$DSN" force "$VERSION" ;;
  *)     echo "Unknown direction: $DIRECTION (use: up|down|force)"; exit 1 ;;
esac

echo "Done: $SERVICE migration '$DIRECTION'"
