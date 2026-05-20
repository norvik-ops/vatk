# ADR-0002: Elastic License v2 als Lizenzmodell

**Status:** Accepted  
**Datum:** 2026-02-01

## Kontext

Vakt ist source-available, aber kommerziell. Die Lizenz muss:

- Kunden den Self-Eigenbetrieb erlauben (das ist das Verkaufsargument)
- Dritte daran hindern, Vakt als Managed-Service weiterzuverkaufen (Konkurrenz aus dem eigenen Code)
- Audit + Fork durch Sicherheitsteams ermöglichen
- DACH-rechtlich verständlich sein

## Entscheidung

**Elastic License v2 (ELv2)**. Sie erlaubt freien Eigenbetrieb, verbietet aber explizit das Anbieten von Vakt-Funktionalität als gehosteten Service an Dritte.

## Alternativen

- **MIT / Apache 2.0** — verworfen: erlaubt AWS-/Google-Cloud-Style „rebrand und vermieten" ohne Gegenleistung. Vakt ist kein Forschungsprojekt, das davon profitiert.
- **AGPL** — verworfen: Copyleft erschreckt Enterprise-Kunden (Legal-Reviews); zwingt Modifikationen nach upstream — Kunden mögen das nicht.
- **Proprietär / Closed Source** — verworfen: widerspricht dem Selbstbild als Security-Tool. Auditoren wollen den Source sehen.
- **BSL (Business Source License)** — erwogen, aber ELv2 ist bekannter und einfacher im Wortlaut.

## Konsequenzen

### Positive

- Kunden dürfen forken, patchen, intern beliebig deployen.
- MSPs deployen pro Kunde eine eigene Instanz und bilanzieren die Lizenzen einzeln — kein Konflikt mit ELv2.
- „Source-available" reicht für die meisten Security-Audits.

### Negative

- Keine offizielle OSI-Open-Source-Lizenz — Vakt ist nicht „Open Source" im strikten Sinne.
- Einige Communities (Debian, manche Stiftungen) lehnen ELv2 ab. Wir kommen damit klar.

### Neutrale

- SPDX-Identifier `Elastic-2.0` in jedem Code-File-Header.

## Referenzen

- `LICENSE`
- `NOTICE`
- https://www.elastic.co/licensing/elastic-license
