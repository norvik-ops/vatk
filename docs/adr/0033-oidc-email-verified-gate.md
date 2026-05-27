# ADR-0033: OIDC `email_verified`-Gate beim Linking auf bestehende Lokal-Accounts

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 56 (Marktreife-Welle 1)
**Related:** Audit-Befund F4 (`vakt-app-analyse_9/security_report.md`), Verify-Bericht `docs/reviews/2026-05-27-auditos-9-verify.md`

## Kontext

`internal/auth/oidc.go:provisionOIDCUser` koppelt OIDC-Subjects an bestehende Lokal-Accounts allein über die Email-Adresse: wenn keine User-Row mit dem OIDC-Sub existiert, wird per Email gesucht und auf Treffer ohne weitere Prüfung verlinkt (`UPDATE users SET oidc_subject = …`).

Ein OIDC-IdP, der unverifizierte Email-Eingaben akzeptiert (z.B. ein selbst-gehostetes Casdoor mit deaktiviertem Email-Confirm oder ein Public-Provider in einem Misconfiguration-Setup), erlaubt damit einen **trivialen Account-Takeover**: Angreifer registriert beim IdP einen Account mit `email = victim@kunde.de` ohne Email-Verifikation, klickt auf „Mit Google anmelden" in Vakt, IdP liefert Profile mit dieser Email aus, Vakt linkt den fremden Subject an den bestehenden lokalen Admin-Account des Opfers.

Der Pfad existiert in drei Aufruf-Stellen:

1. `OIDCLogin` (Z 144) — generisch via Casdoor
2. `SAMLLogin` (Z 209) — Casdoor-proxied SAML
3. `provisionSAMLUser` (Z 227) — direct SAML, mit XML-DSig-Verifikation in `saml_direct.go`

Bei SAML 2 & 3 ist die Email IdP-signiert (Signature-Check in `saml_direct.go:329`) und damit verifiziert. Bei 1 ist die Information aus dem Casdoor-Profile zu nehmen, das das Feld `emailVerified` als Bool exponiert.

## Entscheidung

Wir führen einen expliziten `emailVerified bool`-Parameter in `provisionOIDCUser` ein. Linking auf einen **existierenden lokalen Account mit gleicher Email** ist nur erlaubt, wenn `emailVerified == true`. Andernfalls wird `ErrEmailNotVerified` zurückgegeben und der Login-Pfad bricht ab.

Wichtig: Wenn **kein** existierender lokaler User die Email hat, ist der Pfad weiterhin erlaubt, auch ohne `emailVerified`. Es entsteht ein neuer Account — kein Takeover möglich, weil nichts zum Übernehmen da ist. So bleibt die Onboarding-Erfahrung für Erst-Login per Cloud-IdP unverändert.

### Quellen für `emailVerified`

| Pfad | Quelle |
|------|--------|
| `OIDCLogin` | `casdoorUserProfile.EmailVerified` (Feld `emailVerified` aus `/api/get-account`) |
| `SAMLLogin` | `true` — Casdoor verifiziert die Assertion vor Antwort |
| `provisionSAMLUser` | `true` — `saml_direct.go:329` `sp.ParseResponse` validiert XML-DSig |

Wenn ein IdP das `emailVerified`-Feld nicht setzt, fällt es per Go-Default auf `false` zurück — wir gehen also defensiv mit fehlender Information um.

## Konsequenzen

**Positiv:**
- Account-Takeover über unverifizierten IdP-Email-Claim ist nicht mehr möglich.
- Kein zusätzlicher Konfigurationsaufwand für Admins.
- Erst-Login per Cloud-IdP funktioniert weiter — nur das Linking auf vorhandene Accounts braucht Verifikation.

**Negativ / Risiken:**
- Kunden, deren IdP `emailVerified` nicht setzt UND deren User schon lokal existieren, müssen auf einen IdP wechseln, der das Feld korrekt liefert. Casdoor v1.4+ tut das per Default; alle größeren Public-Provider (Google, GitHub via Casdoor, Keycloak ≥ 18) ebenfalls.
- Fehlermeldung im Frontend muss klar sein („IdP hat Ihre Email nicht verifiziert — bitte Profil ergänzen") — `frontend/src/pages/Login.tsx` zeigt den Backend-Error generisch an; ein präziserer Error-Mapping kommt in Sprint 57.

## Verifikation

- `backend/internal/integration_test/auth_oidc_email_verified_real_test.go` enthält zwei Integration-Tests:
  - `TestOIDC_EmailVerified_LinkingGate` — negative case: unverified email + existing user → `ErrEmailNotVerified`, `oidc_subject` bleibt unverändert.
  - `TestOIDC_EmailVerified_LinksOnVerified` — positive control: verified email + existing user → Link erfolgreich.
- Beide Tests booten Postgres 16 via `testcontainers-go` und laufen alle Migrationen.
- Manueller Smoke nach Deploy: in einer Test-Casdoor-Instanz einen User mit nicht-verifizierter Email anlegen, versuchen sich gegen Vakt mit einer Lokal-User-Email einzuloggen → erwarteter Fehler.

## Abgelehnte Alternativen

- **JWT-`id_token` parsen statt `/api/get-account`-Profile-Feld** — würde die Standard-OIDC-`email_verified`-Claim direkt nutzen, ist aber ein größerer Umbau (token-introspection statt opaque-token). Casdoor-Profile-Feld reicht, weil Casdoor selbst die Claim aus dem upstream-IdP übernimmt.
- **Login komplett ablehnen wenn `emailVerified=false`** — würde Erst-Logins bei strikten IdPs blocken. Wir blocken nur das *Linking auf existierende Lokal-Accounts*, nicht die *Erstellung neuer Accounts via OIDC*.
- **Email-Match deaktivieren, nur über Sub linken** — würde manuelle Re-Provisionierung erfordern wenn ein User von Password auf OIDC wechselt. Verschiebt das Sicherheitsproblem auf die Admin-Schulter ohne UX-Vorteil.
