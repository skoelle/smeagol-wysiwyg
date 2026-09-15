package main

import (
\t"embed"
\t"io/fs"
)

// web/dist contains the frontend assets served under /assets/: the
// vanilla-JS application (main.js, loading the Milkdown WYSIWYG editor
// on demand from a CDN) and the stylesheet (style.css). No Node/npm
// build step is required; go build embeds these files as they are.
//
//go:embed web/dist
var embeddedAssets embed.FS

func assetsFS() (fs.FS, error) {
\treturn fs.Sub(embeddedAssets, "web/dist")
}
