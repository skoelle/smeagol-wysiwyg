// Command smeagol-wysiwyg serves a single vault directory of markdown
// files as a WYSIWYG wiki. See SPEC.md for the full specification.
//
// Usage:
//
//	smeagol-wysiwyg [--host 127.0.0.1] [--port 8000] [vault-path]
//
// If vault-path is omitted, the current working directory is used.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/USERNAME/smeagol-wysiwyg/internal/server"
	"github.com/USERNAME/smeagol-wysiwyg/internal/vault"
	"github.com/USERNAME/smeagol-wysiwyg/internal/watcher"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host/IP to bind to")
	port := flag.Int("port", 8000, "port to listen on")
	flag.Parse()

	vaultPath := "."
	if args := flag.Args(); len(args) > 0 {
		vaultPath = args[0]
	}

	v, err := vault.New(vaultPath)
	if err != nil {
		log.Fatalf("smeagol-wysiwyg: %v", err)
	}

	w, err := watcher.New(v.Root)
	if err != nil {
		log.Fatalf("smeagol-wysiwyg: starting watcher: %v", err)
	}
	defer w.Close()

	assets, err := assetsFS()
	if err != nil {
		log.Fatalf("smeagol-wysiwyg: loading embedded assets: %v", err)
	}

	srv := server.New(v, w, assets)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("smeagol-wysiwyg: vault=%s addr=http://%s", v.Root, addr)

	if err := http.ListenAndServe(addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, "smeagol-wysiwyg:", err)
		os.Exit(1)
	}
}
