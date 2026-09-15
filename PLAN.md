# PLAN.md - Umsetzungsplan smeagol-wysiwyg

Dieser Plan beschreibt die Reihenfolge der Umsetzung basierend auf SPEC.md. Der Plan endet mit einem fertigen, hochladbaren Repository-ZIP. Das Ausspielen auf den Ziel-LXC ist bewusst nicht Teil dieses Plans.

## Editor-Entscheidung

WYSIWYG-Basis: **Milkdown**. Markdown-natives, headless, plugin-getriebenes Framework, passt besser zu unserem Anspruch, dass die Datei auf der Platte immer valides Markdown bleibt, ohne zusaetzliche Serialisierungs-Pakete wie bei TipTap.

## Umsetzungshinweis (Abweichung von der urspruenglichen Editor-Idee)

Um das Repository frei von einer Node/npm-Build-Toolchain zu halten, wird Milkdown nicht ueber einen Vite-Build eingebunden, sondern im Browser per dynamischem ESM-Import von einem CDN geladen, ausschliesslich beim tatsaechlichen Wechsel in den Bearbeitungsmodus. `go build` allein reicht damit aus, um das fertige Binary zu erzeugen. Diese Entscheidung ist im README und im Kopfkommentar von `web/dist/main.js` dokumentiert, inklusive Hinweis, wie man stattdessen lokal vendoren kann, falls komplette Offline-Faehigkeit ab dem ersten Aufruf gewuenscht ist.

## Phase 0: Projekt-Grundgeruest - erledigt

Go-Modul `github.com/USERNAME/smeagol-wysiwyg` angelegt. Ordnerstruktur: `main.go`, `embed.go`, `internal/vault`, `internal/watcher`, `internal/search`, `internal/render`, `internal/server`, `web/dist`, `testdata/vault`.

## Phase 1: Vault-Layer - erledigt

`internal/vault/vault.go`: Path-Traversal-sicheres Aufloesen, Lesen on-demand, atomares Schreiben (Temp-Datei + Rename), rekursiver Verzeichnisbaum. `internal/vault/vault_test.go`: Tests fuer Traversal-Schutz, Lesen, Schreiben, Baum-Aufbau, Existenz-Check.

## Phase 2: Basis-HTTP-Server und Rendering - erledigt

`internal/render/render.go`: Markdown-zu-HTML via goldmark (GFM). `internal/server/server.go`: Routen fuer `/`, `/page/{pfad}`, `/api/tree`, serverseitiges HTML-Template als Seiten-Shell.

## Phase 3: fsnotify-Watcher und Live-Reload - erledigt

`internal/watcher/watcher.go`: rekursives Watchen inkl. zur Laufzeit neu angelegter Unterverzeichnisse, Broadcast an Subscriber-Channels. `/api/events` als SSE-Endpunkt. Client-seitig: Reload bei Fremdaenderung, Konflikt-Banner statt stillem Reload bei aktiver Bearbeitung, Selbst-Saves werden per `ignoreNextReloadFor` nicht als Konflikt gewertet.

## Phase 4: Suche - erledigt

`internal/search/search.go`: rekursiver `filepath.WalkDir`-Scan, Zeilen-Matching, Treffer auch im Dateinamen/Pfad. `internal/search/search_test.go`: Tests mit Umlauten, Gross-/Kleinschreibung, tief verschachtelten Pfaden, leerer Suche. `/api/search?q=` liefert JSON-Ergebnisse.

## Phase 5: Frontend-Grundgeruest und Milkdown-Integration - erledigt

`web/dist/main.js`: Overview-Baum, Suchfeld mit Live-Ergebnissen, Umschalten in den Editier-Modus, Milkdown-Initialisierung mit `commonmark`-Preset und `listener`-Plugin. `web/dist/style.css`: dunkles, aufgeraeumtes Layout, responsive fuer Mobile.

## Phase 6: Instant-Save mit Indikator - erledigt

Debounce (1000ms) im Frontend, Speicher-Indikator mit den Zustaenden "wird bearbeitet", "speichert...", "gespeichert" (mit Uhrzeit), "Fehler". `PUT /api/raw/{pfad}` im Server, atomar ueber den Vault-Layer.

## Phase 7: Mobile-Layout - erledigt

Responsive Anpassungen in `style.css`: Overview-Button, Suchfeld und Edit-Button bleiben auf schmalen Viewports erreichbar (Flex-Wrap in der Topbar statt Verschwinden von Buttons).

## Phase 8: embed.FS und Single-Binary-Build - erledigt

`embed.go` bindet `web/dist` per `//go:embed` ein. `main.go`: CLI-Parsing (`--host`, `--port`, Positionsargument fuer den Vault-Pfad), Server-Start. Kein separater Frontend-Build-Schritt notwendig, siehe Umsetzungshinweis oben.

## Phase 9: Repository-Finalisierung und ZIP - erledigt

README.md mit Bau-, Test- und Ausfuehrungsanleitung. `.gitignore` fuer Go-Build-Artefakte. MIT-Lizenz ergaenzt. Beispiel-Vault unter `testdata/vault/` (inkl. Unterordner und Umlauten) fuer manuelle Tests. Gesamtes Repository als ZIP verpackt, bereit zum Hochladen als neues GitHub-Repository `smeagol-wysiwyg`.

**Dies ist der Abschlusspunkt des Plans.** Ausspielen auf den Ziel-LXC erfolgt separat, nach Pruefung des Repo-Inhalts.

## Bekannte Einschraenkungen dieser Umsetzung

- Der Code wurde in einer Sandbox ohne Go-Compiler und ohne Internetzugang geschrieben und konnte daher nicht selbst kompiliert oder mit `go test` verifiziert werden. Vor dem produktiven Einsatz bitte einmal lokal `go build ./...` und `go test ./...` laufen lassen.
- Der Modul-Pfad in `go.mod` und den Import-Pfaden lautet aktuell `github.com/USERNAME/smeagol-wysiwyg` als Platzhalter und muss vor dem ersten Build auf den tatsaechlichen GitHub-Benutzernamen angepasst werden (Suchen/Ersetzen von `USERNAME`).
- Die Milkdown-Integration in `web/dist/main.js` folgt der dokumentierten Milkdown-7-API, wurde aber nicht gegen eine echte Milkdown-Version im Browser getestet; bei API-Abweichungen ggf. gegen die aktuelle Milkdown-Doku nachjustieren.
