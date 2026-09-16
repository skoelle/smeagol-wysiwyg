# 🧙 smeagol-wysiwyg

A personal wiki inspired by [Smeagol](https://smeagol.dev) (AustinWise/smeagol), but with a real WYSIWYG editor instead of raw Markdown editing. Runs as a single Go binary, points at a directory of Markdown files ("vault") and serves them through a web interface.

Details on the design goals and architecture are in [SPEC.md](SPEC.md).

## ✨ Features

- 📦 Single Go binary, no database, no runtime dependencies.
- 📂 Points at any vault path with Markdown files, including arbitrarily deep subdirectories.
- 📄 Starts by showing `README.md` in the vault root; each subdirectory can have its own `README.md` as an index page.
- 🗂️ Overview button shows a tree view of the entire vault, accessible even on mobile viewports.
- 🔍 Full-text search across all Markdown files (content and path) via recursive filesystem scan with no persistent index.
- ✏️ Real WYSIWYG editor ([Milkdown](https://milkdown.dev)) instead of plain Markdown editing; content is saved as valid Markdown.
- 💾 Instant save: no save button -- changes are saved automatically after a short typing pause, with visible status ("editing" / "saving..." / "saved" / "error").
- 🔄 Live reload: when a file changes externally (e.g. via `vim` on the server), the open page updates automatically with a conflict warning instead of silently overwriting if you are currently editing.
- ⚡ No upfront scan on startup: the vault is treated as externally mutable at all times.
- 🛠️ Formatting toolbar with headings (h1/h2/h3), bold, italic, inline code, lists, blockquote, code block, horizontal rule and hard break.
- ☀️ Light mode UI.

## 🔨 Build

Requires Go 1.23 or later.

Before the first build, replace the `USERNAME` placeholder in the module path (`go.mod`, `main.go`, `internal/server/server.go`) with your actual GitHub username:

```bash
grep -rl "USERNAME" . | xargs sed -i 's/USERNAME/YOUR-GITHUB-NAME/g'
```

Then build:

```bash
go build -o smeagol-wysiwyg .
```

For a specific target (e.g. linux/amd64):

```bash
GOOS=linux GOARCH=amd64 go build -o smeagol-wysiwyg .
```

The frontend assets in `web/dist/` (HTML/CSS/JS) are already in the repository and embedded into the binary via `//go:embed web/dist` (see `embed.go`). No separate npm/Node build step is required -- `go build` is all you need.

The WYSIWYG editor (Milkdown) is vendored into `web/dist/vendor/milkdown.js` and works fully offline. No internet connection is needed.

## 🧪 Tests

```bash
go test ./...
```

Covers: path traversal protection in the vault layer, atomic writes, directory tree building and search (including umlauts and deeply nested paths).

## 🚀 Usage

```bash
./smeagol-wysiwyg --host 127.0.0.1 --port 8000 /path/to/vault
```

Without a path argument the current working directory is used as the vault. Then open `http://127.0.0.1:8000` in your browser. A sample vault to try out is in `testdata/vault/`:

```bash
./smeagol-wysiwyg testdata/vault
```

## 📁 Project Structure

```
.
|-- main.go                  CLI entry point
|-- embed.go                 Embeds web/dist into the binary via go:embed
|-- internal/
|   |-- vault/               Filesystem layer (read/write/tree)
|   |-- watcher/             fsnotify integration for live reload
|   |-- search/              Grep-style full-text search
|   |-- render/              Markdown-to-HTML rendering (goldmark)
|   `-- server/              HTTP routes and page template
|-- web/dist/                Static frontend assets (HTML/CSS/JS)
|-- web/dist/vendor/         Vendored Milkdown editor bundle
|-- testdata/vault/          Example vault for manual testing
|-- SPEC.md                  Full specification
```

## 🔒 Security

smeagol is designed for local-only use or behind a reverse proxy with authentication (e.g. Authelia). The following are intentionally not implemented:

- **No XSS sanitization**: Markdown can contain raw HTML/JS (`html.WithUnsafe()`). This is by design -- only trusted users have vault access.
- **No CSRF protection**: The PUT endpoint has no CSRF token. Not needed for localhost-only or auth-proxy deployments.
- **No security headers**: CSP, X-Frame-Options, etc. are omitted. Not relevant for the intended deployment model.

If you need to expose smeagol to untrusted users, put it behind an authentication proxy and restrict network access.

## 🚫 Non-Goals

See SPEC.md section 4: no multi-user support, no login, no public internet hosting, no image upload, no Git backend as storage engine (versioning the vault folder is left to the user, e.g. via separate `git add`/`git commit`).

## 📜 License

MIT, see [LICENSE](LICENSE).
