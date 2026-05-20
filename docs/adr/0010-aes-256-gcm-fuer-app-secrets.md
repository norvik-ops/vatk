# ADR-0010: AES-256-GCM für Application-Level-Secrets

**Status:** Accepted  
**Datum:** 2026-02-15

## Kontext

Vakt speichert mehrere Klassen sensitiver Daten in der DB:

- Vakt Vault-Secrets (Kunden-API-Keys, DB-Passwörter)
- OIDC-Client-Secrets
- TOTP-Secrets (RFC 6238)
- Verschlüsselte E-Mail-Templates (Vakt Aware-Konfiguration)
- Cloud-Integration-Credentials (AWS-Access-Keys, Azure-Service-Principals)

Diese müssen so verschlüsselt sein, dass ein DB-Dump alleine (ohne den Master-Key) nutzlos ist.

## Entscheidung

**AES-256-GCM** für alle Application-Level-Verschlüsselungen. Master-Key aus `VAKT_SECRET_KEY` (32 Byte hex, environment variable).

- Nonce: 12 Byte, cryptographically random pro Verschlüsselung (`crypto/rand`)
- Output-Format: `nonce || ciphertext || tag` (base64-encoded für DB-Spalten)
- AES-Implementation: `crypto/aes` (Go-Stdlib, hardware-accelerated auf modernen CPUs)

`bcrypt` (cost 12) wird daneben für Passwort-Hashing und API-Key-Hashing verwendet (Einweg-Funktionen, nicht entschlüsselbar).

## Alternativen

- **ChaCha20-Poly1305** — gleichwertig sicher, etwas schneller ohne AES-NI. Verworfen weil: AES-NI ist auf jedem Server-CPU vorhanden, AES ist breiter audited.
- **Libsodium / NaCl** — verworfen weil: zusätzliche Cgo-Abhängigkeit, würde unser distroless-Image-Pattern brechen.
- **Plaintext mit Disk-Encryption** — verworfen: schützt nicht gegen DB-Dump-Leaks, schützt nicht gegen kompromittierte DB-Admins.

## Konsequenzen

### Positive

- Stdlib-only — keine zusätzlichen Dependencies, kein CGO.
- Master-Key nie in der DB — Cross-Boundary-Trennung (DB ≠ App-Secrets).
- Audit-fest: einzelne `shared/crypto`-Funktion mit Tests (Round-Trip, Nonce-Randomness, Tamper-Detection).

### Negative

- Key-Rotation ist eine bedeutsame Operation (alle verschlüsselten Spalten müssen entschlüsselt und neu verschlüsselt werden). Pattern ist dokumentiert in `docs/operations.md`.
- Verlust von `VAKT_SECRET_KEY` = irreversibler Datenverlust für die verschlüsselten Spalten. Backup-Strategie sieht vor: Key separat von DB-Dump sichern.

### Neutrale

- `VAKT_SECRET_KEY` dient gleichzeitig als Paseto-Token-Key (siehe ADR-0003). Diese Doppel-Verwendung ist sicher, weil beide Schemata semantisch unabhängige Inputs verarbeiten.

## Referenzen

- `backend/internal/shared/crypto/crypto.go`
- `backend/internal/shared/crypto/crypto_test.go`
- `docs/operations.md` — Key-Rotation-Runbook
- ADR-0003 (Paseto-Key-Verwendung)
