// Package starter provides embedded starter configs that ship with staghorn.
package starter

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed languages/*.md
var languagesFS embed.FS

// LanguageNames returns the list of available starter language config names.
func LanguageNames() []string {
	entries, err := languagesFS.ReadDir("languages")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			name := entry.Name()[:len(entry.Name())-3] // strip .md
			names = append(names, name)
		}
	}
	return names
}

// BootstrapLanguages copies starter language configs to the target directory.
// It skips files that already exist. Returns the number of files copied.
func BootstrapLanguages(targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create languages directory: %w", err)
	}

	entries, err := languagesFS.ReadDir("languages")
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded languages: %w", err)
	}

	copied := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := languagesFS.ReadFile(filepath.Join("languages", entry.Name()))
		if err != nil {
			return copied, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return copied, fmt.Errorf("failed to write %s: %w", entry.Name(), err)
		}

		copied++
	}

	return copied, nil
}

// GetLanguageConfig returns the content of a starter language config by ID.
func GetLanguageConfig(langID string) ([]byte, error) {
	return languagesFS.ReadFile(filepath.Join("languages", langID+".md"))
}
