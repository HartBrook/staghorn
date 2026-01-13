// Package actions handles staghorn action parsing, registry, and rendering.
package actions

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Source indicates where an action came from.
type Source string

const (
	SourceTeam     Source = "team"
	SourcePersonal Source = "personal"
	SourceProject  Source = "project"
)

// Arg defines an action argument.
type Arg struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Default     string   `yaml:"default"`
	Options     []string `yaml:"options,omitempty"` // Valid options if constrained
	Required    bool     `yaml:"required"`
}

// Frontmatter contains the YAML frontmatter of an action.
type Frontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags,omitempty"`
	Args        []Arg    `yaml:"args,omitempty"`
}

// Action represents a staghorn action.
type Action struct {
	Frontmatter
	Body     string // Markdown content after frontmatter
	Source   Source // Where this action came from
	FilePath string // Path to the action file
}

// Parse parses an action from markdown content.
func Parse(content string, source Source, filePath string) (*Action, error) {
	lines := strings.Split(content, "\n")

	// Check for frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("action must start with YAML frontmatter (---)")
	}

	// Find end of frontmatter
	endIdx := -1
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
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	if fm.Name == "" {
		return nil, fmt.Errorf("action must have a 'name' field in frontmatter")
	}

	// Extract body (everything after frontmatter)
	body := ""
	if endIdx+1 < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	}

	return &Action{
		Frontmatter: fm,
		Body:        body,
		Source:      source,
		FilePath:    filePath,
	}, nil
}

// ParseFile parses an action from a file.
func ParseFile(path string, source Source) (*Action, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read action file: %w", err)
	}
	return Parse(string(content), source, path)
}

// LoadFromDirectory loads all actions from a directory.
func LoadFromDirectory(dir string, source Source) ([]*Action, error) {
	var actions []*Action

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty
		}
		return nil, fmt.Errorf("failed to read actions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .md files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		action, err := ParseFile(path, source)
		if err != nil {
			// Log warning but continue loading other actions
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			continue
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// HasArg checks if the action has a specific argument.
func (a *Action) HasArg(name string) bool {
	for _, arg := range a.Args {
		if arg.Name == name {
			return true
		}
	}
	return false
}

// GetArg returns an argument by name.
func (a *Action) GetArg(name string) *Arg {
	for i := range a.Args {
		if a.Args[i].Name == name {
			return &a.Args[i]
		}
	}
	return nil
}

// ValidateArgs validates provided arguments against the action's definition.
func (a *Action) ValidateArgs(args map[string]string) error {
	// Build set of known arg names
	knownArgs := make(map[string]bool)
	for _, arg := range a.Args {
		knownArgs[arg.Name] = true
	}

	// Check for unknown args
	for name := range args {
		if !knownArgs[name] {
			return fmt.Errorf("unknown argument '%s' (available: %s)", name, a.argNames())
		}
	}

	// Check required args are provided
	for _, arg := range a.Args {
		if arg.Required {
			if _, ok := args[arg.Name]; !ok {
				return fmt.Errorf("required argument '%s' not provided", arg.Name)
			}
		}

		// Validate options if constrained
		if len(arg.Options) > 0 {
			if val, ok := args[arg.Name]; ok {
				valid := false
				for _, opt := range arg.Options {
					if val == opt {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid value '%s' for argument '%s' (valid: %s)",
						val, arg.Name, strings.Join(arg.Options, ", "))
				}
			}
		}
	}

	return nil
}

// argNames returns a comma-separated list of argument names.
func (a *Action) argNames() string {
	if len(a.Args) == 0 {
		return "none"
	}
	names := make([]string, len(a.Args))
	for i, arg := range a.Args {
		names[i] = arg.Name
	}
	return strings.Join(names, ", ")
}

// GetArgWithDefault returns the argument value or its default.
func (a *Action) GetArgWithDefault(args map[string]string, name string) string {
	if val, ok := args[name]; ok {
		return val
	}

	if arg := a.GetArg(name); arg != nil {
		return arg.Default
	}

	return ""
}

// SourceLabel returns a human-readable label for the source.
func (s Source) Label() string {
	switch s {
	case SourceTeam:
		return "team"
	case SourcePersonal:
		return "personal"
	case SourceProject:
		return "project"
	default:
		return string(s)
	}
}

// NewActionTemplate returns a template for creating a new action.
func NewActionTemplate(name, description string) string {
	return fmt.Sprintf(`---
name: %s
description: %s
tags: []
args:
  - name: path
    description: Target path to analyze
    default: "."
---

# %s

%s

## Instructions

1. First step
2. Second step
3. Third step

## Output Format

Describe the expected output format here.
`, name, description, toTitleCase(name), description)
}

// toTitleCase converts kebab-case to Title Case.
func toTitleCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// ReadFrontmatterOnly reads just the frontmatter without loading the full body.
// Useful for listing actions without loading all content into memory.
func ReadFrontmatterOnly(path string) (*Frontmatter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line must be ---
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("action must start with YAML frontmatter (---)")
	}

	// Read until closing ---
	var yamlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		yamlLines = append(yamlLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return nil, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	return &fm, nil
}
