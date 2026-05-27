# ADR-0045: `audit_log` Range-Partitioning auf `created_at`

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 58 (Marktreife-Welle 3)
**Related:** Audit-Befund P1-2, [[ADR-0040]] (Audit-Log Hash-Chain)

## Kontext

Migration 064 hat `audit_log` als monolithische Tabelle mit `UUID PRIMARY KEY` und vier sekundären Indizes angelegt. Über die Lebenszeit eines Customers wächst sie unbeschränkt — jeder Insert kostet 4 B-Tree-Insertions, jede Compliance-Frontend-Query macht einen Index-Scan über die gesamte Tabelle.

Konkrete Bedrohungen (Audit):

- **Insert-Latency-Drift**: bei 100k Audit-Events/Tag (typischer Mid-Size-NIS2-Customer) erreichen die Indizes nach 1-2 Jahren mehrere hundert MB; Insert-Cost steigt unmerklich aber kontinuierlich
- **Retention/Export-Komplexität**: ein „export + delete 2024 events" ist heute ein langer `DELETE WHERE created_at < ...` der die Tabelle minutenlang lockt
- **Vacuum-Druck**: full-table vacuum auf einer >10 GB Tabelle ist täglich Stress, partitions vacuumen unabhängig

## Entscheidung

`audit_log` wird mittels Migration 151 in eine **PARTITION BY RANGE (created_at)**-Tabelle umgebaut. Eine Partition pro Jahr (`audit_log_2025`, `_2026`, `_2027`, `_2028`) plus eine DEFAULT-Partition für alles Außerhalb.

### Schema-Änderungen

- **PRIMARY KEY** wechselt von `(id)` auf `(id, created_at)`. Postgres verlangt, dass die Partition-Key-Spalte in jeder UNIQUE-Constraint enthalten ist.
- Sekundäre Indizes (`audit_log_org_idx`, `_user_idx`, `_resource_idx`, `_org_chain_idx`, `idx_audit_log_org_time`) werden in der Migration neu angelegt — partition-aware (lokale Indizes pro Partition).
- Die `audit_logs`-VIEW (Migration 085) wird neu angelegt, weil `DROP TABLE ... CASCADE` sie verliert.

### Warum Yearly, nicht Monthly

- Audit-Volumen pro Self-Hosted-Customer ist niedrig (typisch <10k Rows/Monat). Yearly-Partitions à 100k-500k Rows sind weder zu klein noch zu groß.
- Monatliche Partitions würden ~12× mehr Partitions/Tabellen-Objekte erzeugen, dafür kleinere Vacuum-Units. Sprint 60+ kann mit `ALTER TABLE … DETACH/ATTACH` jährlich → monatlich splitten, ohne Downtime.

### Migrationspfad

`151_audit_log_partitioned.up.sql`:

1. `CREATE TABLE audit_log_new (...) PARTITION BY RANGE (created_at);`
2. CREATE 4 yearly + 1 DEFAULT partition
3. `INSERT INTO audit_log_new SELECT * FROM audit_log;` (partition routing automatisch)
4. `DROP TABLE audit_log CASCADE;` (entfernt auch die VIEW)
5. `RENAME audit_log_new TO audit_log;`
6. Indizes neu erstellen
7. `audit_logs`-VIEW neu erstellen (back-compat aus Migration 085)

`151_audit_log_partitioned.down.sql` macht die umgekehrte Operation und ist data-preserving.

### Hash-Chain-Kompatibilität

Die per-Org Hash-Chain (ADR-0040) ist mit Partitioning kompatibel:

- Writer (`audit.Write`) liest die Chain-Spitze via `SELECT ... ORDER BY created_at DESC LIMIT 1 FOR UPDATE`. Der Index `audit_log_org_chain_idx` ist nun lokal pro Partition, aber die Query macht einen Partition-Pruning auf alle Partitions, die für die Org Rows haben — meist nur die aktuelle.
- Verifier (`audit.VerifyOrgChain`) iteriert `ORDER BY created_at ASC, id ASC` — Partition-Pruning übernimmt der Postgres-Planner automatisch.
- Integration-Test `audit_partitioned_real_test.go::TestAuditLog_VerifierStillWorksAfterPartitioning` pinnt diese Kompatibilität.

## Konsequenzen

**Positiv:**
- Audit-Log skaliert linear in Customer-Lebenszeit ohne Re-Indexing-Pause
- Yearly-Detach öffnet einen sauberen Retention-Workflow (Sprint 60 Roadmap-Item)
- Lokale Vacuums pro Partition statt globaler Bigtable-Vacuum
- ISO 27001 + DORA Audit-Trail-Anforderungen ohne Operations-Risiko bei großen Customers

**Negativ:**
- Composite PK `(id, created_at)` macht ein paar Code-Stellen marginal expliziter, wenn man eine Row eindeutig löschen will (`WHERE id = $1 AND created_at = $2`) — aber keiner unserer aktuellen Codepfade tut das
- Eine Migration mit `INSERT INTO … SELECT *` kann bei sehr großen `audit_log`-Tabellen lang dauern. Demo-Instanzen + Pen-Test-Lokal (laut [[project_install_base]]) sind klein → unproblematisch. Customers mit großen Audit-Logs sollten die Migration während eines Maintenance-Fensters fahren
- Default-Partition ist ein „Catch-All" — bei Rows mit `created_at` > 2028 landet alles dort. Sprint 60 muss eine automatische Yearly-Roll-Forward implementieren

## Verifikation

- **Migration**: golang-migrate führt 151 als atomisches Tx durch (BEGIN-COMMIT)
- **Integration-Tests** (`internal/integration_test/audit_partitioned_real_test.go`):
  - `TestAuditLog_PartitionedAfterMigration` — `pg_class.relkind = 'p'` für `audit_log`, alle 5 Yearly-Children + Default existieren
  - `TestAuditLog_PartitionRoutingPicksCorrectChild` — Rows mit 2026-Date landen in `audit_log_2026`, 2027 in `audit_log_2027`
  - `TestAuditLog_VerifierStillWorksAfterPartitioning` — Hash-Chain Round-Trip mit `audit.Write` + `audit.VerifyOrgChain` über partitionierte Tabelle
  - `TestAuditLog_LegacyViewStillCompatible` — `SELECT * FROM audit_logs` (Backward-Compat-View) funktioniert
- **Backend-Suite**: `go test ./...` grün auf 32 Paketen nach Migration

## Abgelehnte Alternativen

- **pg_partman** als Auto-Maintainer — externe Extension, würde Customer-Deployment komplexer machen (extension install pro Postgres-Instanz). Sprint 60 kann das nachholen, falls Maintenance-Burden steigt
- **Monthly statt Yearly** — zu viele Partitions für typische Self-Hosted-Volumina; Sprint 60-Kandidat wenn Daten zeigen, dass Yearly-Partitions zu groß werden
- **Hash-Partitioning auf org_id** — würde per-org-Queries beschleunigen, aber Range-Queries auf created_at verlangsamen; und die Cross-Org-Aggregation (Admin-Dashboard) wäre teurer
- **Bleibe bei monolithischer Tabelle, fix nur Vacuum** — verschiebt das Problem, löst es nicht
