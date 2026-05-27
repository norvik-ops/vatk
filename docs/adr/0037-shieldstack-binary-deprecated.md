# ADR-0037: `shieldstack`-Binary deprecated, Git-History-Rewrite ausstehend

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 57 (Marktreife-Welle 2)
**Related:** Audit-Befund F9

## Kontext

Das Auditos-Audit hat im Repo-Tree ein 10,5 MB ELF-Binary unter `backend/shieldstack` gefunden und richtigerweise als Repo-Hygiene-Problem markiert. Verifikation gegen `git log` und `git cat-file -e HEAD:backend/shieldstack` zeigt:

- Die Datei wurde in `f5cbca6` (initial open-source release, Mai 2026) versehentlich commitet.
- In `b83890c` wurde die Datei aus `HEAD` entfernt — der zugehörige Go-Source-Code ebenfalls.
- `.gitignore` enthält bereits den Eintrag `backend/shieldstack`, der Build-Artefakte lokal abdeckt.
- Im aktuellen `HEAD` ist das Binary **nicht** mehr vorhanden — die Audit-Analyse hat die Datei im lokalen Working-Tree gesehen, das ist aber nur ein lokaler Build-Output.

Was bleibt: Das Binary existiert weiterhin in der **Git-History** der commits `f5cbca6` und `b83890c`. Wer den public Mirror clont, lädt die ~22 MB History mit (clone-size-bloat) und kann das Binary durch `git checkout f5cbca6 -- backend/shieldstack` jederzeit reaktivieren.

## Entscheidung

Das `shieldstack`-Binary ist **deprecated**. Es gibt keinen Go-Source-Code mehr im Repo, der es produzieren würde — das Binary war ein versehentlich miteingecheckter Build-Output ohne Funktion im aktuellen Produkt.

Drei Sofortmaßnahmen (Sprint 57):

1. **Lokales Build-Artefakt löschen** (`rm backend/shieldstack`). Künftige Builds laden es nicht erneut, weil kein `cmd/shieldstack/`-Verzeichnis mehr existiert.
2. **`.gitignore`-Eintrag bleibt** als Sicherheitsnetz, falls jemand das Binary versehentlich neu erzeugt.
3. **`docs/dev/repo-hygiene.md`** dokumentiert die Lage transparent.

Eine **destruktive Operation** (Git-History-Rewrite via `git filter-repo` oder BFG) wird **bewusst zurückgestellt**:

- Der Public Mirror würde durch Force-Push beschädigt; jeder, der schon einen Clone hat, müsste neu klonen.
- Co-Maintainer (auch wenn aktuell solo) brauchen eine Vor-Warnung.
- Die History-Belastung ist kosmetisch — kein Security-Risiko, weil das Binary keine geheimen Daten enthält und keine Source-Korrelation hat.

Wenn die History-Cleanup erfolgt (z.B. zum v1.0-Release), kommt eine separate ADR mit Disclosure-Plan.

## Was das Binary nicht war

Aus den verbleibenden Strings im ELF (Go-BuildID, debug info) und dem Pfad: Das war eine frühe Demo eines „SecHealth"-Stack-Inspectors — vermutlich ein Service-Healthcheck-Tool, das nie zur Reife kam. Es enthält weder Customer-Daten noch Krypto-Material noch Lizenz-Keys.

## Verifikation

- `git ls-files backend/shieldstack` → keine Treffer (nicht tracked in HEAD).
- `cat-file -e HEAD:backend/shieldstack` → „not in HEAD" (verifiziert in Sprint 57).
- `grep -rn shieldstack backend/ Makefile Dockerfile*` → keine Build-Referenzen.
- Working-Tree-Check nach `rm backend/shieldstack` zeigt das File verschwunden.
- Build-Plan: `make build` produziert das Binary nicht mehr (kein `cmd/shieldstack/`).

## Konsequenzen

**Positiv:** Nach lokaler Löschung ist der Working-Tree clean. Klone neuer Entwickler bekommen das Binary nicht mehr (außer sie checken einen Pre-`b83890c`-Commit aus). `make build` ist deterministisch.

**Negativ / pending:** Public-Mirror-Clones (`norvik-ops/vatk`) bleiben um die Binary-History bloated bis zum History-Rewrite. Ein potenzieller Code-Reviewer könnte den `f5cbca6`-Commit auschecken und das Binary „wiederbeleben" — funktionsfrei, aber Repo-Trust-irritierend.

## Abgelehnte Alternativen

- **Sofort `git filter-repo` + Force-Push** — destructiver Eingriff in den Public Mirror, ohne Disclosure-Plan, wirkt schlampig.
- **Binary durch dokumentierten Build-Artefakt-Build ersetzen** — kein Source-Code mehr, daher kein deterministischer Re-Build möglich.
- **shieldstack-Funktionalität neu schreiben** — Out-of-Scope, das Produkt braucht es nicht.
