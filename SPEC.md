# SPEC.md - smeagol-wysiwyg (Personal WYSIWYG Wiki, Go)

## 1. Hintergrund und Referenz

Dieses Projekt orientiert sich an [AustinWise/smeagol](https://github.com/AustinWise/smeagol) (smeagol.dev), einem in Rust geschriebenen, lokal laufenden Wiki-Tool ohne Authentifizierung. Die dortige Grundphilosophie wird uebernommen, aber in Go neu umgesetzt und um einen echten WYSIWYG-Editor erweitert.

Kernprinzipien aus der Originalspezifikation, die uebernommen werden:

- Kompatibel mit GitHub: Markdown als Format, Verzeichnisstruktur bleibt lesbar und browsbar auf GitHub/Gitea.
- Einfach und schnell zu installieren: ein einziges natives Executable, keine Laufzeitabhaengigkeiten.
- Laeuft lokal, keine Mehrbenutzerfaehigkeit, keine Authentifizierung notwendig (Non-Goal wie im Original).
- Nicht fuer den oeffentlichen Internet-Einsatz gedacht.

Bewusste Abweichungen vom Original:

- Sprache: Go statt Rust.
- Kein Git-Backend als Speicher-Engine, ausschliesslich das Dateisystem. Git-Versionierung bleibt Sache des Nutzers, ausserhalb des Tools.
- Editor: echtes WYSIWYG statt reinem Markdown-Edit.
- Speicherverhalten: Instant-Save statt explizitem Commit/Save-Button.
- Kein Bild-Upload in Version 1.

## 2. Name

**smeagol-wysiwyg**. Passt problemlos in GitHubs Namensregeln fuer Repositories (max. 100 Zeichen), der Name selbst hat nur 15 Zeichen.

## 3. Zielsetzung

Ein einzelnes Go-Binary, das auf einen Vault-Pfad (Verzeichnis mit Markdown-Dateien, inklusive Unterverzeichnissen) zeigt und ueber eine Web-Oberflaeche im Browser bedient wird. Betrieb direkt auf einem LXC-Container: Binary wird per scp/rsync auf den LXC kopiert, kein Docker-Deployment notwendig.

## 4. Nicht-Ziele

- Keine Mehrbenutzerfaehigkeit, kein Login, keine Rechteverwaltung.
- Kein eigenes Versionierungssystem, keine Commit-Historie im Tool selbst.
- Kein oeffentliches Internet-Hosting.
- Keine Plugin-Architektur in Version 1.
- Kein Bild-Upload / Drag-and-Drop-Medienverwaltung.
- Keine dokumentierte/oeffentliche REST-API. Es gibt intern HTTP-Routen fuers UI, aber keine API-first-Auslegung.
- Kein Vorab-Scan/Indexbau beim Start.

## 5. Funktionale Anforderungen

### 5.1 Vault und Dateisystem

- Start per CLI-Argument mit Pfad zum Vault-Verzeichnis, z. B. `smeagol-wysiwyg /pfad/zum/vault`. Ohne Argument wird das aktuelle Arbeitsverzeichnis verwendet.
- **Kein Einlesen/Indexieren beim Start.** Dateien werden erst beim tatsaechlichen Zugriff von der Platte gelesen. Der Vault gilt als extern veraenderlich.
- Jede Seiten-Anfrage liest die Datei live von der Platte, kein zwischengespeicherter Zustand.
- Schutz gegen Path-Traversal: alle aufgeloesten Pfade muessen innerhalb des Vault-Root liegen.

### 5.2 Live-Reload bei externen Aenderungen

- Der Server ueberwacht das Vault-Verzeichnis rekursiv per `fsnotify`.
- Aendert sich die geoeffnete Datei, wird der Client per Server-Sent-Events benachrichtigt und laedt automatisch neu.
- Bei ungespeicherten lokalen Aenderungen: Konflikt-Hinweis statt stillem Ueberschreiben.
- Die Overview-Baumansicht aktualisiert sich ebenfalls live.

### 5.3 Startseite und Navigation

- `/` zeigt `README.md` im Vault-Root.
- Jedes Unterverzeichnis kann eine eigene `README.md` als Index haben.
- Overview-Button jederzeit erreichbar, auch mobil, zeigt Baumansicht des gesamten Vaults.

### 5.4 WYSIWYG-Darstellung und -Editor

- Standardmaessig visuell gerendert, kein rohes Markdown als Default.
- Inline-Umschalten in Editier-Modus, kein Split-View.
- Editor-Basis: Milkdown, Markdown-natives Round-Trip.
- Unterstuetzt: Ueberschriften, Fett/Kursiv, Listen, Links, Bilder (Anzeige, kein Upload), Code-Bloecke, Tabellen, Blockquotes.

### 5.5 Instant-Save mit Indikator

- Kein Speichern-Button, Debounce (800-1200ms) nach Tippstopp.
- Indikator: "wird bearbeitet", "speichert...", "gespeichert" (mit Zeitstempel), Fehlerzustand.
- Atomares Schreiben (Temp-Datei + Rename).
- Eigene Saves duerfen keinen falschen Konflikt-Hinweis im selben Tab ausloesen.

### 5.6 Suche

- Grep-artiger rekursiver Scan, kein persistenter Index.
- Durchsucht Dateinamen/Pfade und Inhalt.
- Ergebnis: Pfad, Titel, Kontext-Snippet.
- Muss mit Umlauten, Sonderzeichen und tiefen Pfaden funktionieren.

### 5.7 Mobile-Bedienbarkeit

- Overview und Edit muessen auf schmalen Viewports erreichbar bleiben.
- Responsive Layout statt verschwindender Buttons.

## 6. Technische Architektur

### 6.1 Sprache und Build

- Go, Zielausgabe: einzelnes Binary fuer linux/amd64.
- Frontend-Assets via `embed.FS` ins Binary eingebettet.

### 6.2 Backend-Komponenten

- `net/http`, kein schweres Framework.
- Dateisystem-Layer: on-demand lesen, atomar schreiben, Traversal-Schutz.
- `fsnotify`-Watcher, SSE fuer Live-Reload.
- Grep-artige Suche ohne Index.
- goldmark fuer serverseitiges Rendering.

### 6.3 Interne Routen (kein oeffentlicher API-Vertrag)

| Methode | Pfad | Zweck |
|---|---|---|
| GET | `/` | README.md im Vault-Root |
| GET | `/page/{pfad}` | Gerenderte Markdown-Datei |
| GET | `/api/tree` | Verzeichnisbaum fuer Overview |
| GET | `/api/search?q=` | Suchergebnisse |
| GET | `/api/raw/{pfad}` | Rohinhalt fuer den Editor |
| PUT | `/api/raw/{pfad}` | Speichert Editor-Inhalt |
| GET | `/api/events` | SSE fuer Live-Reload |

### 6.4 Konfiguration

- `--host` (Default `127.0.0.1`), `--port` (Default `8000`).
- Positionsargument: Vault-Pfad (Default aktuelles Arbeitsverzeichnis).

## 7. Deployment-Zielbild

- LXC-Container, kein Docker.
- Binary per scp/rsync kopieren, Start ausserhalb dieses Plans (separater Schritt).
- Editor fuer serverseitige Konfiguration: vim.

## 8. Offene Punkte

- Optionaler In-Memory-Suchindex bei Bedarf.
- Transclusion-Feature aus dem Original nicht uebernommen.
- Bild-Upload aktuell ausgeklammert.
- Mermaid/PlantUML-Unterstuetzung als spaetere Option.
- Detaillierte Konflikt-UI bei parallelem externen Reload noch nicht ausgearbeitet.
