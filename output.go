package preflight

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// WriteJSON writes a machine-readable result.
func WriteJSON(w io.Writer, result Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// WriteHuman writes aligned line-oriented output.
func WriteHuman(w io.Writer, result Result) error {
	type row struct {
		findings string
		path     string
	}
	rows := make([]row, 0, len(result.Repositories))
	max := 0
	for _, repo := range result.Repositories {
		findings := findingsString(repo.Findings)
		if len(findings) > max {
			max = len(findings)
		}
		rows = append(rows, row{findings: findings, path: renderPath(result, repo.Path)})
	}
	for _, row := range rows {
		if max == 0 {
			if _, err := fmt.Fprintln(w, row.path); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%-*s  %s\n", max, row.findings, row.path); err != nil {
			return err
		}
	}
	return nil
}

func findingsString(findings []Finding) string {
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		parts = append(parts, string(finding))
	}
	return strings.Join(parts, " ")
}

func renderPath(result Result, path string) string {
	if result.Recursive {
		if rel, err := filepath.Rel(result.Root, path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return path
}
