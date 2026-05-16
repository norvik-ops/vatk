#!/usr/bin/env bash
# SecHealth — Production Update Script
#
# Usage:
#   ./scripts/update.sh [--force]
#
# What it does:
#   1. Checks for a newer version on GitHub
#   2. Creates a database backup before updating
#   3. Pulls new images and restarts services with zero-downtime rollout
#   4. Runs database migrations automatically (AUTO_MIGRATE=true required)
#   5. Verifies the deployment is healthy before finishing
#
# Requirements:
#   - docker, docker compose, curl, jq
#   - Run from the sechealth deployment directory (where docker-compose.yml lives)

set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
GITHUB_REPO="sechealth-app/sechealth"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
FORCE="${1:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# --- Dependency checks ---
for cmd in docker curl jq; do
  command -v "$cmd" &>/dev/null || error "$cmd is required but not installed"
done

docker compose version &>/dev/null || error "docker compose plugin is required"

# --- Determine current and latest version ---
CURRENT_VERSION=$(docker compose -f "$COMPOSE_FILE" exec -T api /app/version 2>/dev/null || echo "unknown")

info "Checking latest release from GitHub..."
LATEST_JSON=$(curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null || echo "{}")
LATEST_VERSION=$(echo "$LATEST_JSON" | jq -r '.tag_name // "unknown"' | sed 's/^v//')

if [[ "$LATEST_VERSION" == "unknown" ]]; then
  warn "Could not reach GitHub. Proceeding with update anyway."
elif [[ "$CURRENT_VERSION" == "$LATEST_VERSION" ]] && [[ "$FORCE" != "--force" ]]; then
  info "Already at the latest version (${CURRENT_VERSION}). Use --force to update anyway."
  exit 0
fi

info "Updating: ${CURRENT_VERSION} → ${LATEST_VERSION}"

# --- Pre-update backup ---
info "Creating pre-update database backup..."
mkdir -p "$BACKUP_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/pre-update-${TIMESTAMP}.sql.gz"

if docker compose -f "$COMPOSE_FILE" exec -T postgres \
    pg_dump -U sechealth sechealth 2>/dev/null | gzip > "$BACKUP_FILE"; then
  info "Backup saved to $BACKUP_FILE"
else
  warn "Backup failed — continue? [y/N]"
  read -r REPLY
  [[ "$REPLY" =~ ^[Yy]$ ]] || error "Aborted by user."
fi

# --- Pull new images ---
info "Pulling latest images..."
docker compose -f "$COMPOSE_FILE" pull api worker

# --- Rolling restart: worker first, then api ---
info "Restarting worker..."
docker compose -f "$COMPOSE_FILE" up -d --no-deps worker

info "Restarting api (migrations run automatically)..."
docker compose -f "$COMPOSE_FILE" up -d --no-deps api

# --- Health check ---
info "Waiting for API to become healthy..."
RETRIES=30
until docker compose -f "$COMPOSE_FILE" exec -T api \
    wget -qO- http://localhost:8080/health/ready 2>/dev/null | grep -q ready; do
  RETRIES=$((RETRIES - 1))
  [[ $RETRIES -le 0 ]] && error "API did not become healthy in time. Roll back with: docker compose -f $COMPOSE_FILE down && docker compose -f $COMPOSE_FILE up -d"
  sleep 2
done

# --- Rebuild and reload nginx with new frontend ---
info "Rebuilding frontend and reloading nginx..."
docker compose -f "$COMPOSE_FILE" up -d --no-deps --build frontend
docker compose -f "$COMPOSE_FILE" exec -T nginx nginx -s reload 2>/dev/null || true

info "Update complete. Running version: $(docker compose -f "$COMPOSE_FILE" exec -T api /app/version 2>/dev/null || echo "check logs")"
info "If anything looks wrong, restore from: $BACKUP_FILE"
