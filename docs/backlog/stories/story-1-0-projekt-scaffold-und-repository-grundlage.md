# Story 1.0 — Projekt-Scaffold und Repository-Grundlage

## Name
Story 1.0 — Projekt-Scaffold und Repository-Grundlage

## Ziel
Das Repository hat eine vollständige, buildbare Grundstruktur – Go-Modul, Verzeichnisbaum, Build-Tooling, Deployment-Konfiguration und CI/CD-Workflows – sodass alle späteren Stories auf einem stabilen Fundament aufbauen.

## Eingangszustand
Das Repository enthält nur `docs/` und `.git/`.

## Ausgangszustand
`go build ./...` und `docker build .` laufen fehlerfrei durch. GitHub Actions führen CI, Security-Scans und Release-Workflows aus.

## Akzeptanzkriterien
* Verzeichnisstruktur:

```
cmd/caldo/main.go
internal/
  config/
  db/
  caldav/
  sync/
  handler/
  middleware/
  scheduler/
  crypto/
  migrations/
  parser/
  query/
  model/
web/
  static/
  templates/
docs/
  arch.md
  prd.md
  epics.md
Makefile
Dockerfile
docker-compose.yml
.github/
  workflows/
  dependabot.yml
.gitignore
```

* `go.mod` enthält ausschließlich die in arch.md Abschnitt 2.1 festgelegten Abhängigkeiten.
* `cmd/caldo/main.go` kompiliert ohne Fehler; `func main()` ist vorhanden und leer.
* `go build ./...` läuft fehlerfrei durch.
* `go vet ./...` erzeugt keine Ausgabe.
* `Makefile` enthält die Targets `build`, `dev`, `tailwind`, `templ`, `test`, `lint` und `docker-build`.
* `Dockerfile` ist mehrstufig: Builder-Stage kompiliert das Binary, Runtime-Stage enthält nur Binary und `web/static/`.
* `docker-compose.yml` enthält einen `caldo`-Service mit konfigurierbaren Umgebungsvariablen, einem benannten Volume für die SQLite-Datenbank und einem `healthcheck` gegen `GET /health`.
* `.gitignore` schließt kompilierte Binaries, `*.db`, `*.db-wal`, `*.db-shm`, `web/static/app.*.css`, `web/static/app.*.js` und `web/static/manifest.json` aus. Generierte `*_templ.go`-Dateien werden eingecheckt.

**GitHub Actions – CI-Workflow** (`.github/workflows/ci.yml`, Trigger: Push und Pull Request auf `main`):

* `go vet ./...` muss fehlerfrei laufen.
* `go test ./... -race` muss fehlerfrei laufen.
* `templ generate` darf keine uncommitteten Änderungen erzeugen (Diff-Check).
* Tailwind-Build darf keine uncommitteten Änderungen erzeugen.
* CI schlägt fehl, wenn generierte Dateien nicht aktuell eingecheckt sind.

**GitHub Actions – Security-Workflow** (`.github/workflows/security.yml`, Trigger: Push auf `main`, wöchentlicher Cron):

* `govulncheck ./...` prüft bekannte Go-Schwachstellen in Abhängigkeiten.
* `gosec ./...` prüft sicherheitsrelevante Code-Muster.
* Trivy scannt das fertig gebaute Docker-Image auf bekannte CVEs (HIGH und CRITICAL).
* Scan-Ergebnisse werden als SARIF in den GitHub Security-Tab hochgeladen.
* Der Security-Workflow verhindert keinen Merge; er ist informativ.

**GitHub Actions – Release-Workflow** (`.github/workflows/release.yml`, Trigger: Git-Tag `v*`):

* Das Binary wird für `linux/amd64` und `linux/arm64` gebaut.
* Das Docker-Image wird für beide Architekturen gebaut und in GitHub Container Registry (`ghcr.io`) gepusht.
* Image-Tags: exakter Versions-Tag sowie `latest`.
* Ein GitHub Release wird automatisch mit dem Image-Digest und einer generierten Changelog-Sektion aus Commits seit dem letzten Tag erstellt.

**Dependabot** (`.github/dependabot.yml`):

* Go-Module werden wöchentlich geprüft.
* GitHub Actions werden wöchentlich geprüft.
* Automatische Pull Requests werden als Draft erstellt.

---
