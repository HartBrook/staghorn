package language

import (
	"os"
	"path/filepath"
	"strings"
)

// LanguageFile represents a loaded language config file.
type LanguageFile struct {
	Language string // Language ID (e.g., "python")
	Content  string // File contents
	Source   string // "team", "personal", or "project"
	Path     string // Full path to the file
}

// LoadLanguageFiles loads language configs from all layers for given languages.
// Returns a map of language ID to a slice of LanguageFiles (one per layer where found).
func LoadLanguageFiles(languages []string, teamDir, personalDir, projectDir string) (map[string][]*LanguageFile, error) {
	result := make(map[string][]*LanguageFile)

	for _, lang := range languages {
		files := make([]*LanguageFile, 0, 3)

		// Team layer (lowest priority, loaded first)
		if teamDir != "" {
			if content, path, err := readLanguageFile(teamDir, lang); err == nil {
				files = append(files, &LanguageFile{
					Language: lang,
					Content:  content,
					Source:   "team",
					Path:     path,
				})
			}
		}

		// Personal layer (middle priority)
		if personalDir != "" {
			if content, path, err := readLanguageFile(personalDir, lang); err == nil {
				files = append(files, &LanguageFile{
					Language: lang,
					Content:  content,
					Source:   "personal",
					Path:     path,
				})
			}
		}

		// Project layer (highest priority, loaded last)
		if projectDir != "" {
			if content, path, err := readLanguageFile(projectDir, lang); err == nil {
				files = append(files, &LanguageFile{
					Language: lang,
					Content:  content,
					Source:   "project",
					Path:     path,
				})
			}
		}

		if len(files) > 0 {
			result[lang] = files
		}
	}

	return result, nil
}

// ListAvailableLanguages returns all language IDs that have files in any of the given directories.
func ListAvailableLanguages(teamDir, personalDir, projectDir string) ([]string, error) {
	langSet := make(map[string]bool)

	for _, dir := range []string{teamDir, personalDir, projectDir} {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // Directory doesn't exist or can't be read
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".md") {
				langID := strings.TrimSuffix(name, ".md")
				langSet[langID] = true
			}
		}
	}

	result := make([]string, 0, len(langSet))
	for lang := range langSet {
		result = append(result, lang)
	}

	return result, nil
}

// readLanguageFile reads a language markdown file from a directory.
func readLanguageFile(dir, lang string) (content string, path string, err error) {
	path = filepath.Join(dir, lang+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return string(data), path, nil
}
