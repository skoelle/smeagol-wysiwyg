// Package server wires together the vault, watcher, search and render
// packages into the HTTP application. All routes here are internal to
// this application's own UI; there is no documented/public API contract
// (see SPEC.md section 6.3).
package server

import (
	"encoding/json"
	"fmt"
	"html/template"
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
		w.WriteHeader(http.StatusOK)
	} else {
		data.Exists = true
		htmlContent, rerr := render.ToHTML(content)
		if rerr != nil {
			http.Error(w, "Fehler beim Rendern: "+rerr.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, tree)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := search.Search(s.Vault.Root, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, results)
}

func (s *Server) handleGetRaw(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	content, err := s.Vault.ReadFile(reqPath)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
}

func (s *Server) handlePutRaw(w http.ResponseWriter, r *http.Request) {
	reqPath := r.PathValue("path")
	defer r.Body.Close()
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 8192)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	if err := s.Vault.WriteFileAtomic(reqPath, buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

const pageHTML = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} - smeagol-wysiwyg</title>
<link rel="stylesheet" href="/assets/style.css">
</head>
<body>
<header class="topbar">
  <button id="btn-overview" class="btn" aria-label="Overview">&#9776; Overview</button>
  <input id="search-input" class="search-input" type="search" placeholder="Suche...">
  <div class="spacer"></div>
  <span id="save-indicator" class="save-indicator" data-state="idle"></span>
  <button id="btn-edit" class="btn">Bearbeiten</button>
</header>

<nav id="overview-panel" class="overview-panel" hidden></nav>
<div id="search-panel" class="search-panel" hidden></div>

<main id="content" data-path="{{.PathForJS}}" data-exists="{{.Exists}}">
{{.ContentHTML}}
</main>

<div id="editor-mount" class="editor-mount" hidden></div>

<script src="/assets/main.js" defer></script>
</body>
</html>
`
