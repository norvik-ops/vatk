# Architecture Decision Records

Diese ADRs dokumentieren wichtige Architekturentscheidungen der Vakt-Plattform. Format: Michael Nygard's "Architecture Decision Records" (kurz, ein File pro Entscheidung, fortlaufend nummeriert, **immutable** nach Akzeptanz — Änderungen kommen als neue ADRs, die alte ADRs „supersede").

## Status-Lifecycle

- **Proposed** — vorgeschlagen, noch nicht entschieden
- **Accepted** — entschieden und in Umsetzung
- **Superseded by ADR-NNNN** — durch neuere Entscheidung ersetzt; Datei bleibt für die Historie

---

## Index

### Kategorie: Architektur

| # | Titel | Status | Zusammenfassung |
|---|-------|--------|-----------------|
| [0001](0001-self-hosted-no-phone-home.md) | Self-Hosted Architektur ohne Phone-Home | Accepted | Alle Daten bleiben in der Infrastruktur des Kunden; kein Telemetrie-Endpoint, kein Cloud-Relay. |
| [0004](0004-modul-isolation-via-package-und-prefix.md) | Modul-Isolation via Go-Package und DB-Prefix | Accepted | Jedes Modul hat ein eigenes Go-Package und einen eigenen DB-Tabellen-Prefix; keine direkten Modul-zu-Modul-Imports. |
| [0005](0005-sqlc-modulweise-keine-vollmigration.md) | sqlc modulweise einführen, keine Vollmigration | Accepted | sqlc wird pro Modul eingeführt statt als Big-Bang-Migration; Ausnahmen dokumentiert. |
| [0009](0009-openapi-single-source-of-truth.md) | OpenAPI-Spec als Single Source of Truth, embedded | Accepted | Die OpenAPI-YAML ist die einzige Quelle für API-Typen; Frontend-Client wird daraus generiert. |
| [0011](0011-opentelemetry-optional-opt-in.md) | OpenTelemetry als opt-in, nicht als Pflicht | Accepted | OTel-Tracing ist opt-in via Env-Var; kein Pflicht-Exporter, um einfaches Deployment zu erhalten. |
| [0013](0013-sqlc-row-types-und-feld-mapper.md) | sqlc Row-Types und Feld-Mapper bei divergenter Spalten-Reihenfolge | Accepted | Bei strukturellen Spalten-Abweichungen zwischen Query-Ergebnis und Domain-Struct explizite Mapper schreiben. |
| [0016](0016-public-mirror-via-script.md) | Public Mirror per Script statt inline rsync im CI; Compile-Check als Gate | Accepted | Public-Mirror-Sync läuft als kuratiertes Script mit Compile-Check, damit das Public Repo immer baut. |
| [0018](0018-goroutine-lifecycle-und-panic-eskalation.md) | Goroutine-Lifecycle (Parent-Context-Pflicht) und Panic-Eskalation via `safego.Run` | Accepted | Alle Goroutinen in `internal/` müssen `safego.Run(ctx, name, fn)` verwenden; kein `context.Background()` außer in `cmd/*`. |
| [0019](0019-sse-statt-websocket-fuer-realtime.md) | Server-Sent Events statt WebSockets für alle Realtime-Pfade | Accepted | SSE statt WebSockets für Notifications, Scan-Progress und AI-Streaming — einfacheres Deployment, Nginx-kompatibel. |
| [0023](0023-typed-cross-module-event-contracts.md) | Typed Cross-Module Event Contracts | Accepted | Cross-Module-Events (FindingCreated, BreachNotified usw.) sind typisierte Go-Structs in `platform/events/types.go`. |
| [0031](0031-release-strategie-v1.md) | Phase-1-Release-Strategie (v0.22.0 → v1.0) | Accepted | Stufenplan von v0.22.0 zu v1.0 mit konkreten Gates und Dokumentations-Items. |

### Kategorie: Auth & Security

