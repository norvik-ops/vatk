# `vakt-admin` — Operator-CLI

Das `vakt-admin`-Binary läuft direkt gegen die PostgreSQL-DB einer Vakt-Instanz und führt Operator-Aufgaben aus, die nicht über das Web-UI abgebildet sind (Password-Reset bei verlorenem MFA-Token, DB-Health-Checks, Bulk-User-Listings für Audits).

> **Wer das Binary nutzen kann, hat DB-Zugriff.** Wer DB-Zugriff hat, kann ohnehin alles ändern. Diese CLI macht häufige Admin-Aktionen idempotent und nachvollziehbar (jede Operation wird im `audit_log` mit `user_email = '<admin-cli>'` markiert).

## Installation

Das Binary ist Teil des Backend-Builds:

```bash
cd backend && go build -o vakt-admin ./cmd/admin
```

Im Docker-Image liegt es unter `/usr/local/bin/vakt-admin` und kann via `docker compose exec` aufgerufen werden:

```bash
docker compose exec api vakt-admin health-check
```

## Konfiguration

Die CLI nutzt dieselben Env-Vars wie der API-Server, mindestens:

- `VAKT_DB_URL` (Pflicht) — Postgres-Connection-String.
- `VAKT_SECRET_KEY` (Pflicht für Passwort-bezogene Commands) — der 32-Byte-Hex-Encryption-Key.

Alternativ kann die DB-URL per `--db-url`-Flag überschrieben werden.

## Commands

### `health-check`

Prüft DB-Connectivity + grundlegende Schema-Integrität.

```bash
vakt-admin health-check
```

Output bei Erfolg:
```
DB connection: ok (Postgres 16.4)
Schema integrity: ok (122 migrations applied)
```

Exit-Code: 0 = healthy, 1 = unhealthy.

### `list-orgs`

Listet alle Organisationen mit User-Count und Created-Timestamp.

```bash
vakt-admin list-orgs
```

Output:
```
SLUG               NAME              USERS  CREATED
acme               Acme GmbH         42     2026-01-15
demo-a3f2b1c9      Demo Visitor      2      2026-05-21
norvik             Norvik Ops Test   1      2024-11-01
```

### `list-users [org-slug]`

Listet User. Ohne Argument: alle User aller Orgs. Mit Argument: nur User der angegebenen Org.

```bash
vakt-admin list-users acme
```

Output:
```
EMAIL              ROLE      ACTIVE  LAST_LOGIN          MFA
admin@acme.local   admin     yes     2026-05-21 09:14    yes
ops@acme.local     analyst   yes     2026-05-20 16:22    no
```

### `reset-password <email> <new-password>`

Setzt das Passwort eines Users zurück. Der neue Plain-String wird bcrypt-gehashed (cost 12) und in `users.password_hash` geschrieben.

```bash
vakt-admin reset-password admin@acme.local 'temporary-12-chars-min'
```

Sicherheits-Anforderungen an das neue Passwort:
- Mindestens 10 Zeichen (Vakt-Default).
- Wird beim Login automatisch auf cost 12 re-hashed, falls ein älterer Cost vorgefunden wird (S13-6 Cost-Upgrade-on-Login).

Audit-Log-Eintrag: `password_reset_admin_cli`, Body mit `recovery_code`/`backup_code`-Redaction (S13-7).

> **Wichtig:** das Klartext-Passwort steht NICHT im Audit-Log. Im Shell-History des Operators steht es allerdings — bei sensiblen Operationen `history -d` aufrufen oder `set +o history` setzen.

## Geplante Commands (Roadmap)

Diese Commands sind in der Roadmap, noch nicht implementiert:

- `vakt-admin unlock <email>` — entsperrt einen automatisch durch Failed-Login-Limit gesperrten Account.
- `vakt-admin disable-mfa <email>` — erlaubt einem User, MFA neu einzurichten (bei verlorenem TOTP-Secret).
- `vakt-admin rotate-license-key` — generiert einen neuen Lizenz-Schlüssel und schickt ihn an die Customer-E-Mail.
- `vakt-admin dump-config` — gibt die effektive Config-Tabelle aus (mit redacted Secrets).
- `vakt-admin migrate up|down [N]` — wrapper um `cmd/migrate`, damit Operator nur ein Binary kennt.

## Operator-Hinweise

- **Idempotenz**: alle Commands sind so geschrieben, dass mehrfaches Ausführen keinen Schaden anrichtet (`reset-password` setzt das Passwort, ein zweites Mal Setzen tut dasselbe).
- **Audit-Trail**: jede Operation hinterlässt einen Eintrag im `audit_log`. Diese Einträge haben `user_id = NULL`, `user_email = '<admin-cli>'` und können im Admin-Audit-Log-Viewer gefiltert werden (`?user_email=<admin-cli>`).
- **Transaktionalität**: Operations, die mehrere Tabellen anfassen (z.B. `delete-org` in der Roadmap), laufen in einer einzelnen DB-Transaction. Bei Fehler in der Mitte: kompletter Rollback.
- **Kein Web-UI-Pendant**: Bewusste Entscheidung. Operationen, die DB-Zugriff voraussetzen, sollten nicht über Web-Auth ausgelöst werden — wer Web-Admin sein kann, muss nicht zwingend DB-Operator sein.

## Referenzen

- `backend/cmd/admin/main.go` — Source-Code
- `docs/operations.md` — Operations-Doku (Backup, Restore, Monitoring)
- `docs/concepts/rbac-model.md` — wer darf was über das Web-UI
- `docs/GLOSSARY.md` — Begriffe (System-User, Audit-Log)
