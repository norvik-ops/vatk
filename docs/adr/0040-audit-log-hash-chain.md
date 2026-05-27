# ADR-0040: Audit-Log Tamper-Evidence via Per-Org Hash-Chain

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 57 (Marktreife-Welle 2)
**Related:** Audit-Befund F8, [[ADR-0010]] (AES-GCM für App-Secrets)

## Kontext

Migration 064 hat das `audit_log` als simple Insert-only-Tabelle angelegt — keine Hash-Spalte, kein Trigger, keine Signatur. Der `writer.go` schrieb mit plain `INSERT`, Fehler wurden geswallowed. Konsequenz: jeder mit `UPDATE`/`DELETE`-Rechten auf `audit_log` (App-User selbst, DB-Admin) konnte historische Audit-Einträge nachträglich modifizieren, löschen oder einfügen — ohne dass das Compliance-Modul es erkennt.

Disqualifiziert Vakt für:

- **ISO 27001 A.12.4.3** „Protection of log information" (verlangt explizit Tamper-Resistenz)
- **NIS2 Art. 21.2(g)** und **DORA Art. 11** — beide erwarten forensisch verwertbare Logs
- **BSI IT-Grundschutz OPS.1.1.5.A8**

Für ein Compliance-Produkt, dessen Hauptverkaufsargument NIS2- und ISO-27001-Bereitschaft ist, ist das ein **Pre-Sales-Disqualifier** (siehe Audit-Sub-Report).

## Entscheidung

Per-Org SHA-256-Hash-Chain auf `audit_log`. Jede neue Row trägt:

| Spalte | Typ | Inhalt |
|---|---|---|
| `prev_hash` | `BYTEA` | `entry_hash` der letzten Row derselben Org (NULL für die erste chained Row) |
| `entry_hash` | `BYTEA` | `SHA-256(prev_hash || canonical(this_row))` |

`canonical()` (`internal/shared/audit/chain.go:canonicalString`) baut eine deterministische Pre-Image-Darstellung aus allen relevanten Feldern, getrennt durch `|` (mit Backslash-Escape), in fixer Reihenfolge. Map-Felder werden mit Schlüssel-Sortierung serialisiert, damit Go's randomisierte Map-Iteration keine Hash-Drift erzeugt.

### Writer-Flow

1. `BEGIN`.
2. `SELECT entry_hash FROM audit_log WHERE org_id=$1 AND entry_hash IS NOT NULL ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE` — sperrt die Chain-Spitze für diese Org gegen konkurrierende Schreiber.
3. `prev_hash` = das gelesene `entry_hash` (oder `nil` für die erste Row).
4. `entry_hash` = `SHA-256(prev_hash || canonical(this_row))`.
5. `INSERT` mit beiden Hashes.
6. `COMMIT`.

Der `FOR UPDATE`-Lock serialisiert Inserts pro Org, ist aber unkritisch in der Praxis: Audit-Inserts sind via fire-and-forget-Goroutine entkoppelt und Audit-Volume ist O(seconds) zwischen Events pro Org. Cross-Org-Inserts bleiben vollparallel.

### Verifier (`cmd/audit-verify`)

- Iteriert alle Orgs mit `entry_hash IS NOT NULL`
- Pro Org: Replay in `(created_at, id) ASC`, rechnet `entry_hash` neu, vergleicht
- Returnt UUID der ersten verdächtigen Row + `exit 2`
- Bei Bedarf scopeable via `VAKT_AUDIT_VERIFY_ORG=<uuid>`

### Pre-Migration-Daten

Rows die vor Migration 149 geschrieben wurden, haben NULL in beiden Hash-Spalten. Der Verifier überspringt diese Prefix-Schicht stillschweigend — sie sind nicht-tamper-evident und werden ehrlich so behandelt.

### Was NICHT in Sprint 57

- **Externe Time-Stamping Authority** (RFC 3161, Sectigo etc.) — sinnvoll für Tier-1-Compliance, aber externes Abhängigkeitsverhältnis bricht das No-Phone-Home-Prinzip. Sprint-58-Diskussion: optional opt-in für Pro-Tier-Kunden, wenn der Customer eigene TSA betreibt.
- **WORM-Mount für Audit-Log-Tabelle** — Postgres-side `BEFORE UPDATE/DELETE` Trigger mit `RAISE EXCEPTION` — Sprint-58-Kandidat zusammen mit RLS-Entscheidung. Hash-Chain ist defense-in-depth-Schicht 1; Trigger wäre Schicht 2.
- **Backfill alter Rows** — würde retroaktive Integrität suggerieren, die nicht existiert. Falsch-Compliance ist schlimmer als ehrlich-noch-nicht.

