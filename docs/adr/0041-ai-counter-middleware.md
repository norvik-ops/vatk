# ADR-0041: AI-Counter als zentrale Echo-Middleware

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 57 (Marktreife-Welle 2)
**Related:** Audit-Befund F3, [[ADR-0021]] (NIS2-Wizard CE-vs-Pro-Cut), [[ADR-0024]] (Modell-Auswahl)

## Kontext

Vakts Community-Edition limitiert AI-Anfragen auf **25 pro Monat pro Org**, um Pro-Tier zu monetarisieren. Die Gate-Logik war inline in den Handlern implementiert (`if err := h.checkCELimit(c); err != nil { return err }`).

Drift war vorhersehbar: 13 AI-Endpoints, davon hatten nur 6 das Gate inline:

| Endpoint | Gate inline? | Counter incrementiert? |
|---|---|---|
| `POST /ai/report` (`GenerateReport`) | ❌ | ✅ (via gateAndGenerate in service.go) |
| `POST /ai/advice` (`ComplianceAdvice`) | ✅ | ✅ |
| `POST /ai/draft-policy` (`DraftPolicy`) | ✅ | ✅ |
| `POST /ai/incident-guide` (`IncidentResponseGuide`) | ✅ | ✅ |
| `POST /ai/chat/stream` (`ChatStream`) | ✅ | ✅ |
| `POST /ai/controls/:id/explain` (`GapExplain`) | ✅ | ❌ |
| `POST /ai/risks/:id/narrative` (`RiskNarrative`) | ✅ | ❌ |
| `POST /ai/agent/run` (`AgentRun`) | ❌ | ❌ |
| `POST /ai/agent/runs/:run_id/approve` | n/a | n/a |
| `POST /ai/agent/runs/:run_id/reject` | n/a | n/a |
| `GET /ai/status`, `GET /ai/usage`, `GET /ai/models`, `GET /ai/insights`, `DELETE /ai/insights/:id` | n/a (Read-Only) | n/a |

Konsequenzen:

- **CE-Customer können `/ai/report` und `/ai/agent/run` unbegrenzt aufrufen** — der monetäre Druck zum Pro-Upgrade entfällt für die zwei lukrativsten Use-Cases (Audit-Reports + Agentic-AI).
- **GapExplain + RiskNarrative haben das Gate, aber zählen nicht hoch** — das Limit wird also nie ausgelöst, selbst wenn ein Customer 100×/Tag GapExplain aufruft.
- Jede neue AI-Route hat ein 50%-Risiko, ohne Gate auszuliefern.

## Entscheidung

Wir verschieben die Gate-Logik in eine **Echo-Middleware** `ai.RequireAILimit(svc)` und wenden sie auf jede LLM-erzeugende Route an. Die Route-Definition in `routes.go` ist damit die *einzige* Source of Truth — neue Routen ohne Gate sind sofort sichtbar im Diff.

```go
aiLimit := RequireAILimit(svc)
g.POST("/ai/report", h.GenerateReport, aiLimit)
g.POST("/ai/agent/run", agentH.AgentRun, aiLimit)
g.POST("/ai/controls/:id/explain", h.GapExplain, aiLimit)
// …
```

Inline-Calls (`h.checkCELimit(c)`) **bleiben** als Defense-in-Depth — sollte jemand zukünftig die Middleware aus einer Route entfernen ohne den Audit-Trail zu lesen, fängt der Inline-Check.

### Record-Calls in SSE-Endpoints

`GapExplain` und `RiskNarrative` bekommen einen `usage.Record(status:"ok", RequestID: "controls.explain" | "risks.narrative")`-Call am Ende ihrer Handler. Ohne diesen würde die Middleware das Gate nie auslösen, weil `CEMonthlyUsage` nur `status='ok'`-Rows zählt.

### Static Route-Coverage-Test

`middleware_test.go::TestRequireAILimit_RouteWiringHasGateEverywhere` pinnt die Liste der gateten Routes als String-Vektor. Wenn jemand eine neue AI-Route hinzufügt ohne den Vektor zu aktualisieren, schlägt der Test fehl. Wenn jemand eine Route umbenennt ohne den Vektor zu aktualisieren, ebenfalls.

## Konsequenzen

**Positiv:**
- Pro-Tier-Druck funktioniert wieder
- Neue Routes erben das Gate by-construction (Middleware-Pattern)
- Route-Test verhindert künftige Drift
- `RequireAILimit` ist unit-testbar (4 Sub-Cases ohne DB)

**Negativ:**
- Inline-Checks in den Handlern sind jetzt redundant (Defense-in-Depth) — werden in Sprint 58 entfernt sobald die Middleware in Production bestätigt ist
- `Record`-Calls in SSE-Handlern können nur Status `"ok"` zuverlässig setzen, weil Tokens unbekannt sind. Cost-Tracking pro Endpoint bleibt nur für nicht-SSE-Pfade akkurat — akzeptabel
- Approve/Reject-Endpoints haben kein Gate (Sub-Aktion eines bereits geGate-ten AgentRun) — Konvention dokumentiert

## Verifikation

- **Unit-Tests** (`internal/services/ai/middleware_test.go`):
  - `TestRequireAILimit_NoTrackerIsNoOp` — kein UsageTracker → Pass-Through (Bootstrap)
  - `TestRequireAILimit_ProBypass` — Pro-License im Context → Pass-Through
  - `TestRequireAILimit_NilLicenseTreatedAsCE` — keine License → CE-Limit gilt (kein Silent-Bypass)
  - `TestRequireAILimit_RouteWiringHasGateEverywhere` — pinnt die 8 gateten POST-Routes
- **Existing Suite** grün (`go test ./internal/services/ai/`)
- **Smoke-Plan** (Post-Deploy):
  1. CE-Setup, 25 `/ai/report`-Calls → 26. soll 402 zurückgeben
  2. Pro-Aktivierung → CE-Limit ignoriert (Demo: `curl /ai/usage` zeigt `is_pro: true`)
  3. `curl /ai/agent/run` mit verbrauchtem CE-Counter → 402 (vorher: 200)

## Abgelehnte Alternativen

- **Inline-Check beibehalten + nur fehlende Endpoints ergänzen** — adressiert den aktuellen Bug, nicht die Drift-Ursache. Sprint-58 hätte das gleiche Problem nochmal
- **Pro-Tier-Erkennung via Path-Prefix** — `g.POST("/ai/pro/agent/run", ...)` — verkomplizierte Frontend-Pfad-Mapping, kein Vorteil
- **HTTP-Hook nach Response** — würde Spec-Sentinel-Bytes brauchen, fragil bei SSE-Streams
- **Counter im Frontend prüfen** — User kann jede UI-Logik umgehen via direkter API-Call; Gate gehört server-side
