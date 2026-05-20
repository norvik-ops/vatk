# ADR-0003: Paseto V4 statt JWT für Authentifizierung

**Status:** Accepted  
**Datum:** 2026-02-01

## Kontext

Token-basierte Authentifizierung ist Pflicht (REST-API, OIDC-Tokens). Die Industrie-Default-Wahl ist JWT, hat aber bekannte Schwachstellen:

- **Algorithm Confusion** (`alg: none`, `alg: HS256` mit RSA-Public-Key als Secret)
- Schlechte Default-Bibliotheken in vielen Sprachen
- Generelle Komplexität: zu viele Modi (RS256/HS256/ES256/EdDSA), zu viele optionale Felder, gefährlich anzupassen
- Hat zu mehreren öffentlichen CVE-Familien geführt

Sicherheits-Produkt darf nicht mit fragiler Auth-Plumbing starten.

## Entscheidung

**Paseto V4 (Local Symmetric)** für interne Access-Tokens. Single-Algorithm-Format (XChaCha20-Poly1305), kein Algorithm-Negotiation, deutlich kleinere Angriffsfläche.

OIDC-/SAML-Tokens von externen IdPs werden über die jeweilige Library validiert (Casdoor-Anbindung) — aber intern wird daraus ein Paseto-Token gemacht.

## Alternativen

- **JWT mit strikt einer Algorithmus-Allowlist** — kann sicher sein, ist aber fragil. Eine PR die fälschlicherweise eine zweite Algorithmus-Variante zulässt = Sicherheitslücke. Default-fail-open.
- **Opaque Tokens + DB-Lookup** — verworfen: jeder Request kostet eine DB-Round-Trip. Bei Multi-Replica nicht skalierbar. (Wir nutzen Opaque Tokens nur für Refresh + API-Keys.)

## Konsequenzen

### Positive

- Algorithm-Confusion-Angriffe schlicht unmöglich.
- Kleinere Library-Surface, weniger CVE-Exposure.
- Tokens sind asymmetrisch verschlüsselt (XChaCha20-Poly1305 = authenticated encryption).

### Negative

- Weniger bekannt als JWT — Onboarding für neue Entwickler braucht ADR-Erklärung (= dieses Dokument).
- Tools wie jwt.io zeigen Paseto nicht direkt an — Debugging anders.

### Neutrale

- `VAKT_SECRET_KEY` (32 Byte hex) ist gleichzeitig Paseto-Key und Master-Encryption-Key. Beim Rotieren beides drehen.

## Referenzen

- `backend/internal/auth/paseto.go`
- https://paseto.io
- ADR-0010 (Master-Key-Strategie)
