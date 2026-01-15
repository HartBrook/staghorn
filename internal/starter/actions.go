// Package starter provides embedded starter actions that ship with staghorn.
package starter

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed actions/*.md
var actionsFS embed.FS

// ActionNames returns the list of available starter action names.
func ActionNames() []string {
	entries, err := actionsFS.ReadDir("actions")
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

// BootstrapActions copies starter actions to the target directory.
// It skips files that already exist. Returns the number of actions copied.
func BootstrapActions(targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create actions directory: %w", err)
	}

	entries, err := actionsFS.ReadDir("actions")
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded actions: %w", err)
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

		content, err := actionsFS.ReadFile(filepath.Join("actions", entry.Name()))
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

// GetAction returns the content of a starter action by name.
func GetAction(name string) ([]byte, error) {
	return actionsFS.ReadFile(filepath.Join("actions", name+".md"))
}

// ListActions returns all embedded action files.
func ListActions() ([]fs.DirEntry, error) {
	return actionsFS.ReadDir("actions")
}
