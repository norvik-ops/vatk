# ADR-0038: `rotate-key` Coverage + HKDF-Binding, SAML-Legacy-Migration

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 57 (Marktreife-Welle 2)
**Related:** Audit-Befund F1, [[ADR-0010]] (AES-256-GCM für App-Secrets)

## Kontext

Das Auditos-Audit hat im `cmd/rotate-key`-Tool zwei strukturelle Defekte aufgedeckt, die *garantiert* zum Datenverlust führen sobald jemand das Tool produktiv einsetzt:

1. **HKDF-Mismatch.** `cmd/api/main.go:379-384` derived sechs purpose-spezifische Sub-Keys aus dem Master via HKDF-SHA256 (`vakt-vault-v1`, `vakt-totp-v1`, `vakt-alert-v1`, `vakt-github-v1`, `vakt-cloud-v1`, `vakt-webhook-v1`). Das alte `rotate-key/main.go:53,56` rief `sharedcrypto.Decrypt(oldMaster, …)` direkt mit dem rawMasterKey — also einem Schlüssel, mit dem die App **niemals** verschlüsselt hat. Resultat: `Decrypt` schlug systematisch fehl, jede Row wurde als „skipped (already rotated?)" geloggt, das Tool meldete „0 rotated, success" — und der Operator hatte den Eindruck, die Rotation wäre durch.
2. **Coverage 2 von 8.** Das Tool rotierte ausschließlich `so_secrets.encrypted_value` und `totp_secrets.secret`. Die fünf weiteren produktiv verschlüsselten Spalten — `notification_channels.{url_encrypted, hmac_secret_encrypted}`, `integrations_github.access_token`, `org_saml_configs.key_pem` — waren komplett unbehandelt. Selbst wenn das HKDF-Mismatch behoben gewesen wäre, hätte die App nach Master-Tausch nicht mehr die Alerting-Webhook-URLs, GitHub-Tokens oder SAML-SP-Schlüssel entschlüsseln können.

Bonus-Inkonsistenz beim Aufräumen entdeckt: `org_saml_configs.key_pem` (Migration 135, Sprint 21) wurde mit dem **rawMasterKey** verschlüsselt — also außerhalb der HKDF-Architektur. Vermutlich Übersehen beim Sprint-21-Review.

## Entscheidung

`cmd/rotate-key` wird komplett umgebaut zu einer Pipeline aus testbaren Stage-Funktionen, je eine pro verschlüsselter Spalte. Jede Stage:

1. HKDF-derived sowohl den OLD- als den NEW-Service-Key aus dem jeweiligen Master.
2. Iteriert über alle relevanten Rows (`SELECT id, ct FROM table`).
3. Versucht `Decrypt(oldKey, ct)`. Schlägt das fehl → `skipped` (Row ist bereits rotiert oder wurde unter einem fremden Key abgelegt). Erfolg → `Encrypt(newKey, plain)` → `UPDATE`.
4. Reportet `rotated` + `skipped` pro Stage.

Die Aufrufkette spiegelt die Production-HKDF-Architektur exakt wider, inklusive der zweistufigen Vault-Derivation:

```
master --HKDF("vakt-vault-v1")--> vaultKey
vaultKey --DeriveProjectKey(projectID)--> projectKey
AES-256-GCM(projectKey, plaintext) --> so_secrets.encrypted_value
```

### SAML-Legacy-Migration

`org_saml_configs.key_pem` bekommt **zwei** Decrypt-Pfade in der Stage:

1. Erstwahl: `HKDF("vakt-saml-v1")` (neuer Pfad, ab dieser ADR).
2. Fallback: rawMasterKey (Legacy, pre-ADR-0038).

Rows, die unter dem Fallback dekodieren, werden mit `migrated=true` markiert und unter dem HKDF-derived **new** key wieder verschlüsselt. Damit konvergieren alle SAML-Rows nach einem `rotate-key`-Lauf auf die HKDF-Architektur.

Parallel dazu wurde `saml_direct.go` so umgebaut, dass es Klartext-Reads ebenfalls beide Pfade probiert (HKDF zuerst, dann raw master). Damit funktionieren Installationen, die *noch nicht* rotate-key ausgeführt haben, weiter — aber jeder erfolgreiche raw-master-Read loggt eine Warnung mit Hinweis auf das Migrationstool.

### Coverage-Tabelle (Sprint 57)

