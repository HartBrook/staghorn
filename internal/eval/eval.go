// Package eval handles staghorn eval definitions, parsing, and execution.
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Source indicates where an eval came from.
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

// Eval represents a staghorn eval definition.
type Eval struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags,omitempty"`
	Context     Context  `yaml:"context,omitempty"`
	Provider    Provider `yaml:"provider,omitempty"`
	Tests       []Test   `yaml:"tests"`

	// Metadata (not from YAML)
	Source   Source `yaml:"-"`
	FilePath string `yaml:"-"`
}

// Context specifies which config layers and languages to test against.
type Context struct {
	// Layers specifies which config layers to include.
	// Options: "team", "personal", "project", or "merged" (default).
	Layers []string `yaml:"layers,omitempty"`

	// Languages specifies which language configs to include.
	Languages []string `yaml:"languages,omitempty"`
}

// Provider configures the LLM provider for evals.
type Provider struct {
	// Model is the model identifier, supports env var expansion.
	// Default: ${STAGHORN_EVAL_MODEL:-claude-sonnet-4-20250514}
	Model string `yaml:"model,omitempty"`
}

// Test represents a single test case in an eval.
type Test struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty"`
	Prompt      string            `yaml:"prompt"`
	Assert      []Assertion       `yaml:"assert"`
}

// Assertion represents a test assertion.
type Assertion struct {
	Type  string      `yaml:"type"`
	Value interface{} `yaml:"value"`
}

// Parse parses an eval from YAML content.
func Parse(content string, source Source, filePath string) (*Eval, error) {
	var eval Eval
	if err := yaml.Unmarshal([]byte(content), &eval); err != nil {
		return nil, fmt.Errorf("invalid eval YAML: %w", err)
	}

	if eval.Name == "" {
		return nil, fmt.Errorf("eval must have a 'name' field")
	}

	if len(eval.Tests) == 0 {
		return nil, fmt.Errorf("eval must have at least one test")
	}

	// Validate tests
	for i, test := range eval.Tests {
		if test.Name == "" {
			return nil, fmt.Errorf("test %d must have a 'name' field", i+1)
		}
		if test.Prompt == "" {
			return nil, fmt.Errorf("test '%s' must have a 'prompt' field", test.Name)
		}
		if len(test.Assert) == 0 {
			return nil, fmt.Errorf("test '%s' must have at least one assertion", test.Name)
		}
	}

	eval.Source = source
	eval.FilePath = filePath

	return &eval, nil
}

// ParseFile parses an eval from a file.
func ParseFile(path string, source Source) (*Eval, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read eval file: %w", err)
	}
	return Parse(string(content), source, path)
}

// LoadFromDirectory loads all evals from a directory.
func LoadFromDirectory(dir string, source Source) ([]*Eval, error) {
	var evals []*Eval

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Directory doesn't exist, return empty
		}
		return nil, fmt.Errorf("failed to read evals directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml files
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		eval, err := ParseFile(path, source)
		if err != nil {
			// Log warning but continue loading other evals
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			continue
		}

		evals = append(evals, eval)
	}

	return evals, nil
}

// HasTag checks if the eval has a specific tag.
func (e *Eval) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// HasAnyTag checks if the eval has any of the specified tags.
func (e *Eval) HasAnyTag(tags []string) bool {
	for _, tag := range tags {
		if e.HasTag(tag) {
			return true
		}
	}
	return false
}

// TestCount returns the number of tests in the eval.
func (e *Eval) TestCount() int {
	return len(e.Tests)
}

// ResolveModel returns the model to use, expanding environment variables.
func (e *Eval) ResolveModel() string {
	model := e.Provider.Model
	if model == "" {
		model = "${STAGHORN_EVAL_MODEL:-claude-sonnet-4-20250514}"
	}
	return expandEnvWithDefault(model)
}

// expandEnvWithDefault expands ${VAR:-default} syntax.
func expandEnvWithDefault(s string) string {
	// Handle ${VAR:-default} pattern
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		inner := s[2 : len(s)-1]
		if idx := strings.Index(inner, ":-"); idx != -1 {
			varName := inner[:idx]
			defaultVal := inner[idx+2:]
			if val := os.Getenv(varName); val != "" {
				return val
			}
			return defaultVal
		}
		// Simple ${VAR} case
		return os.Getenv(inner)
	}
	return s
}

// GetEffectiveLayers returns the layers to test against.
func (e *Eval) GetEffectiveLayers() []string {
	if len(e.Context.Layers) == 0 {
		return []string{"merged"}
	}
	return e.Context.Layers
}

// FilterTests returns a copy of the eval with only matching tests.
// testFilter can be a test name or a prefix pattern ending with *.
func (e *Eval) FilterTests(testFilter string) *Eval {
	if testFilter == "" {
		return e
	}

	var filtered []Test
	isPrefix := strings.HasSuffix(testFilter, "*")
	prefix := strings.TrimSuffix(testFilter, "*")

	for _, t := range e.Tests {
		if isPrefix {
			if strings.HasPrefix(t.Name, prefix) {
				filtered = append(filtered, t)
			}
		} else {
			if t.Name == testFilter {
				filtered = append(filtered, t)
			}
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Return a copy with filtered tests
	result := *e
	result.Tests = filtered
	return &result
}
