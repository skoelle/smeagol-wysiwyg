# smeagol-wysiwyg

Ein persoenliches Wiki nach dem Vorbild von [Smeagol](https://smeagol.dev)
(AustinWise/smeagol), aber mit echtem WYSIWYG-Editor statt reinem
Markdown-Editing. Laeuft als einzelnes Go-Binary, zeigt auf ein
Verzeichnis mit Markdown-Dateien ("Vault") und dient dieses ueber eine
Weboberflaeche aus.

Details zur Zielsetzung stehen in [SPEC.md](SPEC.md), der Umsetzungsplan
in [PLAN.md](PLAN.md).

## Eigenschaften

- Einzelnes Go-Binary, keine Datenbank, keine Laufzeitabhaengigkeiten.
- Zeigt auf einen beliebigen Vault-Pfad mit Markdown-Dateien, inklusive
  beliebig tiefer Unterverzeichnisse.
- Start ruft `README.md` im Vault-Root auf, jedes Unterverzeichnis kann
  eine eigene `README.md` als Index haben.
- Overview-Button zeigt eine Baumansicht des gesamten Vaults, auch auf
  mobilen Viewports erreichbar.
- Volltextsuche ueber alle Markdown-Dateien (Dateiinhalt und Pfad), per
  rekursivem Filesystem-Scan ohne persistenten Index.
- Echter WYSIWYG-Editor ([Milkdown](https://milkdown.dev)) statt reinem
  Markdown-Text-Editing, Inhalte bleiben beim Speichern valides Markdown.
- Instant-Save: kein Speichern-Button, Aenderungen werden nach kurzer
  Tippstopp-Pause automatisch gespeichert, mit sichtbarem Status
  ("wird bearbeitet" / "speichert..." / "gespeichert" / "Fehler").
- Live-Reload: aendert sich eine Datei extern (z. B. per `vim` direkt auf
  dem Server), aktualisiert sich die geoeffnete Seite automatisch, mit
  Konflikt-Hinweis statt stillem Ueberschreiben, falls gerade selbst
  bearbeitet wird.
- Kein Vorab-Scan beim Start: der Vault gilt als jederzeit extern
  veraenderlich.

## Bauen

Voraussetzung: Go 1.22 oder neuer.

Vor dem ersten Build den Platzhalter `USERNAME` im Modul-Pfad ersetzen
(in `go.mod` und in den Import-Zeilen von `main.go` und
`internal/server/server.go`) durch den tatsaechlichen GitHub-Benutzernamen:

```bash
grep -rl "USERNAME" . | xargs sed -i 's/USERNAME/DEIN-GITHUB-NAME/g'
```

Dann bauen:

```bash
go build -o smeagol-wysiwyg .
```

Fuer das Zielsystem (linux/amd64) explizit cross-compilen:

```bash
GOOS=linux GOARCH=amd64 go build -o smeagol-wysiwyg .
```

Die Frontend-Assets in `web/dist/` (HTML/CSS/JS) sind bereits im
Repository enthalten und werden ueber `//go:embed web/dist` direkt ins
Binary eingebettet (siehe `embed.go`). Es ist **kein** separater
npm/Node-Build-Schritt notwendig, `go build` reicht aus.

Der WYSIWYG-Editor selbst (Milkdown) wird im Browser erst beim Wechsel in
den Bearbeitungsmodus per dynamischem ESM-Import von einem CDN geladen
(siehe Kommentar am Kopf von `web/dist/main.js`). Das haelt dieses Repo
frei von einer Node-Toolchain, bedeutet aber, dass fuer den ersten Aufruf
des Editors (nicht fuer das reine Lesen von Seiten) einmalig eine
Internetverbindung im Browser noetig ist; danach greift der
Browser-Cache. Wer eine komplett offline-faehige Variante ab dem ersten
Aufruf moechte, kann die Milkdown-ESM-Bundles lokal vendoren und die
`MILKDOWN_CDN_*`-Konstanten am Kopf von `web/dist/main.js` auf lokale
Pfade unter `web/dist/` umstellen.

## Tests

```bash
go test ./...
```

Testet unter anderem: Path-Traversal-Schutz im Vault-Layer, atomares
Schreiben, Verzeichnisbaum-Aufbau und die Suche (inklusive Umlauten und
tief verschachtelten Pfaden).

> Hinweis: Dieser Code wurde ohne lokalen Go-Compiler geschrieben und vor
> der Auslieferung nicht selbst mit `go build`/`go test` verifiziert.
> Bitte vor dem produktiven Einsatz einmal lokal durchlaufen lassen, siehe
> PLAN.md, Abschnitt "Bekannte Einschraenkungen".

## Ausfuehren

```bash
./smeagol-wysiwyg --host 127.0.0.1 --port 8000 /pfad/zum/vault
```

Ohne Pfad-Argument wird das aktuelle Arbeitsverzeichnis als Vault
verwendet. Danach im Browser `http://127.0.0.1:8000` oeffnen. Ein
Beispiel-Vault zum Ausprobieren liegt unter `testdata/vault/`:

```bash
./smeagol-wysiwyg testdata/vault
```

## Projektstruktur

```
.
|-- main.go                  CLI-Einstiegspunkt
|-- embed.go                 Bindet web/dist per go:embed ins Binary ein
|-- internal/
|   |-- vault/                Dateisystem-Layer (Lesen/Schreiben/Baum)
|   |-- watcher/               fsnotify-Integration fuer Live-Reload
|   |-- search/                Grep-artige Volltextsuche
|   |-- render/                Markdown-zu-HTML-Rendering (goldmark)
|   `-- server/                HTTP-Routen und Seiten-Template
|-- web/dist/                 Statische Frontend-Assets (HTML/CSS/JS)
|-- testdata/vault/            Beispiel-Vault fuer manuelle Tests
|-- SPEC.md                    Vollstaendige Spezifikation
`-- PLAN.md                    Umsetzungsplan (dieses Repo ist dessen Ergebnis)
```

## Nicht-Ziele

Siehe SPEC.md Abschnitt 4: keine Mehrbenutzerfaehigkeit, kein Login, kein
oeffentliches Internet-Hosting, kein Bild-Upload, kein Git-Backend als
Speicher-Engine (Versionierung des Vault-Ordners bleibt dem Nutzer selbst
ueberlassen, z. B. per separatem `git add`/`git commit`).

## Lizenz

MIT, siehe [LICENSE](LICENSE).
