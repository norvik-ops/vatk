#!/usr/bin/env bash
set -euo pipefail

BACKUP_FILE="${1:-}"
if [ -z "$BACKUP_FILE" ] || [ ! -f "$BACKUP_FILE" ]; then
  echo "ERROR: Usage: $0 <backup-file.tar.gz>" >&2
  exit 1
fi

if [ -f .env ]; then
  set -a; source .env; set +a
fi

DB_URL="${VAKT_DB_URL:-}"
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "→ Extracting..."
tar -xzf "$BACKUP_FILE" -C "$WORK_DIR"

echo "→ Verifying manifest..."
cat "$WORK_DIR/manifest.json"

echo "→ Checking dump integrity..."
pg_restore --list "$WORK_DIR/db.pgdump" > /dev/null && echo "✓ Dump is valid"

echo "✓ Backup verification passed: $BACKUP_FILE"
