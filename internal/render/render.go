// Package render turns markdown content into HTML for the initial
// server-side rendering of a page (see SPEC.md section 6.2). The
// client-side Milkdown editor takes over once the user switches into
// edit mode.
package render

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

func ToHTML(source []byte) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
