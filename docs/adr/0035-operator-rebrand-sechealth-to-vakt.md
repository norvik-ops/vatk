# ADR-0035: Operator — Rebrand abschließen, CRD-Group + Kind angleichen

**Status:** Akzeptiert
**Datum:** 2026-05-27
**Entscheider:** Stefan Moseler
**Sprint:** 56 (Marktreife-Welle 1)
**Related:** Audit-Befund F11

## Kontext

Der Operator unter `operator/` war im Rebrand SecHealth → Vakt nur halb umgestellt. Das Audit hat die Inkonsistenz exakt belegt:

- **Go-Code** verwendet bereits `Group: "secrets.vakt.io"` (`api/v1alpha1/register.go:11`), `Kind: VaktSecret`, `Plural: vaktsecrets`.
- **CRD-Manifeste** in `config/crd/sechealthsecret.yaml`, `helm/templates/crd.yaml` und die zugehörigen RBAC-YAMLs deklarieren weiter `Group: secretops.sechealth.io`, `Kind: SecHealthSecret`, `Plural: sechealthsecrets`.
- **Chart.yaml** Maintainer-Email: `hello@sechealth.io` (Domain existiert nicht mehr).

Konsequenz bei einem Helm-Install in einem neuen Cluster:

1. CRD `sechealthsecrets.secretops.sechealth.io` wird installiert.
2. Operator startet, sucht CRs der Gruppe `secrets.vakt.io/vaktsecrets`.
3. API-Discovery findet keine Custom-Resource — Watcher liefert null Events.
4. Reconcile-Loop läuft nie, Anwender bekommen keine Fehlermeldung, die Symptome ähneln einem still-failenden Crash.

## Entscheidung

Wir ziehen alle externen Manifeste auf die Go-Group nach:

| Datei | Vorher | Nachher |
|---|---|---|
| `helm/templates/crd.yaml` | Group `secretops.sechealth.io`, Kind `SecHealthSecret`, Plural `sechealthsecrets`, Short `sss` | Group `secrets.vakt.io`, Kind `VaktSecret`, Plural `vaktsecrets`, Short `vs` |
| `helm/templates/rbac.yaml` | gleiche Gruppe in 3 Rule-Blöcken | analog umgestellt |
| `helm/Chart.yaml` | `sechealth-operator`, `hello@sechealth.io` | `vakt-operator`, `hello@norvikops.de` |
| `helm/values.yaml` | image `ghcr.io/sechealth/sechealth-operator` | `ghcr.io/norvik-ops/vakt-operator` |
| `helm/templates/_helpers.tpl` | Template-Namen `sechealth-operator.*` | `vakt-operator.*` |
| `config/crd/sechealthsecret.yaml` | Group + Kind + Beschreibungen | umbenannt zu `config/crd/vaktsecret.yaml` mit Group `secrets.vakt.io` |
| `config/rbac/role.yaml` | ClusterRole-Name + 3 Rule-Gruppen | analog umgestellt |
| `internal/controller/sechealthsecret_controller{,_test}.go` | nur Dateiname war alt | umbenannt zu `vaktsecret_controller{,_test}.go` |

## Migration für bestehende Installationen

Da gemäß [[project_install_base]] Vakt nur in Demo (ephemer) und Pentest-lokal läuft, ist eine produktive `SecHealthSecret`-Installation auszuschließen. Für externe Forks / Early-Adopters:

```bash
# Vor dem Upgrade:
kubectl get sechealthsecrets.secretops.sechealth.io -A -o yaml > old-crs.yaml

# Helm upgrade rollt die neuen CRD/RBAC aus. Alte CR-Instanzen werden NICHT
# automatisch migriert (k8s erlaubt keinen group rename). Manuell umschreiben
# (sed) und unter neuer API anwenden:
sed -e 's|secretops.sechealth.io/v1alpha1|secrets.vakt.io/v1alpha1|g' \
    -e 's|kind: SecHealthSecret|kind: VaktSecret|g' \
    old-crs.yaml | kubectl apply -f -

# Alte CRD-Instanz löschen:
kubectl delete crd sechealthsecrets.secretops.sechealth.io
```

## Konsequenzen

**Positiv:**
- Helm-Install führt jetzt zu einer funktionalen Watch-Pipeline.
- Operator-Chart-Name + Image-Pfad konsistent mit dem Repo (`norvik-ops/vatk` Mirror).
- Unit-Test pinnt die Group-Konsistenz, weitere Drifts werden CI-rot.

**Negativ:**
- Breaking-Change für eventuelle Early-Adopter — Migrationspfad oben.
- Operator-Chart bekommt einen neuen Major im SemVer-Sinne (0.1.0 → 0.2.0).

## Verifikation

- `operator/api/v1alpha1/register_test.go` — `TestGroupVersionMatchesCRDManifests` liest alle vier Manifest-Dateien und stellt sicher, dass die in `register.go` definierte Group enthalten ist UND keine `secretops.sechealth.io`/`sechealthsecrets`-Reste mehr existieren.
- `go test ./operator/... && go build ./operator/...` grün.
- `grep -r "sechealth" operator/` liefert nur noch ADR-Querverweise.
- Smoke nach Deploy: `kubectl apply -f operator/config/crd/vaktsecret.yaml && kubectl explain vaktsecret`.

## Abgelehnte Alternativen

- **Group im Go-Code zurück auf `secretops.sechealth.io` setzen** — würde den Rebrand rückwärts machen. SecHealth ist tot.
- **Beide Gruppen parallel anbieten** — verdoppelt die CRD-Pflege und führt zu Verwirrung. Wir haben keine Legacy-Production-Last (Demo + Pentest-only).
