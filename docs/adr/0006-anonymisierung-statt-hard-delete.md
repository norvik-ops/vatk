# ADR-0006: Anonymisierung statt Hard-Delete bei DSGVO Art. 17

**Status:** Accepted  
**Datum:** 2026-05-20

## Kontext

DSGVO Art. 17 verlangt das Recht auf Löschung. Ein naheliegender Implementierungsweg ist `DELETE FROM users WHERE id = ...` mit `ON DELETE CASCADE`-Folgen.

Vakt ist gleichzeitig ein Compliance-Produkt. ISO 27001 A.5.28 und BSI ORP.2 verlangen einen revisionssicheren Audit-Trail, der **jede** sicherheitsrelevante Aktion auf einen Akteur zurückführen kann. Eine harte Löschung des Users würde Fremdschlüssel im `audit_log` zerstören und vergangene Aktionen unzuordenbar machen.

DSGVO und ISO 27001 sind hier scheinbar im Konflikt — sind sie aber nicht: DSGVO Art. 17 (3) erlaubt explizit die Aufbewahrung „für die Begründung, Ausübung oder Verteidigung von Rechtsansprüchen", und Art. 5 (1)(e) ist erfüllt, wenn die Person nicht mehr identifizierbar ist.

## Entscheidung

**Anonymisierung in-place** statt Hard-Delete bei Account-Löschung. Konkret:

- `email` → `deleted-<uuid>@vakt.local`
- `display_name` → `[gelöscht]`
- `avatar_url`, `oidc_subject`, `oidc_provider` → NULL
- `password_hash` → NULL (Login dauerhaft unmöglich)
- `is_active` → false
- Alle Sessions + API-Keys werden widerrufen

Der `user_id`-UUID bleibt **stabil**, damit Audit-Log-Joins funktionieren. Aus dem Eintrag lässt sich nichts mehr über die Person ableiten — nur dass eine User-ID X eine Aktion Y vorgenommen hat.

## Alternativen

- **Hard-Delete mit CASCADE** — verworfen: zerstört Audit-Trail-Integrität, verletzt ISO 27001 A.5.28.
- **Soft-Delete mit `deleted_at` flag, PII bleibt** — verworfen: PII bleibt in der DB, also keine echte „Löschung" im Sinne der DSGVO.
- **Hard-Delete + Audit-Log-Anonymisierung in einem Schritt** — verworfen: zu komplex, Schema-weite Migration nötig, jeder neue Audit-Konsument muss berücksichtigt werden.

## Konsequenzen

### Positive

- DSGVO Art. 17 und ISO 27001 A.5.28 gleichzeitig erfüllt.
- Robuste Implementierung — kein Risiko verwaister FK-Referenzen.
- Re-Auth-Passwort-Check (oder OIDC-Login als Identitätsnachweis) schließt böswillige Löschung aus.

### Negative

- Audit-Logs enthalten weiterhin UUIDs ehemaliger Nutzer (rein technische ID — DSGVO-Aufsichtsbehörden akzeptieren das).
- Die ehemals registrierte E-Mail kann frühestens nach DB-Cleanup wiederverwendet werden (UNIQUE-Constraint auf der anonymisierten Adresse — die ist aber eindeutig pro UUID, also kein Problem).

### Neutrale

- Wenn ein Kunde hartes Hard-Delete will (z.B. wegen behördlicher Anordnung), kann er das mit `DELETE FROM users WHERE id = ...` direkt auf der DB tun. Vakt's eigene UI macht das nicht.
- Der `guardLastAdmin`-Check verhindert das Orphaning einer Organisation.

## Referenzen

- `backend/internal/shared/account/account.go` — `DeleteUserAccount`
- `backend/internal/shared/account/account_test.go`
- DSGVO Art. 17 (3) und Art. 5 (1)(e)
- ISO 27001:2022 A.5.28 Logging & monitoring
