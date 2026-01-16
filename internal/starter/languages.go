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

// BootstrapLanguagesSelective copies only the specified starter language configs to the target directory.
// It skips files that already exist. Returns the count and names of installed configs.
func BootstrapLanguagesSelective(targetDir string, names []string) (int, []string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, nil, fmt.Errorf("failed to create languages directory: %w", err)
	}

	// Build set of requested names
	requested := make(map[string]bool)
	for _, name := range names {
		requested[name] = true
	}

	entries, err := languagesFS.ReadDir("languages")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read embedded languages: %w", err)
	}

	copied := 0
	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Get language name (strip .md extension)
		name := entry.Name()
		if filepath.Ext(name) == ".md" {
			name = name[:len(name)-3]
		}

		// Skip if not in requested list
		if !requested[name] {
			continue
		}

		targetPath := filepath.Join(targetDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(targetPath); err == nil {
			continue
		}

		content, err := languagesFS.ReadFile(filepath.Join("languages", entry.Name()))
		if err != nil {
			return copied, installed, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return copied, installed, fmt.Errorf("failed to write %s: %w", entry.Name(), err)
		}

		copied++
		installed = append(installed, name)
	}

	return copied, installed, nil
}

// GetLanguageConfig returns the content of a starter language config by ID.
func GetLanguageConfig(langID string) ([]byte, error) {
	return languagesFS.ReadFile(filepath.Join("languages", langID+".md"))
}
