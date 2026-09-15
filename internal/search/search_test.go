package search

import (
\t"os"
\t"path/filepath"
\t"testing"
)

func setup(t *testing.T) string {
\tt.Helper()
\tdir := t.TempDir()
\tos.MkdirAll(filepath.Join(dir, "unter", "tief", "verzeichnis"), 0o755)
\tos.WriteFile(filepath.Join(dir, "README.md"), []byte("# Übersicht\nDies ist ein Test mit Umlaut: Käse und Müll.\n"), 0o644)
\tos.WriteFile(filepath.Join(dir, "unter", "tief", "verzeichnis", "seite.md"), []byte("# Tiefe Seite\nEnthält auch Käse.\n"), 0o644)
\treturn dir
}

func TestSearchFindsUmlautContent(t *testing.T) {
\tdir := setup(t)
\tresults, err := Search(dir, "Käse")
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif len(results) != 2 {
\t\tt.Fatalf("expected 2 hits, got %d: %+v", len(results), results)
\t}
}

func TestSearchCaseInsensitive(t *testing.T) {
\tdir := setup(t)
\tresults, err := Search(dir, "käse")
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif len(results) != 2 {
\t\tt.Fatalf("expected 2 hits (case-insensitive), got %d", len(results))
\t}
}

func TestSearchDeepPath(t *testing.T) {
\tdir := setup(t)
\tresults, err := Search(dir, "Tiefe Seite")
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tfound := false
\tfor _, r := range results {
\t\tif r.Path == "unter/tief/verzeichnis/seite.md" {
\t\t\tfound = true
\t\t}
\t}
\tif !found {
\t\tt.Fatalf("expected hit in deeply nested path, got %+v", results)
\t}
}

func TestSearchEmptyQuery(t *testing.T) {
\tdir := setup(t)
\tresults, err := Search(dir, "")
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tif results != nil {
\t\tt.Fatalf("expected no results for empty query, got %+v", results)
\t}
}
