# ADR-0044: Auth-Lockout-Checks fail closed bei Redis-Outage

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 58 (Marktreife-Welle 3)
**Related:** Audit-Befund P1-6

## Kontext

`internal/auth/service.go` enthält zwei Lockout-Checks pro Login-Versuch:

- `checkAccountLocked(email)` — pro-Account-Brute-Force-Schutz
- `checkIPLocked(ip)` — pro-IP-Brute-Force-Schutz

Beide schlugen bisher bei Redis-Fehler auf „fail-open" zurück (`return false, nil`) — ein Login-Versuch durfte durch, weil der Counter nicht gelesen werden konnte. Comment im alten Code: „fail open to avoid blocking legitimate logins."

Konsequenz aus dem Audit: in einem Redis-Failover (kurzer Outage ~30-60s) öffnet sich ein **Brute-Force-Window**. Während Redis nicht erreichbar ist, kann ein Angreifer **beliebig viele** Login-Versuche pro Account und IP machen, ohne ausgesperrt zu werden. Bei einem manchmal-flapy-Redis ist das ein wiederkehrender, kaum erkennbarer Verlust an Brute-Force-Protection.

## Entscheidung

Standard ist **fail closed**: wenn Redis nicht erreichbar ist, lehnen wir Login-Anfragen mit HTTP 503 (`AUTH_LOCKOUT_UNAVAILABLE`) ab, bis Redis wieder verfügbar ist. Die Service-Helper geben `(true, ErrLockoutCheckUnavailable)` zurück; der Login-Handler kennt das Sentinel und mapt es auf 503.

**Opt-In zum Legacy-Verhalten** via `VAKT_AUTH_FAIL_OPEN_ON_REDIS_OUTAGE=true`:

- Setzt im API-Bootstrap (`cmd/api/main.go`) `authSvc.WithFailOpenOnRedisOutage(true)`
- Logged ein lautes Warn-Event („audit-relevant choice")
- Verfügbar für Customer, die Login-Availability über Brute-Force-Protection priorisieren (z.B. Demo-Instanzen, Single-User-Setups)

Die Token-Revoke-Logik (`RevokeToken`) hat **bereits** einen PostgreSQL-Fallback (`token_deny_list_fallback`-Tabelle), der fail-closed verhält — Sprint 21. Daher nicht angefasst.

## Konsequenzen

**Positiv:**
- Audit-konform: kein Brute-Force-Window mehr bei Redis-Outage
- Operator hat einen klaren Opt-Out für Availability-First-Deployments
- 503 + machine-readable `AUTH_LOCKOUT_UNAVAILABLE`-Code lässt Frontend einen klaren Retry-Hinweis zeigen

**Negativ:**
- Ein flaky Redis bei einem Customer (z.B. unkonfigurierte Memory-Limits) verhindert nun ALLE Logins, nicht nur Brute-Force-Attempts. Dokumentation muss Operatoren auf das ENV-Flag hinweisen
- Login-Latenz im Fehlerpfad steigt: 2s-Timeout pro Lockout-Check + 2s pro IP-Check = bis zu 4s, statt sofort durchlassen. Vertretbar, weil normal-Path unverändert ist

## Verifikation

- **Unit-Tests** (`internal/auth/lockout_failclosed_test.go`, 4 Tests):
  - Default: `checkAccountLocked` + `checkIPLocked` returnen `(true, ErrLockoutCheckUnavailable)` bei Redis-Outage
  - Mit `WithFailOpenOnRedisOutage(true)`: returnen `(false, nil)` (Legacy-Verhalten)
  - Tests fingieren Redis-Outage via Dial gegen Port 1 (unbindable) — kein Container nötig
- **Smoke** (post-deploy):
  - `docker compose stop redis` — Login muss 503 `AUTH_LOCKOUT_UNAVAILABLE` zurückgeben
  - `VAKT_AUTH_FAIL_OPEN_ON_REDIS_OUTAGE=true docker compose up` — Login muss durchgehen (legacy)
  - Beide Pfade in `docs/dev/redis-outage-runbook.md` (Sprint 59 Doku) festgehalten

## Abgelehnte Alternativen

- **Per-Account-Fallback in PostgreSQL** (analog zu Token-Deny-List) — Counter-Inkrement pro Login wäre teurer Postgres-Lock + INSERT, würde Login-Latenz im Normal-Path durchgehend ~5ms erhöhen. Nicht wert für eine seltene Failure-Mode
- **Soft-Fail mit Banner** (HTTP 200 + Warning-Header) — Frontend würde das ignorieren, Brute-Force-Window bliebe offen
- **Stricter Default mit DOS-Risiko**: kein Opt-Out — Operator-Friction zu hoch bei normalen Redis-Restarts
