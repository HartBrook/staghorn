// Package starter provides embedded starter templates that ship with staghorn.
package starter

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/*.md
var templatesFS embed.FS

// TemplateNames returns the list of available starter template names.
func TemplateNames() []string {
	entries, err := templatesFS.ReadDir("templates")
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

// BootstrapTemplates copies starter templates to the target directory.
// It skips files that already exist. Returns the number of templates copied.
func BootstrapTemplates(targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create templates directory: %w", err)
	}

	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded templates: %w", err)
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

		content, err := templatesFS.ReadFile(filepath.Join("templates", entry.Name()))
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

// GetTemplate returns the content of a starter template by name.
func GetTemplate(name string) ([]byte, error) {
	return templatesFS.ReadFile(filepath.Join("templates", name+".md"))
}
