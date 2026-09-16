// Package server wires together the vault, watcher, search and render
// packages into the HTTP application. All routes here are internal to
// this application's own UI; there is no documented/public API contract
// (see SPEC.md section 6.3).
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/USERNAME/smeagol-wysiwyg/internal/render"
	"github.com/USERNAME/smeagol-wysiwyg/internal/search"
	"github.com/USERNAME/smeagol-wysiwyg/internal/vault"
	"github.com/USERNAME/smeagol-wysiwyg/internal/watcher"
)

const maxBodySize = 10 << 20 // 10 MB

type Server struct {
	Vault   *vault.Vault
	Watcher *watcher.Watcher
	Assets  fs.FS
	mux     *http.ServeMux
}

func New(v *vault.Vault, w *watcher.Watcher, assets fs.FS) *Server {
	s := &Server{Vault: v, Watcher: w, Assets: assets, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(s.Assets)))

	s.mux.HandleFunc("GET /{$}", s.handlePage)
	s.mux.HandleFunc("GET /page/{path...}", s.handlePage)

	s.mux.HandleFunc("GET /api/tree", s.handleTree)
	s.mux.HandleFunc("GET /api/search", s.handleSearch)
	s.mux.HandleFunc("GET /api/raw/{path...}", s.handleGetRaw)
	s.mux.HandleFunc("PUT /api/raw/{path...}", s.handlePutRaw)
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
}

var pageTmpl = template.Must(template.New("page").Parse(pageHTML))

type pageData struct {
	Title       string
	PathForJS   string
	ContentHTML template.HTML
	Exists      bool
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	if reqPath == "" {
		reqPath = "README.md"
	}
	if !strings.HasSuffix(strings.ToLower(reqPath), ".md") {
		reqPath = strings.TrimSuffix(reqPath, "/") + "/README.md"
	}

	data := pageData{PathForJS: reqPath}

	content, err := s.Vault.ReadFile(reqPath)
	if err != nil {
		data.Exists = false
		data.Title = "Nicht gefunden"
		data.ContentHTML = template.HTML(`<p class="empty-state">Diese Seite existiert noch nicht. Wechsle in den Bearbeitungsmodus, um sie anzulegen.</p>`)
		w.WriteHeader(http.StatusNotFound)
	} else {
		data.Exists = true
		htmlContent, rerr := render.ToHTML(content)
		if rerr != nil {
			log.Printf("server: render error for %s: %v", reqPath, rerr)
			http.Error(w, "Fehler beim Rendern", http.StatusInternalServerError)
			return
		}
		data.ContentHTML = template.HTML(htmlContent)
		data.Title = filepath.Base(reqPath)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTmpl.Execute(w, data); err != nil {
		log.Printf("server: template error: %v", err)
	}
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	tree, err := s.Vault.Tree()
	if err != nil {
		log.Printf("server: tree error: %v", err)
		http.Error(w, "failed to build tree", http.StatusInternalServerError)
		return
	}
	writeJSON(w, tree)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := search.Search(s.Vault.Root, q)
	if err != nil {
		log.Printf("server: search error: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

func (s *Server) handleGetRaw(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	content, err := s.Vault.ReadFile(reqPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func (s *Server) handlePutRaw(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, maxBodySize+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if int64(len(buf)) > maxBodySize {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	if err := s.Vault.WriteFileAtomic(reqPath, buf); err != nil {
		if errors.Is(err, vault.ErrOutsideVault) {
			http.Error(w, "invalid path", http.StatusBadRequest)
		} else {
			log.Printf("server: write error for %s: %v", reqPath, err)
			http.Error(w, "write failed", http.StatusInternalServerError)
		}
		return
	}
	writeJSON(w, map[string]any{
		"ok":   true,
		"path": reqPath,
		"time": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch, unsubscribe := s.Watcher.Subscribe()
	defer unsubscribe()

	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := json.Marshal(ev)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	// SetEscapeHTML(false) is intentional: JSON is consumed by fetch(),
	// not embedded in HTML. Content-Type: application/json prevents
	// browser HTML interpretation. Keeps Markdown snippets readable in DevTools.
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

const pageHTML = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - smeagol-wysiwyg</title>
<link rel="icon" type="image/svg+xml" href="/assets/favicon.svg">
<link rel="stylesheet" href="/assets/style.css">
</head>
<body>
<header class="topbar">
  <button id="btn-overview" class="btn" aria-label="Overview">&#9776; Overview</button>
  <button id="btn-toc" class="btn" aria-label="Table of Contents">&#9776; TOC</button>
  <input id="search-input" class="search-input" type="search" placeholder="Suche...">
  <div class="spacer"></div>
  <span id="save-indicator" class="save-indicator" data-state="idle"></span>
  <button id="btn-edit" class="btn">Bearbeiten</button>
</header>

<nav id="overview-panel" class="overview-panel" hidden></nav>
<div id="toc-panel" class="toc-panel" hidden></div>
<div id="search-panel" class="search-panel" hidden></div>

<main id="content" data-path="{{.PathForJS}}" data-exists="{{.Exists}}">
{{.ContentHTML}}
</main>

<div id="editor-toolbar" class="editor-toolbar" hidden>
  <div class="toolbar-group">
    <button data-cmd="h1" title="Ueberschrift 1">=</button>
    <button data-cmd="h2" title="Ueberschrift 2">==</button>
    <button data-cmd="h3" title="Ueberschrift 3">===</button>
    <button data-cmd="paragraph" title="Absatz">P</button>
  </div>
  <div class="toolbar-sep"></div>
  <div class="toolbar-group">
    <button data-cmd="bold" title="Fett (Ctrl+B)"><b>B</b></button>
    <button data-cmd="italic" title="Kursiv (Ctrl+I)"><i>I</i></button>
    <button data-cmd="inlinecode" title="Inline-Code (Ctrl+E)">&lt;/&gt;</button>
  </div>
  <div class="toolbar-sep"></div>
  <div class="toolbar-group">
    <button data-cmd="bulletlist" title="Aufzaehlung">* List</button>
    <button data-cmd="orderedlist" title="Nummeriert">1. List</button>
    <button data-cmd="blockquote" title="Zitat (Ctrl+Shift+9)">&ldquo;</button>
    <button data-cmd="codeblock" title="Codeblock (Ctrl+Shift+K)">{ }</button>
  </div>
  <div class="toolbar-sep"></div>
  <div class="toolbar-group">
    <button data-cmd="hr" title="Trennlinie">---</button>
    <button data-cmd="hardbreak" title="Zeilenumbruch">Shift+Enter</button>
  </div>
</div>

<div id="editor-mount" class="editor-mount" hidden></div>

<script src="/assets/main.js" defer></script>
</body>
</html>
`
