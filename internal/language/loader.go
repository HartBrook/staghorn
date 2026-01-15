package language

import (
	"os"
	"path/filepath"
	"regexp"
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
// Personal and project files without user content (just headings/comments) are skipped.
func LoadLanguageFiles(languages []string, teamDir, personalDir, projectDir string) (map[string][]*LanguageFile, error) {
	result := make(map[string][]*LanguageFile)

	for _, lang := range languages {
		files := make([]*LanguageFile, 0, 3)

		// Team layer (lowest priority, loaded first)
		// Team files are always included - they're managed by the team
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
		// Skip if file exists but has no user content (just template heading)
		if personalDir != "" {
			if content, path, err := readLanguageFile(personalDir, lang); err == nil {
				if HasUserContent(content) {
					files = append(files, &LanguageFile{
						Language: lang,
						Content:  content,
						Source:   "personal",
						Path:     path,
					})
				}
			}
		}

		// Project layer (highest priority, loaded last)
		// Skip if file exists but has no user content (just template heading)
		if projectDir != "" {
			if content, path, err := readLanguageFile(projectDir, lang); err == nil {
				if HasUserContent(content) {
					files = append(files, &LanguageFile{
						Language: lang,
						Content:  content,
						Source:   "project",
						Path:     path,
					})
				}
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

// Regex patterns for content detection
var (
	// Matches markdown headings: # Heading, ## Heading, etc.
	headingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+.*$`)
	// Matches HTML comments: <!-- ... -->
	htmlCommentPattern = regexp.MustCompile(`<!--[\s\S]*?-->`)
)

// HasUserContent checks if a markdown file contains user-added content
// beyond just headings and HTML comments. This is used to skip "empty"
// personal language files that were created but not yet customized.
func HasUserContent(content string) bool {
	// Remove HTML comments (including staghorn markers)
	content = htmlCommentPattern.ReplaceAllString(content, "")

	// Remove markdown headings
	content = headingPattern.ReplaceAllString(content, "")

	// Remove whitespace
	content = strings.TrimSpace(content)

	// If anything remains, the user added content
	return content != ""
}
