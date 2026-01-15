// Package language provides language detection and configuration loading.
package language

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Language represents a supported programming language.
type Language struct {
	ID          string   // "python", "go", "typescript"
	DisplayName string   // "Python", "Go", "TypeScript"
	Markers     []string // Detection marker files
}

// SupportedLanguages lists all languages staghorn can detect.
var SupportedLanguages = []Language{
	{ID: "python", DisplayName: "Python", Markers: []string{"pyproject.toml", "setup.py", "requirements.txt", "Pipfile", "poetry.lock"}},
	{ID: "go", DisplayName: "Go", Markers: []string{"go.mod"}},
	{ID: "typescript", DisplayName: "TypeScript", Markers: []string{"tsconfig.json"}},
	{ID: "javascript", DisplayName: "JavaScript", Markers: []string{"package.json"}},
	{ID: "rust", DisplayName: "Rust", Markers: []string{"Cargo.toml"}},
	{ID: "java", DisplayName: "Java", Markers: []string{"pom.xml", "build.gradle", "build.gradle.kts"}},
	{ID: "ruby", DisplayName: "Ruby", Markers: []string{"Gemfile"}},
	{ID: "csharp", DisplayName: "C#", Markers: []string{"*.csproj", "*.sln"}},
	{ID: "swift", DisplayName: "Swift", Markers: []string{"Package.swift"}},
	{ID: "kotlin", DisplayName: "Kotlin", Markers: []string{"build.gradle.kts"}},
}

// LanguageConfig contains language-specific settings.
type LanguageConfig struct {
	AutoDetect bool     `yaml:"auto_detect"`
	Enabled    []string `yaml:"enabled,omitempty"`
	Disabled   []string `yaml:"disabled,omitempty"`
}

// Detect scans projectRoot for language marker files.
// Returns a sorted list of detected language IDs.
func Detect(projectRoot string) ([]string, error) {
	detected := make(map[string]bool)

	for _, lang := range SupportedLanguages {
		for _, marker := range lang.Markers {
			if containsWildcard(marker) {
				matches, err := filepath.Glob(filepath.Join(projectRoot, marker))
				if err == nil && len(matches) > 0 {
					detected[lang.ID] = true
					break
				}
			} else {
				path := filepath.Join(projectRoot, marker)
				if _, err := os.Stat(path); err == nil {
					detected[lang.ID] = true
					break
				}
			}
		}
	}

	// TypeScript supersedes JavaScript (if both detected, keep only TypeScript)
	if detected["typescript"] {
		delete(detected, "javascript")
	}

	// Kotlin and Java both use build.gradle.kts, prefer to keep both if detected
	// (Kotlin projects often include Java code)

	result := make([]string, 0, len(detected))
	for lang := range detected {
		result = append(result, lang)
	}
	sort.Strings(result)
	return result, nil
}

// Resolve determines final language list from config + detection.
func Resolve(cfg *LanguageConfig, projectRoot string) ([]string, error) {
	// Explicit list takes precedence
	if len(cfg.Enabled) > 0 {
		return FilterDisabled(cfg.Enabled, cfg.Disabled), nil
	}

	// If auto-detect is disabled and no explicit list, return empty
	if !cfg.AutoDetect {
		return []string{}, nil
	}

	// Auto-detect from project files
	detected, err := Detect(projectRoot)
	if err != nil {
		return nil, err
	}

	return FilterDisabled(detected, cfg.Disabled), nil
}

// GetLanguage returns the Language struct for a given ID.
func GetLanguage(id string) *Language {
	for i := range SupportedLanguages {
		if SupportedLanguages[i].ID == id {
			return &SupportedLanguages[i]
		}
	}
	return nil
}

// GetDisplayName returns the display name for a language ID.
func GetDisplayName(id string) string {
	if lang := GetLanguage(id); lang != nil {
		return lang.DisplayName
	}
	return cases.Title(language.English).String(id)
}

// FilterDisabled removes disabled languages from the list.
func FilterDisabled(languages, disabled []string) []string {
	if len(disabled) == 0 {
		return languages
	}

	disabledSet := make(map[string]bool)
	for _, d := range disabled {
		disabledSet[d] = true
	}

	result := make([]string, 0, len(languages))
	for _, lang := range languages {
		if !disabledSet[lang] {
			result = append(result, lang)
		}
	}
	return result
}

// containsWildcard checks if a pattern contains glob wildcards.
func containsWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}
