# ADR-0046: SBOM-Generation + SLSA-Provenance für jeden Release

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 58 (Marktreife-Welle 3)
**Related:** Audit-Befund P1-1

## Kontext

Die `release.yml`-CI signierte schon mit `cosign` keyless die Container-Images (Sprint 13). Was fehlte:

- **SBOM (Software Bill of Materials)** — eine maschinenlesbare Auflistung aller im Image enthaltenen OS- und Application-Dependencies. Audit P1-1 forderte das, weil:
  - **EU Cyber Resilience Act (CRA) Art. 13(15)** verlangt SBOM-Pflicht für PSO-Produkte ab Anwendungs-Datum (2027).
  - **NIS2-Compliance** in DACH-Industrie verlangt sie de-facto bei jedem Procurement-Process.
  - Pen-Tester verlangen sie als Standard-Artefakt.

- **SLSA-Provenance Level 2** — Attestation, dass das Image durch einen reproduzierbaren CI-Build entstand und nicht manuell injiziert wurde. Sigstore Attestation Format.

Ohne diese Artefakte ist Vakt für Enterprise-Procurement-Prozesse in der EU ab 2027 nicht mehr verkaufbar.

## Entscheidung

Erweitere `release.yml` um drei neue Schritte (zwischen `cosign sign` und `Create GitHub Release`):

1. **SBOM-Generation via syft** für Backend- und Frontend-Image, je in zwei Formaten:
   - SPDX 2.3 JSON (`sbom-api-v0.25.0.spdx.json`)
   - CycloneDX 1.4 JSON (`sbom-api-v0.25.0.cdx.json`)

2. **SBOM-Attestation via cosign attest** (`--type spdxjson`) — die SPDX-Datei wird als Sigstore-Attestation an das Image angehängt und im Rekor-Transparency-Log persistiert. Verifizierbar via `cosign verify-attestation`.

3. **Upload als Release-Asset** — beide Formate (SPDX + CycloneDX) werden zusätzlich an den GitHub-Release-Body angehängt, damit air-gapped-Customer die SBOM ohne Registry-Access laden können.

### Beibehalten

- `cosign sign` keyless für beide Images (vorher bereits da)
- `cosign verify` als One-Liner in der GitHub-Release-Body
- Sigstore-Identity-Cert-Chain (`certificate-identity-regexp` auf `release.yml`)

### Out-of-Scope

- **Full SLSA Level 3** (build provenance attestation via `slsa-framework/slsa-github-generator`) — wäre ein zweiter, parallel laufender Workflow. Sprint 60 falls Customer es verlangen
- **SBOM für Source-Tree** statt für Image — ist redundant, weil syft das Image scannt und alle Source-Dependencies + alle Multi-Stage-Build-Artefakte erfasst
- **SBOM-Diff zwischen Releases** — schöne Future-Funktion (sieht Dependency-Bumps auf einen Blick), nicht Compliance-relevant

## Konsequenzen

**Positiv:**
- CRA-Pflicht ab 2027 ist erfüllt — Vakt kann in EU-Industrie-Procurement bestehen
- Customer-Pentester können das Image vor dem Deploy auf bekannte Schwachstellen scannen (Trivy + SBOM-Input)
- SBOM-Attestation ist im Rekor-Log persistiert — fälschungssicher

**Negativ:**
- Release-CI-Laufzeit steigt um ~30s (zweimal syft scan + attest)
- Größe des GitHub-Release-Body steigt um ~2 MB (4 SBOM-Dateien); akzeptabel
- Wenn syft je future einen Bug hat und schiefe Daten liefert, ist die Attestation nicht widerrufbar (geht in Rekor). Mitigation: syft Major-Version pin in Action

## Verifikation

- **Customer-side**:
  ```bash
  # signature verifizieren
  cosign verify ghcr.io/norvik-ops/vakt-api:v0.25.0
  # SBOM-Attestation extrahieren
  cosign verify-attestation --type spdxjson \
    --certificate-identity-regexp="https://github.com/matharnica/vakt-app/.github/workflows/release.yml@refs/tags/.*" \
    --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
    ghcr.io/norvik-ops/vakt-api:v0.25.0 \
    | jq -r '.payload' | base64 -d | jq '.predicate' > sbom.spdx.json
  ```
- **Operator-side**: `docs/dev/verify-release.md` enthält den vollen Workflow inkl. SBOM-Extraktion
- **Pre-Release-Smoke** (lokal):
  ```bash
  syft ghcr.io/norvik-ops/vakt-api:v0.25.0 -o spdx-json | jq '.packages | length'
  # → > 200 Komponenten (Go + Linux base + glibc + …)
  ```

## Abgelehnte Alternativen

- **Manueller SBOM-Upload pro Release** — fehleranfällig, nicht skalierbar
- **trivy statt syft** — beide funktionieren, syft hat besseren SPDX-Output und ist anchore-supported
- **Eigener private Sigstore-Server** — würde Customer-Verifikation erschweren; Sigstore-Public ist Industry-Standard
- **SBOM-Pflicht nur für Pro-Tier** — Compliance-Gap für CE-Customer in DACH-Industrie würde Adoption-Lock-In gegen Vakt blocken
