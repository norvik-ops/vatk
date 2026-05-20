# ADR-0011: OpenTelemetry als opt-in, nicht als Pflicht

**Status:** Accepted  
**Datum:** 2026-05-20

## Kontext

Self-hosted Kunden brauchen Observability, um eine produktive Instanz zu betreiben. OpenTelemetry (OTel) ist der Industrie-Standard für Distributed Tracing und Metriken.

OTel produktiv einzubauen heißt aber: Standard-Konfiguration sendet Traces an einen OTel-Collector (Jaeger, Tempo, Honeycomb). Das ist ein **Phone-Home** — selbst wenn der Collector lokal läuft, ist die Defaultkonfiguration mit Cloud-Endpoints schlecht.

## Entscheidung

**OpenTelemetry-Instrumentation immer im Code vorhanden**, aber:

1. Default-Export ist **deaktiviert** (no-op Exporter).
2. Aktivierung nur über explizite Env-Variablen:
   - `OTEL_EXPORTER_OTLP_ENDPOINT` (z.B. `http://tempo:4317`)
   - `OTEL_SERVICE_NAME=vakt-api`
3. Ein **optionaler** Compose-Stack (`docker-compose.observability.yml`) startet Tempo + Loki + Grafana lokal — opt-in via `docker compose --profile observability up`.
4. Startup-Log warnt klar: „OTel disabled — set OTEL_EXPORTER_OTLP_ENDPOINT to enable".

Der OTel-Endpoint ist immer eine Kunden-Adresse — Vakt sendet niemals Telemetrie an Norvik-Server (ADR-0001).

## Alternativen

- **OTel verpflichtend mit zentralem Collector** — verworfen: Phone-Home-Verstoß (ADR-0001).
- **Kein OTel, eigenes Tracing** — verworfen: erfindet Industrie-Standard neu, schwer zu pflegen, schlechter Devex.
- **OTel als Build-Tag** — erwogen, aber Build-Tags machen Releases komplizierter (separate Binaries).

## Konsequenzen

### Positive

- Operability für Kunden mit OTel-Stack out of the box.
- Code ist immer instrumentiert — Kunden müssen sich nicht zwischen „performant" und „observable" entscheiden.
- Compatible mit existierenden DACH-OTel-Setups (Tempo, Grafana Loki on-prem).

### Negative

- Ein kleiner Overhead durch die OTel-API-Calls auch wenn der Exporter no-op ist (vernachlässigbar).
- Dokumentation muss beide Wege beschreiben (Default off, opt-in mit Stack).

### Neutrale

- Prometheus-Metriken werden weiterhin separat über `/metrics` ausgeliefert (`VAKT_METRICS_ENABLED=true`).

## Referenzen

- ADR-0001 (Phone-Home-Verbot)
- `backend/internal/shared/telemetry/` (Sprint 9)
- `docker-compose.observability.yml` (Sprint 9)
