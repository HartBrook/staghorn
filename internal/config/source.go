// Package config handles staghorn configuration.
package config

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// repoPattern matches owner/repo format.
var repoPattern = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)$`)

// SourceConfig supports both simple string and multi-source configurations.
// Simple: source: "owner/repo"
// Multi:  source: { default: "owner/repo", base: "other/repo", languages: {...} }
type SourceConfig struct {
	// Default is the fallback source for all items not explicitly configured.
	Default string `yaml:"default,omitempty"`

	// Base overrides the source for the main CLAUDE.md file.
	Base string `yaml:"base,omitempty"`

	// Languages maps language IDs to their source repos.
	// Example: { "python": "acme/python-standards" }
	Languages map[string]string `yaml:"languages,omitempty"`

	// Commands maps command names to their source repos.
	// Example: { "code-review": "acme/internal-commands" }
	Commands map[string]string `yaml:"commands,omitempty"`

	// Skills maps skill names to their source repos.
	// Example: { "react": "vercel-labs/agent-skills/skills/react" }
	Skills map[string]string `yaml:"skills,omitempty"`
}

// Source wraps the flexible source configuration.
// It can be unmarshaled from either a string or an object.
type Source struct {
	// Simple holds the repo string when source is a simple string.
	Simple string

	// Multi holds the structured config when source is an object.
	Multi *SourceConfig
}

// UnmarshalYAML implements custom unmarshaling to handle both string and object formats.
func (s *Source) UnmarshalYAML(node *yaml.Node) error {
	// Try string first
	if node.Kind == yaml.ScalarNode {
		s.Simple = node.Value
		return nil
	}

	// Try object
	if node.Kind == yaml.MappingNode {
		s.Multi = &SourceConfig{}
		return node.Decode(s.Multi)
	}

	return fmt.Errorf("source must be a string or object, got %v", node.Kind)
}

// MarshalYAML implements custom marshaling to output the appropriate format.
func (s Source) MarshalYAML() (interface{}, error) {
	if s.Simple != "" {
		return s.Simple, nil
	}
	return s.Multi, nil
}

// IsMultiSource returns true if this is a multi-source configuration.
func (s *Source) IsMultiSource() bool {
	return s.Multi != nil
}

// DefaultRepo returns the default repository for this source configuration.
func (s *Source) DefaultRepo() string {
	if s.Simple != "" {
		return s.Simple
	}
	if s.Multi != nil {
		return s.Multi.Default
	}
	return ""
}

// RepoForBase returns the repository to use for the base CLAUDE.md.
func (s *Source) RepoForBase() string {
	if s.Multi != nil && s.Multi.Base != "" {
		return s.Multi.Base
	}
	return s.DefaultRepo()
}

// RepoForLanguage returns the repository to use for a specific language config.
func (s *Source) RepoForLanguage(lang string) string {
	if s.Multi != nil && s.Multi.Languages != nil {
		if repo, ok := s.Multi.Languages[lang]; ok {
			return repo
		}
	}
	return s.DefaultRepo()
}

// RepoForCommand returns the repository to use for a specific command.
func (s *Source) RepoForCommand(cmd string) string {
	if s.Multi != nil && s.Multi.Commands != nil {
		if repo, ok := s.Multi.Commands[cmd]; ok {
			return repo
		}
	}
	return s.DefaultRepo()
}

// RepoForSkill returns the repository to use for a specific skill.
func (s *Source) RepoForSkill(skill string) string {
	if s.Multi != nil && s.Multi.Skills != nil {
		if repo, ok := s.Multi.Skills[skill]; ok {
			return repo
		}
	}
	return s.DefaultRepo()
}

// AllRepos returns all unique repositories referenced by this source config.
// Useful for syncing all sources at once.
func (s *Source) AllRepos() []string {
	seen := make(map[string]bool)
	var repos []string

	addRepo := func(repo string) {
		if repo != "" && !seen[repo] {
			seen[repo] = true
			repos = append(repos, repo)
		}
	}

	addRepo(s.DefaultRepo())

	if s.Multi != nil {
		addRepo(s.Multi.Base)
		for _, repo := range s.Multi.Languages {
			addRepo(repo)
		}
		for _, repo := range s.Multi.Commands {
			addRepo(repo)
		}
		for _, repo := range s.Multi.Skills {
			addRepo(repo)
		}
	}

	return repos
}

// ParseRepo extracts owner and repo name from a repository string.
// Accepts formats:
//   - "https://github.com/owner/repo"
//   - "https://github.com/owner/repo.git"
//   - "https://github.com/owner/repo/tree/main"
//   - "https://github.com/owner/repo/blob/main/file.md"
//   - "github.com/owner/repo"
//   - "owner/repo"
func ParseRepo(repoStr string) (owner, repo string, err error) {
	if repoStr == "" {
		return "", "", fmt.Errorf("repository string is empty")
	}

	// Strip protocol
	repoStr = strings.TrimPrefix(repoStr, "https://")
	repoStr = strings.TrimPrefix(repoStr, "http://")

	// Strip host (github.com for now, extensible for gitlab.com etc.)
	repoStr = strings.TrimPrefix(repoStr, "github.com/")

	// Strip .git suffix and trailing slashes
	repoStr = strings.TrimSuffix(repoStr, ".git")
	repoStr = strings.TrimSuffix(repoStr, "/")

	// Split by / and take only owner/repo (first two parts)
	// This handles URLs like owner/repo/tree/main or owner/repo/blob/main/file.md
	parts := strings.Split(repoStr, "/")
	if len(parts) >= 2 {
		repoStr = parts[0] + "/" + parts[1]
	}

	// Match owner/repo pattern
	matches := repoPattern.FindStringSubmatch(repoStr)
	if matches == nil {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repoStr)
	}

	return matches[1], matches[2], nil
}

// IsEmpty returns true if no source is configured.
func (s *Source) IsEmpty() bool {
	return s.DefaultRepo() == ""
}

// Validate checks that the source configuration is valid.
// An empty source is valid (for local-only mode).
func (s *Source) Validate() error {
	defaultRepo := s.DefaultRepo()
	if defaultRepo == "" {
		// Empty source is valid for local-only mode
		return nil
	}

	// Validate default repo format
	if _, _, err := ParseRepo(defaultRepo); err != nil {
		return fmt.Errorf("invalid default source: %w", err)
	}

	// Validate multi-source repos if present
	if s.Multi != nil {
		if s.Multi.Base != "" {
			if _, _, err := ParseRepo(s.Multi.Base); err != nil {
				return fmt.Errorf("invalid base source: %w", err)
			}
		}
		for lang, repo := range s.Multi.Languages {
			if _, _, err := ParseRepo(repo); err != nil {
				return fmt.Errorf("invalid source for language %q: %w", lang, err)
			}
		}
		for cmd, repo := range s.Multi.Commands {
			if _, _, err := ParseRepo(repo); err != nil {
				return fmt.Errorf("invalid source for command %q: %w", cmd, err)
			}
		}
		for skill, repo := range s.Multi.Skills {
			if _, _, err := ParseRepo(repo); err != nil {
				return fmt.Errorf("invalid source for skill %q: %w", skill, err)
			}
		}
	}

	return nil
}
