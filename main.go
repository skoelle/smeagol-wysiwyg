// Command smeagol-wysiwyg serves a single vault directory of markdown
// files as a WYSIWYG wiki. See SPEC.md for the full specification.
//
// Usage:
//
//\tsmeagol-wysiwyg [--host 127.0.0.1] [--port 8000] [vault-path]
//
// If vault-path is omitted, the current working directory is used.
package main

import (
\t"flag"
\t"fmt"
\t"log"
\t"net/http"
\t"os"

\t"github.com/USERNAME/smeagol-wysiwyg/internal/server"
\t"github.com/USERNAME/smeagol-wysiwyg/internal/vault"
\t"github.com/USERNAME/smeagol-wysiwyg/internal/watcher"
)

func main() {
\thost := flag.String("host", "127.0.0.1", "host/IP to bind to")
\tport := flag.Int("port", 8000, "port to listen on")
\tflag.Parse()

\tvaultPath := "."
\tif args := flag.Args(); len(args) > 0 {
\t\tvaultPath = args[0]
\t}

\tv, err := vault.New(vaultPath)
\tif err != nil {
\t\tlog.Fatalf("smeagol-wysiwyg: %v", err)
\t}

\tw, err := watcher.New(v.Root)
\tif err != nil {
\t\tlog.Fatalf("smeagol-wysiwyg: starting watcher: %v", err)
\t}
\tdefer w.Close()

\tassets, err := assetsFS()
\tif err != nil {
\t\tlog.Fatalf("smeagol-wysiwyg: loading embedded assets: %v", err)
\t}

\tsrv := server.New(v, w, assets)

\taddr := fmt.Sprintf("%s:%d", *host, *port)
\tlog.Printf("smeagol-wysiwyg: vault=%s addr=http://%s", v.Root, addr)

\tif err := http.ListenAndServe(addr, srv); err != nil {
\t\tfmt.Fprintln(os.Stderr, "smeagol-wysiwyg:", err)
\t\tos.Exit(1)
\t}
}
