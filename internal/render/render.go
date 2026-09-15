// Package render turns markdown content into HTML for the initial
// server-side rendering of a page (see SPEC.md section 6.2). The
// client-side Milkdown editor takes over once the user switches into
// edit mode.
package render

import (
\t"bytes"

\t"github.com/yuin/goldmark"
\t"github.com/yuin/goldmark/extension"
\t"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
\tgoldmark.WithExtensions(extension.GFM),
\tgoldmark.WithRendererOptions(
\t\thtml.WithUnsafe(),
\t),
)

func ToHTML(source []byte) (string, error) {
\tvar buf bytes.Buffer
\tif err := md.Convert(source, &buf); err != nil {
\t\treturn "", err
\t}
\treturn buf.String(), nil
}
