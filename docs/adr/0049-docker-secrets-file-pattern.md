# ADR-0049: Docker Secrets — File-basiertes Secret-Loading

**Status:** Akzeptiert  
**Datum:** 2026-05-29  
**Entscheider:** Stefan Moseler  
**Related:** Mega-Audit v2 Findings C-01 (VAKT_SECRET_KEY in docker inspect), M-01 (SMTP_PASS sichtbar)

## Kontext

Der Mega-Audit v2 hat festgestellt, dass `VAKT_SECRET_KEY`, `VAKT_DB_URL` und `SMTP_PASS`
als Klartext in Container-ENV stehen und damit via `docker inspect` für jeden Nutzer mit
Docker-Socket-Zugriff (Root auf dem Server) lesbar sind. Das ist kein Code-Bug, sondern eine
Ops-Hygiene-Lücke.

Drei Optionen wurden evaluiert:

| Option | Umsetzungsaufwand | Bleibt Secret aus `docker inspect` | Anforderungen |
|---|---|---|---|
| **_FILE-Pattern** | Klein (1 Helper + Compose-Mounts) | Ja | Nur Dateisystem |
| Docker Swarm Secrets | Mittel (Migration zu Swarm) | Ja | Swarm-Mode aktiv |
| HashiCorp Vault / sops | Groß | Ja | Externer Dienst |

## Entscheidung

**_FILE-Pattern** (Option 1): Die App liest Secrets aus Dateipfaden, die via Environment-Variable
angegeben werden (`VAKT_SECRET_KEY_FILE`, `VAKT_DB_URL_FILE`). Beide Varianten werden
backward-kompatibel unterstützt — ohne `_FILE`-Var fällt die Konfig auf die normale ENV-Var
zurück. Bestehende Deployments (`.env`-basiert) funktionieren unverändert.

**Warum nicht Swarm Secrets:** Der Demo-Server läuft als Single-Node ohne Swarm-Mode. Migration
zu Swarm nur für Secret-Management wäre Over-Engineering.

**Warum nicht Vault/sops:** Externer Dienst widerspricht dem Self-Hosted-Prinzip von Vakt.

## Implementierung

### vakt-app (`backend/internal/config/config.go`)
```go
func readEnvOrFile(envKey, fileKey string) (string, error) {
    if f := os.Getenv(fileKey); f != "" {
        b, err := os.ReadFile(f)
        if err != nil {
            return "", fmt.Errorf("cannot read %s=%q: %w", fileKey, f, err)
        }
        return strings.TrimSpace(string(b)), nil
    }
    return os.Getenv(envKey), nil
}
```
Angewendet auf: `VAKT_SECRET_KEY` / `VAKT_SECRET_KEY_FILE` und `VAKT_DB_URL` / `VAKT_DB_URL_FILE`.

### vakt-server (`docker-compose.yml`)
```yaml
vakt-api:
  environment:
    VAKT_SECRET_KEY_FILE: /run/secrets/vakt_secret_key
    VAKT_DB_URL_FILE: /run/secrets/vakt_db_url
  volumes:
    - ./secrets/vakt_secret_key:/run/secrets/vakt_secret_key:ro
    - ./secrets/vakt_db_url:/run/secrets/vakt_db_url:ro
```
Secret-Dateien liegen in `./secrets/` (chmod 600, Verzeichnis chmod 700, in `.gitignore`).

### Datenbank (`postgres:16`)
`postgres:16` unterstützt `POSTGRES_PASSWORD_FILE` nativ. Kein Code-Änderung nötig.

### Form-Handler (`form-handler/main.go`)
`SMTP_PASS_FILE`-Unterstützung via analogem `readSecretFile()`-Helper.

## Konsequenzen

**Positiv:**
- `docker inspect` zeigt nur Dateipfade, keine Klartext-Secrets
- Backward-kompatibel: .env-basierte Setups unverändert lauffähig
- Kein externer Dienst erforderlich

**Einschränkung:**
- Secret-Dateien liegen auf derselben Maschine wie die DB → Root-Äquivalenz bleibt bestehen.
  Das ist ein akzeptiertes Restrisiko für Single-Node-Deployments: wer Root hat, hat alles.
  Die Maßnahme schützt gegen unprivilegierte Nutzer mit Docker-Socket-Zugriff (falls ein
  Container compromised wird) und gegen versehentliche Logging-Exposition.
- Secrets-Verzeichnis muss in Off-Site-Backup einbezogen werden (siehe ADR-0049 Follow-up).
