// Package search implements the vault search as a straightforward
// recursive filesystem scan (grep-style), on purpose without any
// persistent index. See SPEC.md section 5.6.
package search

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Result struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

func Search(root, query string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	needle := strings.ToLower(query)

	var results []Result
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		fileHit := strings.Contains(strings.ToLower(rel), needle)

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		title := d.Name()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		lineNo := 0
		matchedAny := false
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if title == d.Name() && strings.HasPrefix(strings.TrimSpace(line), "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# "))
			}
			if strings.Contains(strings.ToLower(line), needle) {
				matchedAny = true
				results = append(results, Result{
					Path:    rel,
					Title:   title,
					Line:    lineNo,
					Snippet: strings.TrimSpace(line),
				})
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("search: scanner error reading %s: %v", path, err)
		}
		if fileHit && !matchedAny {
			results = append(results, Result{
				Path:    rel,
				Title:   title,
				Line:    0,
				Snippet: "(Treffer im Dateinamen/Pfad)",
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Path == results[j].Path {
			return results[i].Line < results[j].Line
		}
		return results[i].Path < results[j].Path
	})
	return results, nil
}
