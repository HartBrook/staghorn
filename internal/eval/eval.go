// Package eval handles staghorn eval definitions, parsing, and execution.
package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidAssertionTypes lists all valid assertion types supported by Promptfoo.
var ValidAssertionTypes = []string{
	"llm-rubric",
	"contains",
	"contains-any",
	"contains-all",
	"not-contains",
	"regex",
	"javascript",
}

// ValidationLevel indicates the severity of a validation issue.
type ValidationLevel string

const (
	ValidationLevelError   ValidationLevel = "error"
	ValidationLevelWarning ValidationLevel = "warning"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	Field   string          // e.g., "tests[0].assert[0].type"
	Message string          // Human-readable error message
	Level   ValidationLevel // error or warning
}

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

// Validate performs detailed validation of the eval and returns any issues found.
// Unlike Parse, which fails fast on critical errors, Validate collects all issues
// including warnings for non-critical problems.
func (e *Eval) Validate() []ValidationError {
	var errors []ValidationError

	// Check eval-level fields
	if e.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "eval must have a name",
			Level:   ValidationLevelError,
		})
	} else if !isValidName(e.Name) {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "name should contain only lowercase letters, numbers, and hyphens",
			Level:   ValidationLevelWarning,
		})
	}

	if e.Description == "" {
		errors = append(errors, ValidationError{
			Field:   "description",
			Message: "eval should have a description",
			Level:   ValidationLevelWarning,
		})
	}

	// Validate tags
	for i, tag := range e.Tags {
		if !isValidTag(tag) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: fmt.Sprintf("tag %q should contain only lowercase letters, numbers, and hyphens", tag),
				Level:   ValidationLevelWarning,
			})
		}
	}

	// Check tests
	if len(e.Tests) == 0 {
		errors = append(errors, ValidationError{
			Field:   "tests",
			Message: "eval must have at least one test",
			Level:   ValidationLevelError,
		})
	}

	for i, test := range e.Tests {
		testErrors := validateTest(test, i)
		errors = append(errors, testErrors...)
	}

	return errors
}

// validateTest validates a single test and returns any issues.
func validateTest(test Test, index int) []ValidationError {
	var errors []ValidationError
	prefix := fmt.Sprintf("tests[%d]", index)

	if test.Name == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".name",
			Message: "test must have a name",
			Level:   ValidationLevelError,
		})
	}

	if test.Description == "" {
		errors = append(errors, ValidationError{
			Field:   prefix,
			Message: fmt.Sprintf("test %q should have a description", test.Name),
			Level:   ValidationLevelWarning,
		})
	}

	if test.Prompt == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".prompt",
			Message: fmt.Sprintf("test %q must have a prompt", test.Name),
			Level:   ValidationLevelError,
		})
	} else if strings.TrimSpace(test.Prompt) == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".prompt",
			Message: fmt.Sprintf("test %q has an empty prompt (whitespace only)", test.Name),
			Level:   ValidationLevelError,
		})
	}

	if len(test.Assert) == 0 {
		errors = append(errors, ValidationError{
			Field:   prefix + ".assert",
			Message: fmt.Sprintf("test %q must have at least one assertion", test.Name),
			Level:   ValidationLevelError,
		})
	}

	for j, assertion := range test.Assert {
		assertErrors := validateAssertion(assertion, prefix, j)
		errors = append(errors, assertErrors...)
	}

	return errors
}

// validateAssertion validates a single assertion and returns any issues.
func validateAssertion(assertion Assertion, testPrefix string, index int) []ValidationError {
	var errors []ValidationError
	field := fmt.Sprintf("%s.assert[%d]", testPrefix, index)

	if assertion.Type == "" {
		errors = append(errors, ValidationError{
			Field:   field + ".type",
			Message: "assertion must have a type",
			Level:   ValidationLevelError,
		})
		return errors
	}

	if !isValidAssertionType(assertion.Type) {
		suggestion := suggestAssertionType(assertion.Type)
		msg := fmt.Sprintf("invalid assertion type %q", assertion.Type)
		if suggestion != "" {
			msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
		}
		errors = append(errors, ValidationError{
			Field:   field + ".type",
			Message: msg,
			Level:   ValidationLevelError,
		})
	}

	if assertion.Value == nil {
		errors = append(errors, ValidationError{
			Field:   field + ".value",
			Message: "assertion must have a value",
			Level:   ValidationLevelError,
		})
	}

	return errors
}

// isValidAssertionType checks if the assertion type is valid.
func isValidAssertionType(t string) bool {
	for _, valid := range ValidAssertionTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// suggestAssertionType suggests a valid assertion type based on the invalid input.
func suggestAssertionType(invalid string) string {
	// Normalize input for comparison
	normalized := strings.ToLower(strings.ReplaceAll(invalid, "_", "-"))

	// Check for exact match after normalization
	for _, valid := range ValidAssertionTypes {
		if normalized == valid {
			return valid
		}
	}

	// Check for partial matches
	for _, valid := range ValidAssertionTypes {
		if strings.Contains(normalized, strings.ReplaceAll(valid, "-", "")) ||
			strings.Contains(strings.ReplaceAll(valid, "-", ""), normalized) {
			return valid
		}
	}

	// Check for common typos
	typoMap := map[string]string{
		"llm_rubric":   "llm-rubric",
		"llmrubric":    "llm-rubric",
		"rubric":       "llm-rubric",
		"contain":      "contains",
		"notcontains":  "not-contains",
		"not_contains": "not-contains",
		"containsall":  "contains-all",
		"containsany":  "contains-any",
		"contains_all": "contains-all",
		"contains_any": "contains-any",
		"js":           "javascript",
		"regexp":       "regex",
	}

	if suggestion, ok := typoMap[normalized]; ok {
		return suggestion
	}

	return ""
}

// isValidName checks if a name follows naming conventions.
func isValidName(name string) bool {
	// Allow lowercase letters, numbers, and hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`, name)
	return matched
}

// isValidTag checks if a tag follows naming conventions.
func isValidTag(tag string) bool {
	// Allow lowercase letters, numbers, and hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`, tag)
	return matched
}

// HasErrors returns true if there are any errors (not just warnings).
func HasErrors(errors []ValidationError) bool {
	for _, err := range errors {
		if err.Level == ValidationLevelError {
			return true
		}
	}
	return false
}

// CountByLevel counts errors and warnings separately.
func CountByLevel(errors []ValidationError) (errorCount, warningCount int) {
	for _, err := range errors {
		if err.Level == ValidationLevelError {
			errorCount++
		} else {
			warningCount++
		}
	}
	return
}
