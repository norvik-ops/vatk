# ADR-0009: OpenAPI-Spec als Single Source of Truth, embedded

**Status:** Accepted  
**Datum:** 2026-05-20

## Kontext

Vor diesem ADR existierten zwei OpenAPI-Specs nebeneinander:

1. `docs/api/openapi.yaml` — 3981 Zeilen, 70+ Endpoints, gepflegt manuell beim Hinzufügen neuer Routes.
2. `backend/internal/shared/apidocs/handler.go::generateSpec()` — hardcoded Go-Funktion mit ~10 Endpoints, die der Server tatsächlich auslieferte.

Resultat: Kunden, die Swagger-UI ansurften, sahen nur 10 Endpoints. Wer die `docs/api/openapi.yaml` direkt ansurfte, sah 70 — aber die war nirgendwo verlinkt. Eine externe Analyse markierte das als P0.

## Entscheidung

**Eine einzige Spec-Datei** wird zur Build-Zeit ins Backend-Binary embedded:

- Datei: `backend/internal/shared/apidocs/openapi.yaml`
- Embedding: `//go:embed openapi.yaml` in `embed.go`
- Auslieferung: `GET /api/v1/openapi.yaml` (öffentlich, unauthenticated)
- Swagger-UI: `GET /api/docs` lädt diese URL
- Public-Repo-Sync: kopiert die embedded Datei nach `docs/api/openapi.yaml` für SDK-Generatoren und externe Konsumenten

Die hardcoded Go-Funktion `generateSpec()` wurde komplett entfernt. CI-Test `spec_test.go` prüft, dass die YAML valide ist und dass eine Liste von Pflicht-Endpoints darin steht — PRs, die Endpoints aushebeln, gehen rot.

## Alternativen

- **swaggo/swag** (Code-Annotations → Spec-Gen) — erwogen: würde Spec aus Handler-Kommentaren generieren. Verworfen weil: bestehende Handler haben keine Annotations (Aufwand >1 Sprint sie nachzuziehen), und der Wert ist begrenzt — die Spec wird primär für UI/SDK gelesen, nicht für Handler-Doku.
- **Spec im `docs/api/` lassen, Backend liest sie zur Laufzeit** — verworfen: bricht den „self-contained binary"-Vertrag von Vakt; Pfad muss konfigurierbar sein, eine weitere Quelle für Operations-Fehler.
- **Zwei Specs synchronisiert halten** (Status quo bis Mai 2026) — verworfen: erwiesen unzuverlässig.

## Konsequenzen

### Positive

- Single Source of Truth: das was im Build ist, ist was der Server ausliefert.
- CI verhindert Divergenz zwischen Code und Doku.
- Kunden können aus der Spec eigene SDKs generieren (`openapi-generator`, `oapi-codegen`, …).

### Negative

- Spec-Updates erfordern Backend-Rebuild — keine „nur Doku-Änderung"-PRs mehr.
- Spec-Datei ist 4000+ Zeilen YAML — anstrengend zu pflegen, aber überschaubar.

### Neutrale

- Falls künftig zu swaggo/swag gewechselt wird, ist `openapi.yaml` die garantierte Snapshot-Quelle — Migration ist datentechnisch trivial.

## Referenzen

- `backend/internal/shared/apidocs/openapi.yaml`
- `backend/internal/shared/apidocs/spec_test.go`
- `.github/workflows/sync-public-repo.yml`
