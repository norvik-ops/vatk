#!/usr/bin/env bash
# Performance baseline check — run after vite build
# Usage: ./scripts/bundle-check.sh [dist-dir]
set -euo pipefail

DIST="${1:-frontend/dist}"

if [ ! -d "$DIST" ]; then
  echo "ERROR: dist directory '$DIST' not found."
  echo "Run first: cd frontend && npm run build"
  exit 1
fi

echo "=== Vakt Bundle Size Check ==="
echo "Dist: $DIST"
echo ""

# Check main bundle gzip size
MAIN=$(find "$DIST/assets" -name "index-*.js" 2>/dev/null | head -1)
if [ -z "$MAIN" ]; then
  echo "WARNING: No index-*.js found in $DIST/assets — Vite output structure may have changed."
else
  SIZE=$(gzip -c "$MAIN" | wc -c)
  SIZE_KB=$((SIZE / 1024))
  echo "Main bundle (gzip): ${SIZE_KB} KB"
  if [ "$SIZE" -gt 614400 ]; then
    echo "WARNING: Main bundle > 600 KB gzip — consider code splitting"
  else
    echo "OK: Main bundle within 600 KB gzip threshold"
  fi
fi

# Report all JS chunks
echo ""
echo "All JS chunks (gzip):"
TOTAL=0
for f in "$DIST/assets/"*.js; do
  [ -f "$f" ] || continue
  CHUNK_SIZE=$(gzip -c "$f" | wc -c)
  CHUNK_KB=$((CHUNK_SIZE / 1024))
  TOTAL=$((TOTAL + CHUNK_SIZE))
  BASENAME=$(basename "$f")
  if [ "$CHUNK_SIZE" -gt 512000 ]; then
    echo "  [WARN > 500 KB] $BASENAME: ${CHUNK_KB} KB"
  else
    echo "  $BASENAME: ${CHUNK_KB} KB"
  fi
done
TOTAL_KB=$((TOTAL / 1024))
echo ""
echo "Total JS (gzip): ${TOTAL_KB} KB"

# Report CSS
echo ""
echo "CSS (gzip):"
for f in "$DIST/assets/"*.css; do
  [ -f "$f" ] || continue
  CSS_SIZE=$(gzip -c "$f" | wc -c)
  CSS_KB=$((CSS_SIZE / 1024))
  echo "  $(basename "$f"): ${CSS_KB} KB"
done

echo ""
echo "Done. To update docs/dev/performance-baseline.md, run the API latency checks manually:"
echo "  curl -w \"%{time_total}s\\n\" -o /dev/null -s http://localhost/health"
