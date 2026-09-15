// Package server wires together the vault, watcher, search and render
// packages into the HTTP application. All routes here are internal to
// this application's own UI; there is no documented/public API contract
// (see SPEC.md section 6.3).
package server

import (
\t"encoding/json"
\t"fmt"
\t"html/template"
\t"io/fs"
\t"log"
\t"net/http"
\t"path/filepath"
\t"strings"
\t"time"

\t"github.com/USERNAME/smeagol-wysiwyg/internal/render"
\t"github.com/USERNAME/smeagol-wysiwyg/internal/search"
\t"github.com/USERNAME/smeagol-wysiwyg/internal/vault"
\t"github.com/USERNAME/smeagol-wysiwyg/internal/watcher"
)

type Server struct {
\tVault   *vault.Vault
\tWatcher *watcher.Watcher
\tAssets  fs.FS
\tmux     *http.ServeMux
}

func New(v *vault.Vault, w *watcher.Watcher, assets fs.FS) *Server {
\ts := &Server{Vault: v, Watcher: w, Assets: assets, mux: http.NewServeMux()}
\ts.routes()
\treturn s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
\ts.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
\ts.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(s.Assets)))

\ts.mux.HandleFunc("GET /{$}", s.handlePage)
\ts.mux.HandleFunc("GET /page/{path...}", s.handlePage)

\ts.mux.HandleFunc("GET /api/tree", s.handleTree)
\ts.mux.HandleFunc("GET /api/search", s.handleSearch)
\ts.mux.HandleFunc("GET /api/raw/{path...}", s.handleGetRaw)
\ts.mux.HandleFunc("PUT /api/raw/{path...}", s.handlePutRaw)
\ts.mux.HandleFunc("GET /api/events", s.handleEvents)
}

var pageTmpl = template.Must(template.New("page").Parse(pageHTML))

type pageData struct {
\tTitle       string
\tPathForJS   string
\tContentHTML template.HTML
\tExists      bool
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
\treqPath := r.PathValue("path")
\tif reqPath == "" {
\t\treqPath = "README.md"
\t}
\tif !strings.HasSuffix(strings.ToLower(reqPath), ".md") {
\t\treqPath = strings.TrimSuffix(reqPath, "/") + "/README.md"
\t}

\tdata := pageData{PathForJS: reqPath}

\tcontent, err := s.Vault.ReadFile(reqPath)
\tif err != nil {
\t\tdata.Exists = false
\t\tdata.Title = "Nicht gefunden"
\t\tdata.ContentHTML = template.HTML(`<p class="empty-state">Diese Seite existiert noch nicht. Wechsle in den Bearbeitungsmodus, um sie anzulegen.</p>`)
\t\tw.WriteHeader(http.StatusOK)
\t} else {
\t\tdata.Exists = true
\t\thtmlContent, rerr := render.ToHTML(content)
\t\tif rerr != nil {
\t\t\thttp.Error(w, "Fehler beim Rendern: "+rerr.Error(), http.StatusInternalServerError)
\t\t\treturn
\t\t}
\t\tdata.ContentHTML = template.HTML(htmlContent)
\t\tdata.Title = filepath.Base(reqPath)
\t}

\tw.Header().Set("Content-Type", "text/html; charset=utf-8")
\tif err := pageTmpl.Execute(w, data); err != nil {
\t\tlog.Printf("server: template error: %v", err)
\t}
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
\ttree, err := s.Vault.Tree()
\tif err != nil {
\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)
\t\treturn
\t}
\twriteJSON(w, tree)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
\tq := r.URL.Query().Get("q")
\tresults, err := search.Search(s.Vault.Root, q)
\tif err != nil {
\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)
\t\treturn
\t}
\twriteJSON(w, results)
}

func (s *Server) handleGetRaw(w http.ResponseWriter, r *http.Request) {
\treqPath := r.PathValue("path")
\tcontent, err := s.Vault.ReadFile(reqPath)
\tif err != nil {
\t\tw.Header().Set("Content-Type", "text/plain; charset=utf-8")
\t\tw.WriteHeader(http.StatusOK)
\t\treturn
\t}
\tw.Header().Set("Content-Type", "text/plain; charset=utf-8")
\tw.Write(content)
}

func (s *Server) handlePutRaw(w http.ResponseWriter, r *http.Request) {
\treqPath := r.PathValue("path")
\tdefer r.Body.Close()
\tbuf := make([]byte, 0, 8192)
\ttmp := make([]byte, 8192)
\tfor {
\t\tn, err := r.Body.Read(tmp)
\t\tif n > 0 {
\t\t\tbuf = append(buf, tmp[:n]...)
\t\t}
\t\tif err != nil {
\t\t\tbreak
\t\t}
\t}
\tif err := s.Vault.WriteFileAtomic(reqPath, buf); err != nil {
\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)
\t\treturn
\t}
\twriteJSON(w, map[string]any{
\t\t"ok":   true,
\t\t"path": reqPath,
\t\t"time": time.Now().Format(time.RFC3339),
\t})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
\tflusher, ok := w.(http.Flusher)
\tif !ok {
\t\thttp.Error(w, "streaming not supported", http.StatusInternalServerError)
\t\treturn
\t}
\tw.Header().Set("Content-Type", "text/event-stream")
\tw.Header().Set("Cache-Control", "no-cache")
\tw.Header().Set("Connection", "keep-alive")

\tch, unsubscribe := s.Watcher.Subscribe()
\tdefer unsubscribe()

\tfmt.Fprintf(w, ": connected\n\n")
\tflusher.Flush()

\tfor {
\t\tselect {
\t\tcase ev, ok := <-ch:
\t\t\tif !ok {
\t\t\t\treturn
\t\t\t}
\t\t\tpayload, _ := json.Marshal(ev)
\t\t\tfmt.Fprintf(w, "data: %s\n\n", payload)
\t\t\tflusher.Flush()
\t\tcase <-r.Context().Done():
\t\t\treturn
\t\t}
\t}
}

func writeJSON(w http.ResponseWriter, v any) {
\tw.Header().Set("Content-Type", "application/json; charset=utf-8")
\tenc := json.NewEncoder(w)
\tenc.SetEscapeHTML(false)
\t_ = enc.Encode(v)
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
