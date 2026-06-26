// Package report — JSON report serialization.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSON serializes v to a JSON file at path, creating parent dirs as needed.
func WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("report: mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("report: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("report: write %s: %w", path, err)
	}
	return nil
}

// WriteMarkdown writes a markdown string to a file.
func WriteMarkdown(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("report: mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
