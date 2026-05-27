# ADR-0042: Row Level Security zurückgenommen — Migration 012-Theater entfernt

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 58 (Marktreife-Welle 3)
**Related:** Audit-Befund F6
**Supersedes (teilweise):** ein Teil von Migration 012 (`012_msp_multitenancy.up.sql`)

## Kontext

Migration 012 (Mai 2025, Sprint E16 — „MSP Multi-Tenancy") aktivierte `ROW LEVEL SECURITY` auf acht Tabellen (`vb_assets`, `vb_findings`, `ck_frameworks`, `ck_controls`, `ck_evidence`, `so_projects`, `so_secrets`, `pg_campaigns`) mit Policies der Form:

```sql
CREATE POLICY xxx_org ON xxx
    USING (org_id::text = current_setting('app.current_org_id', true));
```

Das Auditos-Audit hat zwei strukturelle Probleme aufgedeckt:

1. **`app.current_org_id` wird nie gesetzt.** Grep gegen das gesamte Backend (`set_config`, `app.current_org_id`): **0 Treffer**. Die Policies sind tatsächlich aktiv, aber wenn die Session-Variable NULL ist, würden sie alle Rows ausblenden — die App würde sofort kaputt sein.
2. **Kein `FORCE ROW LEVEL SECURITY`.** Postgres-Default ist: Table-Owner bypasst RLS. Der App-User ist Owner aller `vakt_*`-Tabellen (Standard golang-migrate setup). Damit ignoriert er die Policies sowieso.

Konsequenz: die Migration *aussieht* wie Defense-in-Depth, *liefert* aber nichts. Pen-Tester, die diese RLS sehen, würden den Schutz erwarten; bei näherer Inspektion wäre das ein Vertrauensbruch. Die anderen 22+ org-keyed Tabellen (`audit_log`, `ck_risks`, `hr_*`, `po_*`, `sr_*`, …) haben sowieso keine RLS — was die Inkonsistenz noch deutlicher macht.

## Entscheidungs-Optionen

| Option | Beschreibung | Beurteilung |
|---|---|---|
| **A** | App-Layer verdrahtet `SET LOCAL app.current_org_id = <orgID>` pro pgxpool-Acquisition, dazu `FORCE ROW LEVEL SECURITY` und Sweep auf alle 30+ org-keyed Tabellen | Echter Defense-in-Depth; großer Refactor (Pool-Acquire-Hook, Tx-Wrapping); Performance-Overhead messbar; Pull-Connection-Lifecycle muss überarbeitet werden |
| **B** | RLS entfernen, App-Layer ist alleinige Source of Truth — und das ist sie bisher auch | Ehrlich; sofort umsetzbar; verliert Defense-in-Depth gegen DB-Admin (im self-hosted Setup IST der Kunde der DB-Admin → fragwürdiger Schutz sowieso) |
| **C** | RLS selektiv für extra-sensitive Tabellen (z.B. `so_secrets`, `audit_log`) mit FORCE und App-Side-Hook | Hybrid: Pro-Risiko gewichtet, aber inkonsistent — Auditor fragt: warum diese Tabellen, nicht jene? |

## Entscheidung

**Option B**. Migration 150 macht `DROP POLICY` + `DISABLE ROW LEVEL SECURITY` auf den acht Migration-012-Tabellen.

Begründung:
- Vakt ist **self-hosted**: der Customer kontrolliert die DB; ein RLS-Bypass durch DB-Admin ist im Threat-Model nicht primär (Customer = Trusted-Layer).
- Die App-Layer-Isolation ist konsequent (`WHERE org_id = $1::uuid` in jeder Query), seit dem sqlc-Sweep (Sprints 25–27) gibt es kaum noch Raum für Bypass-Bugs.
- Theater zu entfernen senkt das Mismatch-Risiko bei externen Audits — ein Auditor, der erwartet, RLS sehe nach DB-Defense-in-Depth aus, wird das tatsächlich Vorgefundene **nicht** als Kompromiss verstehen, sondern als Compliance-Versprechen, das nicht eingelöst wird.

Falls in einer Zukunfts-Iteration echter Defense-in-Depth gewünscht ist (z.B. für eine separate Hosted-Edition mit Multi-Tenant-DB), kann eine neue Migration die RLS mit `FORCE` und ordentlichem App-Hook wiedereinführen. Das ist Pro/Enterprise-Tier-Diskussion, Sprint 60+.

## Konsequenzen

**Positiv:**
- Architektur ist ehrlich: App-Layer enforced Org-Isolation, keine Pseudo-DB-Schutzschicht
- Eine Tabelle weniger im Setup, kein Risiko, dass eine zukünftige App-Query unsichtbar gegen die Policy läuft
- Externe Pentester sehen einen konsistenten, dokumentierten Sicherheits-Model — kein Mismatch

**Negativ:**
- Wirklich-DB-Admin (z.B. Internal-Threat-Model in einem mit-MSP-betriebenen Deployment) hat keine zusätzliche Hürde mehr. Im self-hosted Modell ist das richtig; in einer hypothetischen Hosted-Edition wäre Option A nötig
- Eine Marketing-Claim „DB-Level Row-Level-Security" muss aus allen Pitches raus (war ohnehin nicht wahr)

## Verifikation

- `grep -rn 'ENABLE ROW LEVEL SECURITY' backend/db/migrations/` zeigt nur noch die geplante neue Migration (150 disable). Die ursprüngliche 012 bleibt im Migration-History-Trail (sonst würden bestehende DBs nicht weitermigrieren können).
- `make migrate` durchführbar gegen frische Postgres → Migrations 1..150 grün
- Smoke-Plan (Post-Deploy):
  ```sql
  SELECT tablename, rowsecurity FROM pg_tables
   WHERE schemaname='public' AND rowsecurity = true;
  -- Expected: 0 rows (or only tables enabled by a future migration)
  ```
- Bestehende Integration-Tests (`hr_evidence_real_test.go`, `audit_chain_real_test.go`, `auth_oidc_*`, `rotate_key_real_test.go`) müssen ohne Anpassung grün bleiben — sie haben RLS nie aktiviert und sollten von der Rücknahme nicht betroffen sein

## Folgewirkungen

- **`SECURITY.md` Update** — der Abschnitt "Defense-in-Depth" sollte explizit dokumentieren, dass die Org-Isolation App-Layer ist
- **`docs/security/threat-model.md`** (Sprint 58 oder 59 Kandidat) — sollte den expliziten Trust-Layer (Customer-DB = trusted) klarmachen
- **Marketing-Update**: jede Stelle, die „Row-Level Security on database level" verspricht, muss umformuliert oder gestrichen werden

## Abgelehnte Alternativen — Detail

- **Option A (App-Layer setzt session var + FORCE)**: zu aufwändig für Sprint 58, würde `pgxpool.Pool`-Wrapper brauchen, der bei jeder Acquisition ein `SET LOCAL` einfügt. Sprint 60+ wenn die Hosted-Edition-Diskussion echt wird
- **Option C (selektive Tabellen)**: ohne klare Threat-Model-Begründung willkürlich. Würde die Frage „warum diese 2 Tabellen, nicht alle 30" unbeantwortet lassen