| # | Titel | Status | Zusammenfassung |
|---|-------|--------|-----------------|
| [0003](0003-paseto-v4-statt-jwt.md) | Paseto V4 statt JWT für Authentifizierung | Accepted | Paseto v4 statt JWT, weil JWT algorithm-confusion-Angriffe ermöglicht; kein `alg`-Header in Paseto. |
| [0010](0010-aes-256-gcm-fuer-app-secrets.md) | AES-256-GCM für Application-Level-Secrets | Accepted | Alle Vault-Secrets werden mit AES-256-GCM verschlüsselt; HKDF-Schlüsselableitung pro Projekt. |
| [0017](0017-api-contract-tests.md) | API-Contract-Tests gegen Backend ↔ Frontend Drift | Accepted | OpenAPI-Spec als Kontrakt; Drift zwischen Backend, OpenAPI und Frontend-Client ist ein CI-Gate. |
| [0020](0020-ai-agent-tool-permissions.md) | AI-Agent darf nur Tools nutzen die der User darf | Accepted | Kein Privilege-Escalation via AI — Agent-Tool-Calls werden gegen RBAC des initiierenden Users geprüft. |
| [0022](0022-auth-tier-cut.md) | Auth-Tier-Cut — SAML CE, SCIM/SIEM/IP-Allowlist Pro | Accepted | SAML 2.0 ist Community-Feature; SCIM, SIEM-Forwarder und IP-Allowlist sind Pro-Features. |
| [0028](0028-nis2-embedded-mode-security.md) | NIS2 Embedded-Mode Security — `frame-ancestors *` bewusst gewählt | Accepted | Clickjacking-Risiko-Analyse für den öffentlichen NIS2-Wizard im Embedded-Modus; Entscheidung dokumentiert. |
| [0032](0032-ai-integrity-richtlinien.md) | AI-Integrity-Richtlinien — Prompt-Injection-Schutz und AI-Sicherheitsgrenzen | Accepted | Prompt-Separator-Standard, Input-Sanitierung, Rate-Limiting per Org, Streaming-Pflicht und Source-Attribution für alle AI-Calls. |

### Kategorie: Produkt & Features

| # | Titel | Status | Zusammenfassung |
|---|-------|--------|-----------------|
| [0006](0006-anonymisierung-statt-hard-delete.md) | Anonymisierung statt Hard-Delete bei DSGVO Art. 17 | Accepted | Gelöschte User werden anonymisiert (kein Hard-Delete), damit Audit-Trail erhalten bleibt. |
| [0007](0007-betriebsrat-mode-write-time-anonymisierung.md) | Betriebsrat-Modus: Anonymisierung beim Schreiben | Accepted | Im Betriebsrat-Modus wird PII gar nicht erst in die DB geschrieben — keine nachträgliche Anonymisierung. |
| [0012](0012-frontend-test-coverage-pragmatisch.md) | Test-Coverage pragmatisch nach Risiko, nicht nach Quote | Accepted | Keine Coverage-Quote; stattdessen risikobasiert: Auth-, Krypto-, Migrations- und Finance-Logik brauchen Tests. |
| [0014](0014-ai-copilot-community-feature.md) | AI Copilot ist Community-Feature, kein Pro-Gate | Accepted | Lokaler AI-Advisor via Ollama ist in der Community-Edition enthalten — kein Pro-Gate. |
| [0015](0015-ephemere-demo-sessions.md) | Ephemere Demo-Sessions pro Visitor (4 h, Random-Credentials) | Accepted | Demo-Orgs werden pro Visitor ephemer angelegt mit zufälligen Credentials; Cleanup nach 4 Stunden. |
| [0021](0021-nis2-wizard-ce-vs-pro-cut.md) | NIS2-Wizard CE vs Pro Cut | Accepted | NIS2-Wizard ist Community (Top-of-Funnel); Branded-PDF, Re-Assessment-History und Multi-Framework sind Pro. |
| [0024](0024-model-selection-policy.md) | Model-Selection-Policy | Accepted | Default-Modell `qwen2.5:3b` (Apache 2.0); Auswahl-Kriterien für alternative Modelle dokumentiert. |
| [0029](0029-demo-login-screen-design.md) | Demo-Login-Screen UI-Design | Accepted | Amber-Banner, Credentials-Card und Auto-Fill-Buttons als Standard-Design für den ephemeren Demo-Flow. |
| [0030](0030-i18n-date-formatting.md) | i18n-konforme Datumsformatierung via `useFormatDate` | Accepted | Alle Datumsangaben im Frontend über den `useFormatDate`-Hook; kein hardcoded `'de-DE'` in React-Komponenten. |

### Kategorie: Lizenz & Compliance

| # | Titel | Status | Zusammenfassung |
|---|-------|--------|-----------------|
| [0002](0002-elastic-license-v2.md) | Elastic License v2 als Lizenzmodell | Accepted | ELv2: Source-available, kostenlos für Eigenbetrieb, kein Managed-Service-Weiterverkauf. |
| [0008](0008-kein-msp-portal.md) | Kein MSP-Portal — Phone-Home-Verstoß | Accepted | Ein zentrales MSP-Portal würde Kundendaten aggregieren und das Kern-Versprechen (kein Phone-Home) brechen. |

---

## Wann eine neue ADR schreiben?

Für Entscheidungen die:

- mehrere Module betreffen
- schwer reversibel sind (Datenbank-Schema, externe Verträge, Lizenzmodell)
- eine erkennbare Alternative hatten (sonst dokumentier es als Code-Kommentar)
- für künftige Entwickler nicht aus dem Code allein erschließbar sind

Faustregel: Wenn du einem neuen Entwickler erklärt, „warum machen wir das so?" und die Antwort länger als ein Absatz ist — schreib ein ADR.

## Template

Siehe [`0000-template.md`](0000-template.md).
