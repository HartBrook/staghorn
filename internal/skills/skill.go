// Package skills handles staghorn skill parsing, registry, and syncing.
// Skills are directories containing SKILL.md plus optional supporting files.
// They follow the Agent Skills standard (agentskills.io) with Claude Code extensions.
package skills

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Source indicates where a skill came from.
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

// Arg defines a skill argument (same as commands.Arg).
type Arg struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Default     string   `yaml:"default"`
	Options     []string `yaml:"options,omitempty"`
	Required    bool     `yaml:"required"`
}

// Hooks defines pre/post execution hooks.
type Hooks struct {
	Pre  string `yaml:"pre,omitempty"`
	Post string `yaml:"post,omitempty"`
}

// Metadata is an arbitrary key-value map for additional skill metadata.
type Metadata map[string]string

// Frontmatter contains the YAML frontmatter of a skill.
// Fields follow the Agent Skills standard (agentskills.io) with Claude Code extensions.
type Frontmatter struct {
	// Agent Skills Standard fields (cross-tool compatible)
	Name          string   `yaml:"name"`
	Description   string   `yaml:"description"`
	License       string   `yaml:"license,omitempty"`
	Compatibility string   `yaml:"compatibility,omitempty"`
	Metadata      Metadata `yaml:"metadata,omitempty"`
	AllowedTools  string   `yaml:"allowed-tools,omitempty"` // Space-delimited per standard

	// Staghorn extensions (for backwards compatibility with commands)
	Tags []string `yaml:"tags,omitempty"`
	Args []Arg    `yaml:"args,omitempty"`

	// Claude Code extensions (ignored by other tools)
	DisableModelInvocation bool   `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool  `yaml:"user-invocable,omitempty"` // pointer to distinguish unset from false
	Context                string `yaml:"context,omitempty"`        // "normal" or "fork"
	Agent                  string `yaml:"agent,omitempty"`          // Subagent type for context: fork
	ArgumentHint           string `yaml:"argument-hint,omitempty"`
	Model                  string `yaml:"model,omitempty"`
	Hooks                  *Hooks `yaml:"hooks,omitempty"`
}

// Skill represents a staghorn skill.
type Skill struct {
	Frontmatter
	Body            string            // Markdown content after frontmatter
	Source          Source            // Where this skill came from
	DirPath         string            // Path to the skill directory
	SupportingFiles map[string]string // Relative path -> absolute path
}

// ParseDir parses a skill from its directory.
// The directory must contain a SKILL.md file.
func ParseDir(dirPath string, source Source) (*Skill, error) {
	skillMDPath := filepath.Join(dirPath, "SKILL.md")

	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	skill, err := Parse(string(content), source, dirPath)
	if err != nil {
		return nil, err
	}

	// Discover supporting files
	supportingFiles, err := discoverSupportingFiles(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to discover supporting files: %w", err)
	}
	skill.SupportingFiles = supportingFiles

	return skill, nil
}

// Parse parses a skill from SKILL.md content.
func Parse(content string, source Source, dirPath string) (*Skill, error) {
	lines := strings.Split(content, "\n")

	// Check for frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
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
		return nil, fmt.Errorf("skill must have a 'name' field in frontmatter")
	}

	if fm.Description == "" {
		return nil, fmt.Errorf("skill must have a 'description' field in frontmatter")
	}

	// Validate name format per Agent Skills standard
	if err := validateSkillName(fm.Name); err != nil {
		return nil, err
	}

	// Extract body (everything after frontmatter)
	body := ""
	if endIdx+1 < len(lines) {
		body = strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	}

	return &Skill{
		Frontmatter: fm,
		Body:        body,
		Source:      source,
		DirPath:     dirPath,
	}, nil
}

// validateSkillName validates that the name follows the Agent Skills standard:
// - Max 64 characters
// - Lowercase letters, numbers, and hyphens only
// - Must not start or end with hyphen
// - No consecutive hyphens
func validateSkillName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("skill name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("skill name exceeds 64 characters")
	}
	if name[0] == '-' || name[len(name)-1] == '-' {
		return fmt.Errorf("skill name cannot start or end with hyphen")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("skill name cannot contain consecutive hyphens")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("skill name contains invalid character '%c' (allowed: lowercase a-z, 0-9, -)", r)
		}
	}
	return nil
}

// discoverSupportingFiles walks the skill directory and returns all non-SKILL.md files.
func discoverSupportingFiles(dirPath string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and SKILL.md
		if info.IsDir() {
			return nil
		}
		if info.Name() == "SKILL.md" {
			return nil
		}

		// Get relative path from skill directory
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		files[relPath] = path
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// LoadFromDirectory loads all skills from a parent directory.
// Each subdirectory that contains a SKILL.md is treated as a skill.
func LoadFromDirectory(dir string, source Source) ([]*Skill, error) {
	var skills []*Skill

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty
		}
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skillMD := filepath.Join(skillDir, "SKILL.md")

		// Check if SKILL.md exists
		if _, err := os.Stat(skillMD); os.IsNotExist(err) {
			continue // Not a skill directory
		}

		skill, err := ParseDir(skillDir, source)
		if err != nil {
			// Log warning but continue loading other skills
			log.Printf("Warning: failed to parse skill %s: %v", entry.Name(), err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// HasArg checks if the skill has a specific argument.
func (s *Skill) HasArg(name string) bool {
	for _, arg := range s.Args {
		if arg.Name == name {
			return true
		}
	}
	return false
}

// GetArg returns an argument by name.
func (s *Skill) GetArg(name string) *Arg {
	for i := range s.Args {
		if s.Args[i].Name == name {
			return &s.Args[i]
		}
	}
	return nil
}

// AllowedToolsList returns the allowed tools as a slice.
// The standard uses space-delimited format.
func (s *Skill) AllowedToolsList() []string {
	if s.AllowedTools == "" {
		return nil
	}
	return strings.Fields(s.AllowedTools)
}

// IsUserInvocable returns true if the skill can be invoked by users.
// Defaults to true if not explicitly set.
func (s *Skill) IsUserInvocable() bool {
	if s.UserInvocable == nil {
		return true
	}
	return *s.UserInvocable
}

// ReadFrontmatterOnly reads just the frontmatter without loading supporting files.
// Useful for listing skills without loading all content into memory.
func ReadFrontmatterOnly(dirPath string) (*Frontmatter, error) {
	skillMDPath := filepath.Join(dirPath, "SKILL.md")

	file, err := os.Open(skillMDPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line must be ---
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
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

// NewSkillTemplate returns a template for creating a new skill.
func NewSkillTemplate(name, description string) string {
	return fmt.Sprintf(`---
name: %s
description: %s
allowed-tools: Read Grep Glob
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
