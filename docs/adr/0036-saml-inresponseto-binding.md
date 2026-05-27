# ADR-0036: SAML `InResponseTo`-Binding über signiertes Cookie

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 56 (Marktreife-Welle 1)
**Related:** Audit-Befund F5

## Kontext

`saml_direct.go:SAMLDirectACS` ruft `sp.ParseResponse(c.Request(), nil)` — der zweite Parameter ist `possibleRequestIDs []string`, der einer der wichtigsten Replay-Schutzmechanismen in SAML 2.0 ist. Ein `nil`-Wert bedeutet: crewjam/saml akzeptiert **jede** signierte Assertion vom IdP, ohne zu prüfen, ob sie auf einen vom SP ausgelösten `AuthnRequest` antwortet.

Folgen:

- Ein Angreifer, der jemals eine gültige SAML-Assertion abgefangen hat (z.B. via IdP-Bug, leakendes Proxy-Log, downloaded Browser-Network-Tab), kann sie beliebig oft gegen den ACS abspielen.
- Ein bösartiger IdP (oder ein durch DNS umgeleiteter IdP-Endpoint) kann unaufgeforderte Assertions einreichen.
- In Kombination mit einem fehlerhaften XML-DSig-Implementierungsbug am IdP-Ende (z.B. XSW-Class-Attacks) wäre der fehlende `InResponseTo`-Check ein Multiplikator.

Die crewjam/saml-Library verlangt aktiv eine Liste der momentan ausstehenden Request-IDs — sie macht den Match (`assertion.InResponseTo == one_of(requestIDs)`) selbst, sobald wir die Liste übergeben.

## Entscheidung

Wir binden den AuthnRequest an die Browser-Session des ausstellenden Nutzers über ein HMAC-signiertes, kurz-lebiges HttpOnly-Cookie:

1. **`SAMLInitiate`**: Statt `sp.MakeRedirectAuthenticationRequest("")` zerlegen wir den Flow in zwei Schritte:
   - `sp.MakeAuthenticationRequest(...)` → liefert die `*AuthnRequest`-Struktur, deren `.ID` wir abgreifen
   - `authReq.Redirect("", sp)` → liefert die signierte Redirect-URL für den Browser
   - Wir signieren die `ID` mit `HMAC-SHA256(HKDF(masterKey, "vakt-saml-reqid-v1"), id)` und setzen sie als Cookie `saml_req_id`:
     - `HttpOnly: true`, `Secure: <derived from request scheme>`, `SameSite: None` (IdP-Cross-Origin-POST braucht das)
     - `Path: /api/v1/auth/saml`, `MaxAge: 600` Sekunden
2. **`SAMLDirectACS`**: 
   - Liest `saml_req_id`-Cookie, verifiziert HMAC, extrahiert `id`
   - Übergibt `[]string{id}` als `possibleRequestIDs` an `sp.ParseResponse(...)`
   - Löscht das Cookie sofort (MaxAge = -1) — Single-Use, kein Replay möglich
   - Wenn das Cookie fehlt/ungültig: `possibleRequestIDs` bleibt `nil` → crewjam/saml lehnt jede Assertion mit nicht-leerem `InResponseTo` ab

## Warum Cookie statt Redis?

- **Stateless**: Wir vermeiden zusätzliche Pool-Calls in den SAML-Pfad
- **Browser-bound**: Das Cookie hängt am Browser, der den Initiate-Request stellte — ein anderer Browser-Session kann das Cookie ohne XSS oder TLS-Break nicht klauen
- **HKDF-derived Key**: Konsistent mit der bestehenden HKDF-Architektur (`cmd/api/main.go:379-384`)
- **Single-Use Cookie**: ACS löscht das Cookie sofort. Ein zweites POST würde mit `nil`-Liste fehlschlagen
- **Kein Redis-Coupling**: Falls Redis ausfällt, wäre der SAML-Login sonst blockiert

Trade-off: Bei einem mehrfach-Browser-Workflow (User klickt im SP zweimal „Login mit SAML"), gewinnt der zweite Klick — das erste Cookie wird überschrieben. Akzeptabel, weil normaler SAML-Flow synchron ist.

## Konsequenzen

**Positiv:**
- Klassischer SAML-Replay ist nicht mehr möglich.
- Single-Use Enforcement (Cookie-Clear auf ACS) gibt zusätzlich Defense-in-Depth.
- Kein neuer Infrastruktur-Punkt (Redis bleibt unverändert involviert oder nicht).

**Negativ:**
- Browser, die `SameSite=None`-Cookies blockieren (sehr restriktive Anti-Tracking-Setups), könnten den Flow brechen. Akzeptabel — SAML SP-initiated braucht das Cross-Site-Cookie-Verhalten ohnehin.
- IdP-initiated SAML (IdP startet den Flow ohne unseren `AuthnRequest`) liefert keine `InResponseTo`-Claim — diese Assertions werden mit unserem neuen Code abgelehnt. Das ist **die richtige Entscheidung** (IdP-initiated ist anti-SAML-Best-Practice), aber muss in der Setup-Doku stehen: „Nur SP-initiated SAML wird unterstützt."

## Verifikation

- `internal/auth/saml_request_id_test.go` — 6 Unit-Tests:
  - Round-Trip Sign+Verify
  - Reject mit anderem Master-Key (HKDF-Domain-Trennung)
  - Reject mit getampter Signatur
  - Reject mit getampter ID
  - Reject malformed cookies (leer, ohne `.`, leading/trailing)
  - Reject leere ID beim Signen
- `make test` grün im `internal/auth/`-Paket nach Änderung
- Manueller Smoke (nach Deploy):
  - IdP-Login starten → Browser zeigt Cookie `saml_req_id` mit kurzem Wert
  - Login-Flow zu Ende führen → Cookie gelöscht, Access-Token erhalten
  - SAML-Response per `curl` ohne Cookie an ACS senden → 401 mit `AUTH_SAML_INVALID_ASSERTION`

## Abgelehnte Alternativen

- **Server-side state (Redis-Set per Org)**: Mehr Code, mehr Failure-Modes, kein Vorteil gegenüber Cookie
- **`possibleRequestIDs = nil` lassen + nur signature-trust**: Genau der jetzige Bug. Lehnen wir ab.
- **Casdoor RequestID-Tracking nutzen**: Funktioniert nur für die Casdoor-Proxy-Variante, nicht für `saml_direct.go`
