package search

import (
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub", "deep", "nested"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Overview\nTest with umlauts: Käse and Müll.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "deep", "nested", "page.md"), []byte("# Deep Page\nAlso contains Käse.\n"), 0o644)
	return dir
}

func TestSearchFindsUmlautContent(t *testing.T) {
	dir := setup(t)
	results, err := Search(dir, "Käse")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 hits, got %d: %+v", len(results), results)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	dir := setup(t)
	results, err := Search(dir, "käse")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 hits (case-insensitive), got %d", len(results))
	}
}

func TestSearchDeepPath(t *testing.T) {
	dir := setup(t)
	results, err := Search(dir, "Deep Page")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range results {
		if r.Path == "sub/deep/nested/page.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hit in deeply nested path, got %+v", results)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	dir := setup(t)
	results, err := Search(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected no results for empty query, got %+v", results)
	}
}
