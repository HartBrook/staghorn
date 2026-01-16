// Package commands handles staghorn command parsing, registry, and rendering.
package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Source indicates where a command came from.
type Source string

const (
	SourceTeam     Source = "team"
	SourcePersonal Source = "personal"
	SourceProject  Source = "project"
	SourceStarter  Source = "starter"
)

// Arg defines a command argument.
type Arg struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Default     string   `yaml:"default"`
	Options     []string `yaml:"options,omitempty"` // Valid options if constrained
	Required    bool     `yaml:"required"`
}

// Frontmatter contains the YAML frontmatter of a command.
type Frontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags,omitempty"`
	Args        []Arg    `yaml:"args,omitempty"`
}

// Command represents a staghorn command.
type Command struct {
	Frontmatter
	Body     string // Markdown content after frontmatter
	Source   Source // Where this command came from
	FilePath string // Path to the command file
}

// Parse parses a command from markdown content.
func Parse(content string, source Source, filePath string) (*Command, error) {
	lines := strings.Split(content, "\n")

	// Check for frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("command must start with YAML frontmatter (---)")
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
		return nil, fmt.Errorf("command must have a 'name' field in frontmatter")
	}

	// Extract body (everything after frontmatter)
	body := ""
	if endIdx+1 < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	}

	return &Command{
		Frontmatter: fm,
		Body:        body,
		Source:      source,
		FilePath:    filePath,
	}, nil
}

// ParseFile parses a command from a file.
func ParseFile(path string, source Source) (*Command, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read command file: %w", err)
	}
	return Parse(string(content), source, path)
}

// LoadFromDirectory loads all commands from a directory.
func LoadFromDirectory(dir string, source Source) ([]*Command, error) {
	var cmds []*Command

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty
		}
		return nil, fmt.Errorf("failed to read commands directory: %w", err)
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
		cmd, err := ParseFile(path, source)
		if err != nil {
			// Log warning but continue loading other commands
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			continue
		}

		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

// HasArg checks if the command has a specific argument.
func (c *Command) HasArg(name string) bool {
	for _, arg := range c.Args {
		if arg.Name == name {
			return true
		}
	}
	return false
}

// GetArg returns an argument by name.
func (c *Command) GetArg(name string) *Arg {
	for i := range c.Args {
		if c.Args[i].Name == name {
			return &c.Args[i]
		}
	}
	return nil
}

// ValidateArgs validates provided arguments against the command's definition.
func (c *Command) ValidateArgs(args map[string]string) error {
	// Build set of known arg names
	knownArgs := make(map[string]bool)
	for _, arg := range c.Args {
		knownArgs[arg.Name] = true
	}

	// Check for unknown args
	for name := range args {
		if !knownArgs[name] {
			return fmt.Errorf("unknown argument '%s' (available: %s)", name, c.argNames())
		}
	}

	// Check required args are provided
	for _, arg := range c.Args {
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
func (c *Command) argNames() string {
	if len(c.Args) == 0 {
		return "none"
	}
	names := make([]string, len(c.Args))
	for i, arg := range c.Args {
		names[i] = arg.Name
	}
	return strings.Join(names, ", ")
}

// GetArgWithDefault returns the argument value or its default.
func (c *Command) GetArgWithDefault(args map[string]string, name string) string {
	if val, ok := args[name]; ok {
		return val
	}

	if arg := c.GetArg(name); arg != nil {
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
	case SourceStarter:
		return "starter"
	default:
		return string(s)
	}
}

// NewCommandTemplate returns a template for creating a new command.
func NewCommandTemplate(name, description string) string {
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
// Useful for listing commands without loading all content into memory.
func ReadFrontmatterOnly(path string) (*Frontmatter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line must be ---
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("command must start with YAML frontmatter (---)")
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