## Konsequenzen

**Positiv:**
- Audit-Log ist **forensisch verwertbar** ab Migration 149
- Manipulation einer einzelnen Row bricht die Chain ab dieser Row — der Verifier zeigt UUID der Tamper-Stelle
- Löschen einer Row bricht die Chain bei der **nächsten** Row (sie verweist auf einen `prev_hash`, dessen Quelle nicht mehr existiert)
- Insert einer fremden Row bricht die Chain ab der eingefügten Row
- ISO/NIS2/DORA-Tauglichkeit auf Audit-Trail-Ebene erreicht

**Negativ:**
- Audit-Writes brauchen zusätzlich `SELECT FOR UPDATE` + `BEGIN/COMMIT` — gemessen vernachlässigbar, aber bei extremem Audit-Volume (>>100/sec/org) zu beobachten
- Eine *fortgesetzte* Tampering-Attacke kann theoretisch die Chain konsistent neu schreiben (Forge entry_hash für jede gefälschte Row). Defense gegen das gehört in eine externe TSA — Sprint-58-Kandidat
- Zwei sehr nahe `created_at`-Werte können in Tests die Reihenfolge zwischen Insert und Verify abweichend bestimmen. Wir sortieren defensive nach `(created_at, id)` ASC

## Verifikation

- **Unit-Tests** in `internal/shared/audit/chain_test.go` (5 Tests, 8 Sub-Cases):
  - Deterministisch bei gleicher Eingabe
  - `prev_hash`-Variation ändert `entry_hash`
  - Jede Feld-Mutation bricht die Equality (8 Mutationen geprüft)
  - Map-Order-Stabilität (Bug-Regression: naive `json.Marshal` der Details wäre nicht-deterministisch)
  - Pipe-Separator-Injection bricht nicht den Verifier
- **Integration-Tests** in `internal/integration_test/audit_chain_real_test.go` (3 Tests):
  - `TestAuditChain_VerifyAfterInserts` — 3 sukzessive Writes verifizieren grün
  - `TestAuditChain_DetectsTamperedRow` — `UPDATE audit_log SET action='EVIL'` wird durch Verifier lokalisiert
  - `TestAuditChain_DetectsDeletedRow` — `DELETE` lokalisiert auf die Folge-Row
- **cmd/audit-verify** Tool gebaut, Exit-Codes definiert (0/1/2)
- **CI-Plan**: Ein wöchentlicher Cron-Job in der Demo-Instanz, der `audit-verify` ausführt und Slack benachrichtigt — Sprint 58.

## Operative Empfehlungen

- Customer-Installationen sollten `audit-verify` einmal pro Tag im Cron-Job laufen lassen
- Datenbank-Backups sollten den `audit_log`-Inhalt vollständig erhalten (Range-Backup ist Sprint-58-Plan)
- Bei Verifier-Exit-Code 2: **Sofort** DB-Snapshot ziehen, alle App-Sessions invalidieren, Master-Key rotieren (`cmd/rotate-key`), Incident-Response gemäß `docs/security/incident-response.md`

## Abgelehnte Alternativen

- **Globale Hash-Chain (eine Chain für alle Orgs)** — eine schreibende Org würde alle anderen Schreiber pro-Insert blockieren (`SELECT FOR UPDATE`-Hotspot). Pro-Org-Chain skaliert linear in Orgs
- **Merkle-Tree pro Org** — komplexer, schneller bei Bulk-Verify, aber Pro-Insert-Latenz schlechter (Logarithmische Anzahl Hashes)
- **HMAC statt SHA-256** — würde geheimen Key zum Hashen brauchen, der bei App-Compromise auch leakt; Defense-in-Depth ist eher externe TSA
- **Trigger statt App-Side-Chain** — würde DB-User Privilege erhöhen müssen, vermischt App-Logik mit DB-Logik
