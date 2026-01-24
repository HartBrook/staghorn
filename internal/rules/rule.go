// Package rules handles staghorn rule parsing, registry, and rendering.
package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseErrors collects multiple parse errors when loading rules from a directory.
// Individual parse failures don't prevent other rules from loading.
type ParseErrors struct {
	Errors []error
}

func (e *ParseErrors) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d rules failed to parse", len(e.Errors))
}

// Source indicates where a rule came from.
type Source string

const (
	SourceTeam     Source = "team"
	SourcePersonal Source = "personal"
	SourceProject  Source = "project"
	SourceStarter  Source = "starter"
)

// Label returns a human-readable label for the source.
func (s Source) Label() string {
	switch s {
	case SourceTeam:
		return "team"
	case SourcePersonal:
		return "personal"
	case SourceProject:
		return "project"
	case SourceStarter:
		return "starter"
	default:
		return string(s)
	}
}

// Frontmatter contains the YAML frontmatter of a rule.
type Frontmatter struct {
	Paths []string `yaml:"paths,omitempty"` // Glob patterns for path-scoping
}

// Rule represents a staghorn rule file.
type Rule struct {
	Frontmatter        // Embedded; access paths via rule.Frontmatter.Paths or rule.Paths
	Name        string // Derived from filename (e.g., "security" from "security.md")
	Body        string // Markdown content after frontmatter
	Source      Source // Where this rule came from
	FilePath    string // Absolute path to the rule file
	RelPath     string // Relative path within rules/ (preserves subdirs, e.g., "api/rest.md")
}

// Parse parses a rule from markdown content.
func Parse(content string, source Source, filePath, relPath string) (*Rule, error) {
	lines := strings.Split(content, "\n")

	var fm Frontmatter
	var body string
	var endIdx int

	// Check for frontmatter delimiter
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		// Find end of frontmatter
		endIdx = -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIdx = i
				break
			}
		}

		if endIdx == -1 {
			return nil, fmt.Errorf("unterminated frontmatter (missing closing ---)")
		}

		// Parse frontmatter
		frontmatterYAML := strings.Join(lines[1:endIdx], "\n")
		if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
			return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
		}

		// Extract body (everything after frontmatter)
		if endIdx+1 < len(lines) {
			body = strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
		}
	} else {
		// No frontmatter, entire content is the body
		body = strings.TrimSpace(content)
	}

	// Derive name from filename
	name := strings.TrimSuffix(filepath.Base(relPath), ".md")

	return &Rule{
		Frontmatter: fm,
		Name:        name,
		Body:        body,
		Source:      source,
		FilePath:    filePath,
		RelPath:     relPath,
	}, nil
}

// ParseFile parses a rule from a file.
func ParseFile(path string, source Source, relPath string) (*Rule, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}
	return Parse(string(content), source, path, relPath)
}

// LoadFromDirectory loads all rules from a directory (recursive).
// Parse errors for individual files are collected in the returned ParseErrors slice
// but do not prevent other rules from loading.
func LoadFromDirectory(dir string, source Source) ([]*Rule, error) {
	var rules []*Rule
	var parseErrors []error

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // Directory doesn't exist, return empty
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Calculate relative path from the rules directory
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		rule, err := ParseFile(path, source, relPath)
		if err != nil {
			// Collect parse error but continue loading other rules
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse %s: %w", path, err))
			return nil
		}

		rules = append(rules, rule)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read rules directory: %w", err)
	}

	// Return first parse error if any occurred (caller can check for more via type assertion)
	if len(parseErrors) > 0 {
		return rules, &ParseErrors{Errors: parseErrors}
	}

	return rules, nil
}

// HasPathScope returns true if the rule has path-specific scope.
func (r *Rule) HasPathScope() bool {
	return len(r.Frontmatter.Paths) > 0
}
