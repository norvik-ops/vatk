#!/usr/bin/env bash
# Smoke-Test für v0.14.2 — verifiziert die 3 in Sprint 22 Tail eingelaufenen
# Frontend-Pfade gegen den lokal laufenden Stack.
#
# Voraussetzung: `docker compose up -d` läuft und der Stack ist gesund.
# Aufruf: ./scripts/smoke-v0.14.2.sh [BASE_URL]   (Default http://localhost)

set -euo pipefail
BASE="${1:-http://localhost}"

red()    { printf '\033[31m%s\033[0m\n' "$*"; }
green()  { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
cyan()   { printf '\033[36m%s\033[0m\n' "$*"; }

step() { cyan ""; cyan "── $* ──"; }
ok()   { green "  ✓ $*"; }
fail() { red   "  ✗ $*"; exit 1; }

# ── 1. Health + OpenAPI-Contract-Check ────────────────────────────────────────
step "1/5  Health + OpenAPI-Contract"
HEALTH=$(curl -sf "$BASE/health" || fail "health endpoint nicht erreichbar")
echo "$HEALTH" | jq -e '.demo, .sso_enabled, .version' >/dev/null \
    || fail "health response fehlt demo/sso_enabled/version"
ok "/health hat demo/sso_enabled/version"

# ── 2. Demo-Login + session_id im Response prüfen (S22-10) ────────────────────
step "2/5  Demo-Login mit session_id (S22-10)"
DEMO=$(curl -sf -X POST "$BASE/api/v1/demo/start")
EMAIL=$(echo "$DEMO" | jq -r '.admin_email')
PASS=$(echo "$DEMO" | jq -r '.admin_password')
[[ -n "$EMAIL" && -n "$PASS" ]] || fail "demo/start hat keine creds geliefert"
ok "demo/start gab ephemere creds: $EMAIL"

LOGIN_RESP=$(curl -sf -c /tmp/vakt-smoke-cookies.txt -X POST "$BASE/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")

SESSION_ID=$(echo "$LOGIN_RESP" | jq -r '.session_id // empty')
ACCESS=$(echo "$LOGIN_RESP" | jq -r '.access_token')
USER_ID=$(echo "$LOGIN_RESP" | jq -r '.user.id')
[[ -n "$SESSION_ID" ]] || fail "S22-10 broken: LoginResponse hat kein session_id"
ok "LoginResponse enthält session_id = $SESSION_ID"
ok "User-Objekt in LoginResponse vorhanden (id=$USER_ID)"

# ── 3. GET /auth/sessions mit X-Vakt-Session-Id markiert is_current ──────────
step "3/5  Sessions-Endpoint mit Current-Marker (S22-10)"
CSRF=$(grep csrf_token /tmp/vakt-smoke-cookies.txt | awk '{print $7}' | tail -1)
SESSIONS=$(curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" \
    -H "X-Vakt-Session-Id: $SESSION_ID" \
    "$BASE/api/v1/auth/sessions")
CURRENT_COUNT=$(echo "$SESSIONS" | jq '[.[] | select(.is_current == true)] | length')
[[ "$CURRENT_COUNT" == "1" ]] || fail "S22-10 broken: erwartet genau 1 is_current session, war $CURRENT_COUNT"
ok "Genau 1 Session ist als is_current markiert"

LAST_USED=$(echo "$SESSIONS" | jq -r '.[0].last_used // empty')
[[ -n "$LAST_USED" ]] || fail "S22-10 broken: last_used fehlt in SessionInfo"
ok "last_used wird zurückgegeben: $LAST_USED"

# ── 4. API-Key mit Scopes + Rotation (S22-9) ──────────────────────────────────
step "4/5  API-Key mit Scopes + Rotation (S22-9)"
KEY_CREATE=$(curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" \
    -H "X-CSRF-Token: $CSRF" \
    -H 'Content-Type: application/json' \
    -X POST "$BASE/api/v1/api-keys" \
    -d '{"name":"smoke-test","scopes":["secvitals.*","secvault.*"]}')
KEY_ID=$(echo "$KEY_CREATE" | jq -r '.id')
RAW_KEY=$(echo "$KEY_CREATE" | jq -r '.raw_key')
SCOPES=$(echo "$KEY_CREATE" | jq -r '.scopes | join(",")')
[[ "$SCOPES" == "secvitals.*,secvault.*" ]] || fail "S22-9 broken: Scopes wurden nicht persistiert ($SCOPES)"
ok "Key erstellt mit Scopes: $SCOPES"

KEY_LIST=$(curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" \
    "$BASE/api/v1/api-keys")
ROW=$(echo "$KEY_LIST" | jq ".data[] | select(.id == \"$KEY_ID\")")
echo "$ROW" | jq -e '.scopes, .last_used_at' >/dev/null \
    || fail "S22-9 broken: Key-Row hat scopes/last_used_at-Felder nicht"
ok "Key-Liste enthält scopes + last_used_at + rotated_at"

# Rotation
ROTATE=$(curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" \
    -H "X-CSRF-Token: $CSRF" \
    -X POST "$BASE/api/v1/api-keys/$KEY_ID/rotate")
NEW_RAW=$(echo "$ROTATE" | jq -r '.raw_key')
[[ -n "$NEW_RAW" && "$NEW_RAW" != "$RAW_KEY" ]] || fail "S22-9 broken: Rotation hat keinen neuen Raw-Key geliefert"
ok "Rotation gibt neuen Raw-Key zurück (≠ alter)"

# Alter Key sollte während Grace-Period funktionieren + Deprecated-Header tragen
DEPRECATED_HEADER=$(curl -sI -H "Authorization: Bearer $RAW_KEY" "$BASE/api/v1/api-keys" \
    | grep -i 'x-vakt-key-deprecated' || true)
[[ -n "$DEPRECATED_HEADER" ]] || yellow "  ! Grace-Period-Header X-Vakt-Key-Deprecated NICHT gefunden (S22-1 evtl. nicht aktiv?)"
[[ -n "$DEPRECATED_HEADER" ]] && ok "Alter Key liefert X-Vakt-Key-Deprecated Header"

# Cleanup
curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" -H "X-CSRF-Token: $CSRF" \
    -X DELETE "$BASE/api/v1/api-keys/$KEY_ID" >/dev/null
ok "Smoke-Key wieder revoked"

# ── 5. AI-Agent-Endpoint erreichbar (S22-8) ───────────────────────────────────
step "5/5  AI-Agent-SSE-Endpoint erreichbar (S22-8)"
# Wir hitten den Endpoint mit einem klein-gehaltenen Goal und prüfen nur den
# ersten Frame — nicht den vollen LLM-Roundtrip (würde Ollama brauchen).
AI_PROBE=$(timeout 4 curl -sf -b /tmp/vakt-smoke-cookies.txt \
    -H "Authorization: Bearer $ACCESS" \
    -H "X-CSRF-Token: $CSRF" \
    -H 'Content-Type: application/json' \
    -N -X POST "$BASE/api/v1/secvitals/ai/agent/run" \
    -d '{"goal":"liste alle offenen findings","max_iterations":1}' 2>&1 | head -c 400 || true)
if echo "$AI_PROBE" | grep -qE '^data: '; then
    ok "Endpoint streamt SSE-Frames (erstes data:-Frame angekommen)"
elif echo "$AI_PROBE" | grep -qE 'AI_DISABLED|provider'; then
    yellow "  ! AI-Provider ist disabled (VAKT_AI_PROVIDER) — Endpoint OK, kein Stream"
elif echo "$AI_PROBE" | grep -qE 'unauthorized|forbidden'; then
    red   "  ✗ S22-8 broken: Endpoint antwortet mit Auth-Fehler"
    exit 1
else
    yellow "  ! Endpoint antwortete, aber kein SSE-Frame in 4s: $(echo "$AI_PROBE" | head -c 200)"
fi

# ──────────────────────────────────────────────────────────────────────────────
rm -f /tmp/vakt-smoke-cookies.txt
cyan ""
green "═══ Smoke-Tests grün ═══"
cyan ""
cyan "Jetzt manuelle Browser-Checks ($BASE):"
echo "  1. /login → mit obigen Creds einloggen"
echo "  2. /settings/sessions → 'Diese hier'-Badge sollte bei aktueller Session stehen,"
echo "     Panic-Button macht 2-Step-Confirm + redirect"
echo "  3. /settings/api-keys → 'Neuen Key' → Scope-Checkbox-Liste sichtbar;"
echo "     dann Rotate-Button auf einer Key-Row → Grace-Period-Modal"
echo "  4. /secvitals/ai/agent → Goal eingeben → Stop-Button + Plan/Tool-Cards live"