| Spalte | Stage | HKDF-Purpose | Speicherformat |
|---|---|---|---|
| `so_secrets.encrypted_value` | `rotateVaultSecrets` | `vakt-vault-v1` + per-project | BYTEA |
| `totp_secrets.secret` (enabled rows) | `rotateTOTPSecrets` | `vakt-totp-v1` | BYTEA |
| `notification_channels.url_encrypted` | `rotateAlertChannelURLs` | `vakt-alert-v1` | BYTEA |
| `notification_channels.hmac_secret_encrypted` | `rotateAlertChannelHMACs` | `vakt-alert-v1` | BYTEA |
| `integrations_github.access_token` | `rotateGitHubAccessTokens` | `vakt-github-v1` | TEXT (hex) |
| `org_saml_configs.key_pem` | `rotateSAMLKeyPEMs` | `vakt-saml-v1` (+ legacy raw fallback) | BYTEA |

### Out-of-Scope (Sprint 58)

`cloud_integrations.config` ist ein JSONB mit verschlüsselten Sub-Werten. Die Rotation erfordert JSON-aware Re-Encryption pro Sub-Feld — eine eigene Stage in Sprint 58. Bis dahin warnt der Sprint-57-Bericht explizit, dass Cloud-Integrations vorab dekonfiguriert werden müssen, bevor der Master-Key getauscht wird.

`webhooks.secret` ist **Klartext** (Migration 098). Das ist ein eigener Audit-Punkt — die Spalte wurde laut Git-History für outgoing-webhook-HMAC-Signing eingeführt und nie verschlüsselt. Sprint 58 plant eine eigene Migration + Refactor; bis dahin keine Rotation nötig (weil nichts zu rotieren ist).

## Konsequenzen

**Positiv:**
- `rotate-key` ist nun korrekt: kein Datenverlust mehr im Erfolgsfall.
- Re-running ist sicher: bereits rotierte Rows werden geskippt, nicht erneut rotiert.
- Die Stage-Funktionen sind unit-testbar (siehe `rotate_test.go`), die End-to-End-Pipeline integration-testbar (siehe `internal/integration_test/rotate_key_real_test.go`).
- SAML-Legacy-Inkonsistenz wird im Zuge des Rotation-Laufs aufgelöst — Operatoren haben einen klaren Migrationspfad.

**Negativ / offen:**
- Cloud-Integrations sind noch nicht abgedeckt — Sprint 58.
- `webhooks.secret` Plaintext muss separat verschlüsselt + migriert werden.
- Die alte rotate-key Binary in Customer-Installationen ist unverändert kaputt — Release-Notes für v0.23.0+ müssen explizit auf den Tool-Fix hinweisen.

## Verifikation

- **Unit-Tests** (`cmd/rotate-key/rotate_test.go`, 7 Tests):
  - `TestReencryptRow_HappyPath` — Decrypt(old)/Encrypt(new) roundtrip
  - `TestReencryptRow_AlreadyRotated` — Skip-Signal bei Decrypt-Fehler
  - `TestReencryptRow_RotationIsReversibleWithOldKey` — Old key kann newCT nicht mehr lesen
  - `TestReencryptSAMLRow_HKDFPath` — HKDF-Quelle → HKDF-Ziel
  - `TestReencryptSAMLRow_LegacyMigration` — rawMaster-Quelle → HKDF-Ziel mit `migrated=true`
  - `TestReencryptSAMLRow_Skipped` — Fremde-Schlüssel-Quelle → errRowSkipped
  - `TestDecodeMasterKey_ChecksLength` — Eingabe-Validierung
- **Integration-Test** (`internal/integration_test/rotate_key_real_test.go`, build tag `integration`):
  - Bootet Postgres 16 via testcontainers, läuft alle Migrationen
  - Seedet je eine Row für jede der sechs Stages **unter den OLD-HKDF-Sub-Keys** + eine extra SAML-Row unter dem **rawMaster**
  - Baut die rotate-key-Binary on-demand, ruft sie als Subprozess mit ENV-Vars
  - Decryptet jede Row anschließend mit den NEW-HKDF-Sub-Keys → muss erfolgreich sein
  - Decryptet die Vault-Row zusätzlich mit dem OLD-Project-Key → muss fehlschlagen (Defense-in-Depth)
- Skip-Behavior: Test pinnt, dass beide SAML-Rows nach Rotation existieren (HKDF + legacy-migrated)

## Abgelehnte Alternativen

- **Tool entfernen statt fixen** — Customer-Need bleibt, ein kaputtes Tool ist schlimmer als kein Tool, aber kein Tool ist schlimmer als ein funktionierendes
- **Per-Cloud-Integration im selben Sprint mitrotieren** — schiebt den Sprint-Abschluss um Tage; lieber sauber begrenzt
- **SAML-Legacy ohne Migration belassen** — würde die HKDF-Inkonsistenz festschreiben; ein Compliance-Produkt darf da nicht ambivalent sein
- **In-place HKDF-Detection statt zweier Pfade** — schwierig zu testen, da AES-GCM keinen distinguishable Header hat. Try-and-fail mit zwei Pfaden ist explizit und nicht teurer
