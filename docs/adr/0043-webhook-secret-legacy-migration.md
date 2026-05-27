# ADR-0043: `webhooks.secret` Legacy-Plaintext-Migration

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 58 (Marktreife-Welle 3)
**Related:** Audit-Befund P1-2 Subnote, [[ADR-0038]] (rotate-key Coverage)

## Kontext

Migration 098 (`098_webhooks.up.sql`) hat `webhooks.secret` als `TEXT`-Spalte angelegt. Eine spätere Iteration (vor Sprint 56) hat `internal/shared/platform/webhooks/service.go` so umgebaut, dass beim Insert/Update das Klartext-Secret mit der HKDF-derived `vakt-webhook-v1`-Sub-Key AES-256-GCM-verschlüsselt und mit Präfix `enc:v1:` plus base64-URL-Encoding gespeichert wird (`encryptSecret`/`decryptSecret` in `service.go:95-124`).

Audit P1-2 listete „`webhooks.secret` Plaintext-TEXT-Spalte" als offenes Problem. Verifiziert gegen den Code (Sprint 57) lag die Lage tatsächlich so:

- **Neue Rows** (Insert/Update nach Sprint X) werden korrekt `enc:v1:…`-encrypted.
- **Legacy-Rows** (Insert pre-Sprint X) sind weiterhin Klartext. `decryptSecret` erkennt sie über den fehlenden Prefix und gibt sie unverändert zurück — funktional korrekt, aber Compliance-relevant.

Konsequenz: in jedem Customer-Setup, das seit v0.6.x eine Webhook angelegt hat, liegen ggf. weiterhin Klartext-Secrets in der DB.

## Entscheidung

Drei Schritte schließen die Lücke:

1. **`MigrateLegacyPlaintextSecrets(ctx) (int, error)`** in `WebhookService` walkt alle `webhooks.secret`-Rows ohne `enc:v1:`-Prefix, encryptet sie mit dem konfigurierten Master-Key und updated die Row. Idempotent (eine zweite Ausführung findet nichts mehr). Fehler pro Row werden ge-loggt aber nicht eskaliert — eine einzelne korrupte Row darf nicht den ganzen Boot blockieren.

2. **API-Boot-Hook** in `cmd/api/main.go` (direkt nach `NewWebhookService`) ruft die Migration **idempotent** bei jedem Start auf. Sobald alle Customer einen v0.25.0+-Boot hatten, sind alle Legacy-Rows umgeschrieben.

3. **rotate-key Stage** `rotateWebhookSecrets` rotiert nur `enc:v1:`-Rows — Legacy-Plaintext wird *nicht* mit-rotiert, sondern dem Boot-Hook überlassen (Trennung: rotate-key kümmert sich um Key-Wechsel, Boot-Hook um Format-Migration). Wenn ein Customer rotate-key vor seinem ersten v0.25.0-Boot ausführt, bleibt das Plaintext-Row unverändert, und der erste Folgeboot migriert es.

## Konsequenzen

**Positiv:**
- Webhook-Secrets sind ab einer Boot-Wave alle verschlüsselt
- Boot-Hook ist idempotent + selbstheilend (logging Errors statt fatal)
- rotate-key bleibt fokussiert auf seine eine Aufgabe (Key-Rotation, nicht Format-Migration)

**Negativ / akzeptiert:**
- Ein Customer, der noch nie einen Webhook angelegt hat, lädt eine triviale Migration mit (5 Zeilen Code, nano-sekunden-Aufwand) — vernachlässigbar
- Ein Operator, der unmittelbar nach einem Migration-Push den Master-Key tauscht, lässt einen Legacy-Plaintext-Row unbearbeitet; der nächste API-Boot fixt es

## Verifikation

- **Unit-Tests** (`internal/shared/platform/webhooks/secret_format_test.go`, 5 Tests):
  - Roundtrip encrypt/decrypt
  - Legacy-Plaintext-Erkennung (no prefix → returned as-is)
  - Dev-Mode (no master key → plaintext passthrough)
  - Wrong-Key-Failure (Defense-in-Depth)
  - Plaintext darf nicht in Stored Value erscheinen
- **rotate-key Integration-Test** (`rotate_key_real_test.go`) enthält einen `webhooks.secret`-Roundtrip mit echtem Postgres
- **Boot-Hook**: bei lokalem `make api-local` mit präparierter DB (eine Plaintext-Row) wird ein Log-Eintrag emittiert (`migrated=1`); zweiter Start findet 0

## Abgelehnte Alternativen

- **Migration 151 macht die Encryption** — DB-Migrations haben keinen Master-Key (architektonisch korrekt: Migrations sollen kryptografische Operationen vermeiden)
- **`enc:v2:` als neues Format** — überflüssig solange `enc:v1:` (AES-256-GCM mit HKDF-derived Key) state-of-the-art ist
- **Plaintext-Path entfernen** — würde Customer-DBs vor erstem Boot brechen
