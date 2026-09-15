// Package search implements the vault search as a straightforward
// recursive filesystem scan (grep-style), on purpose without any
// persistent index. See SPEC.md section 5.6.
package search

import (
\t"bufio"
\t"os"
\t"path/filepath"
\t"sort"
\t"strings"
)

type Result struct {
\tPath    string `json:"path"`
\tTitle   string `json:"title"`
\tLine    int    `json:"line"`
\tSnippet string `json:"snippet"`
}

func Search(root, query string) ([]Result, error) {
\tquery = strings.TrimSpace(query)
\tif query == "" {
\t\treturn nil, nil
\t}
\tneedle := strings.ToLower(query)

\tvar results []Result
\terr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
\t\tif err != nil {
\t\t\treturn nil
\t\t}
\t\tif d.IsDir() {
\t\t\tif strings.HasPrefix(d.Name(), ".") && path != root {
\t\t\t\treturn filepath.SkipDir
\t\t\t}
\t\t\treturn nil
\t\t}
\t\tif !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
\t\t\treturn nil
\t\t}

\t\trel, err := filepath.Rel(root, path)
\t\tif err != nil {
\t\t\treturn nil
\t\t}
\t\trel = filepath.ToSlash(rel)

\t\tfileHit := strings.Contains(strings.ToLower(rel), needle)

\t\tf, err := os.Open(path)
\t\tif err != nil {
\t\t\treturn nil
\t\t}
\t\tdefer f.Close()

\t\ttitle := d.Name()
\t\tscanner := bufio.NewScanner(f)
\t\tlineNo := 0
\t\tmatchedAny := false
\t\tfor scanner.Scan() {
\t\t\tlineNo++
\t\t\tline := scanner.Text()
\t\t\tif title == d.Name() && strings.HasPrefix(strings.TrimSpace(line), "# ") {
\t\t\t\ttitle = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# "))
\t\t\t}
\t\t\tif strings.Contains(strings.ToLower(line), needle) {
\t\t\t\tmatchedAny = true
\t\t\t\tresults = append(results, Result{
\t\t\t\t\tPath:    rel,
\t\t\t\t\tTitle:   title,
\t\t\t\t\tLine:    lineNo,
\t\t\t\t\tSnippet: strings.TrimSpace(line),
\t\t\t\t})
\t\t\t}
\t\t}
\t\tif fileHit && !matchedAny {
\t\t\tresults = append(results, Result{
\t\t\t\tPath:    rel,
\t\t\t\tTitle:   title,
\t\t\t\tLine:    0,
\t\t\t\tSnippet: "(Treffer im Dateinamen/Pfad)",
\t\t\t})
\t\t}
\t\treturn nil
\t})
\tif err != nil {
\t\treturn nil, err
\t}

\tsort.SliceStable(results, func(i, j int) bool {
\t\tif results[i].Path == results[j].Path {
\t\t\treturn results[i].Line < results[j].Line
\t\t}
\t\treturn results[i].Path < results[j].Path
\t})
\treturn results, nil
}
