# SPEC.md - smeagol-wysiwyg (Personal WYSIWYG Wiki, Go)

## 1. Background and Reference

This project is inspired by [AustinWise/smeagol](https://github.com/AustinWise/smeagol) (smeagol.dev), a locally running wiki tool written in Rust with no authentication. The core philosophy is adopted here, but reimplemented in Go and extended with a real WYSIWYG editor.

Core principles carried over from the original:

- GitHub-compatible: Markdown as the format, directory structure remains readable and browsable on GitHub/Gitea.
- Simple and fast to install: a single native executable, no runtime dependencies.
- Runs locally, no multi-user support, no authentication needed (non-goal, same as the original).
- Not intended for public internet hosting.

Deliberate deviations from the original:

- Language: Go instead of Rust.
- No Git backend as storage engine; filesystem only. Git versioning is the user's responsibility, outside the tool.
- Editor: real WYSIWYG instead of plain Markdown editing.
- Save behavior: instant save instead of an explicit commit/save button.
- No image upload in version 1.

## 2. Name

**smeagol-wysiwyg**. Fits comfortably within GitHub's repository naming rules (max 100 characters); the name itself is only 15 characters.

## 3. Goal

A single Go binary that points at a vault path (a directory of Markdown files, including subdirectories) and serves them through a web interface in the browser. Designed to run directly on an LXC container: binary is copied via scp/rsync, no Docker deployment needed.

## 4. Non-Goals

- No multi-user support, no login, no access control.
- No built-in versioning system, no commit history within the tool.
- No public internet hosting.
- No plugin architecture in version 1.
- No image upload / drag-and-drop media management.
- No documented/public REST API. There are internal HTTP routes for the UI, but no API-first design.
- No upfront scan or index building on startup.

## 5. Functional Requirements

### 5.1 Vault and Filesystem

- Started via CLI argument with the path to the vault directory, e.g. `smeagol-wysiwyg /path/to/vault`. Without an argument, the current working directory is used.
- **No reading/indexing on startup.** Files are only read from disk on actual access. The vault is treated as externally mutable.
- Every page request reads the file live from disk; no cached state.
- Path traversal protection: all resolved paths must stay within the vault root.

### 5.2 Live Reload on External Changes

- The server watches the vault directory recursively via `fsnotify`.
- When the open file changes, the client is notified via Server-Sent Events and reloads automatically.
- On unsaved local changes: conflict warning instead of a silent reload.
- The overview tree view also updates live.

### 5.3 Start Page and Navigation

- `/` shows `README.md` in the vault root.
- Each subdirectory can have its own `README.md` as an index page.
- Overview button always accessible, including on mobile, shows the tree view of the entire vault.

### 5.4 WYSIWYG Display and Editor

- Visually rendered by default, no raw Markdown as the default view.
- Inline switch to edit mode, no split view.
- Editor: Milkdown, Markdown-native round-trip.
- Supported: headings (h1/h2/h3), bold/italic, lists, links, images (display only, no upload), code blocks, tables, blockquotes, inline code, horizontal rule, hard break.

### 5.5 Instant Save with Indicator

- No save button; debounce (800–1200ms) after typing stops.
- Indicator: "editing", "saving...", "saved" (with timestamp), error state.
- Atomic writes (temp file + rename).
- Own saves must not trigger a false conflict warning in the same tab.

### 5.6 Search

- Grep-style recursive scan, no persistent index.
- Searches file names/paths and content.
- Results: path, title, context snippet.
- Must work with umlauts, special characters, and deeply nested paths.

### 5.7 Mobile Usability

- Overview and Edit must remain accessible on narrow viewports.
- Responsive layout instead of disappearing buttons.

## 6. Technical Architecture

### 6.1 Language and Build

- Go, target output: single binary for linux/amd64.
- Frontend assets embedded via `embed.FS`.

### 6.2 Backend Components

- `net/http`, no heavy framework.
- Filesystem layer: on-demand reads, atomic writes, traversal protection.
- `fsnotify` watcher, SSE for live reload.
- Grep-style search without index.
- goldmark for server-side Markdown rendering.

### 6.3 Internal Routes (no public API contract)

| Method | Path | Purpose |
|---|---|---|
| GET | `/` | README.md in vault root |
| GET | `/page/{path}` | Rendered Markdown page |
| GET | `/api/tree` | Directory tree for overview |
| GET | `/api/search?q=` | Search results |
| GET | `/api/raw/{path}` | Raw content for the editor |
| PUT | `/api/raw/{path}` | Saves editor content |
| GET | `/api/events` | SSE for live reload |

### 6.4 Configuration

- `--host` (default `127.0.0.1`), `--port` (default `8000`).
- Positional argument: vault path (default current working directory).

## 7. Deployment Target

- LXC container, no Docker.
- Binary copied via scp/rsync; startup is outside this spec (separate step).
- Server-side configuration editor: vim.

## 8. Open Items

- Optional in-memory search index if needed.
- Transclusion feature from the original not adopted.
- Image upload currently excluded.
- Mermaid/PlantUML support as a future option.
- Detailed conflict UI for parallel external reloads not yet designed.
