package optimize

import (
	"regexp"
	"sort"
	"strings"
)

// AnchorCategory indicates how strict validation should be for an anchor type.
type AnchorCategory int

const (
	// AnchorStrict means missing anchors should fail validation (file paths, commands).
	AnchorStrict AnchorCategory = iota
	// AnchorSoft means missing anchors generate warnings but don't fail (tool names).
	AnchorSoft
)

// CategorizedAnchors holds anchors grouped by their validation strictness.
type CategorizedAnchors struct {
	Strict []string // File paths, commands - must be preserved
	Soft   []string // Tool names - warn if missing but don't fail
}

// All returns all anchors as a flat list (for backward compatibility).
func (c *CategorizedAnchors) All() []string {
	all := make([]string, 0, len(c.Strict)+len(c.Soft))
	all = append(all, c.Strict...)
	all = append(all, c.Soft...)
	sort.Strings(all)
	return all
}

// ExtractCategorizedAnchors finds critical content and categorizes by validation strictness.
func ExtractCategorizedAnchors(content string) *CategorizedAnchors {
	result := &CategorizedAnchors{}
	seen := make(map[string]bool)

	// Soft anchors: tool names (LLM may consolidate/rephrase these)
	for _, a := range extractToolNames(content) {
		if !seen[a] {
			result.Soft = append(result.Soft, a)
			seen[a] = true
		}
	}

	// Strict anchors: file paths (project-specific, must preserve exactly)
	for _, a := range extractFilePaths(content) {
		if !seen[a] {
			result.Strict = append(result.Strict, a)
			seen[a] = true
		}
	}

	// Strict anchors: shell commands (must preserve exactly)
	for _, a := range extractCommands(content) {
		if !seen[a] {
			result.Strict = append(result.Strict, a)
			seen[a] = true
		}
	}

	// Strict anchors: function/class names from code blocks
	for _, a := range extractCodeBlockAnchors(content) {
		if !seen[a] {
			result.Strict = append(result.Strict, a)
			seen[a] = true
		}
	}

	sort.Strings(result.Strict)
	sort.Strings(result.Soft)

	return result
}

// ExtractAnchors finds critical content that must be preserved during optimization.
// This includes tool names, file paths, commands, and other specific identifiers.
// For categorized extraction with different validation strictness, use ExtractCategorizedAnchors.
func ExtractAnchors(content string) []string {
	return ExtractCategorizedAnchors(content).All()
}

// ValidationResult contains the results of anchor validation.
type ValidationResult struct {
	Preserved     []string // Anchors that were preserved
	MissingStrict []string // Strict anchors that are missing (should fail)
	MissingSoft   []string // Soft anchors that are missing (warnings only)
}

// HasStrictFailures returns true if any strict anchors are missing.
func (v *ValidationResult) HasStrictFailures() bool {
	return len(v.MissingStrict) > 0
}

// AllMissing returns all missing anchors (for backward compatibility).
func (v *ValidationResult) AllMissing() []string {
	all := make([]string, 0, len(v.MissingStrict)+len(v.MissingSoft))
	all = append(all, v.MissingStrict...)
	all = append(all, v.MissingSoft...)
	sort.Strings(all)
	return all
}

// ValidateAnchorsCategorized checks anchors with different strictness levels.
func ValidateAnchorsCategorized(original, optimized string) *ValidationResult {
	anchors := ExtractCategorizedAnchors(original)
	optimizedLower := strings.ToLower(optimized)

	result := &ValidationResult{}

	// Check strict anchors
	for _, anchor := range anchors.Strict {
		if strings.Contains(optimizedLower, strings.ToLower(anchor)) {
			result.Preserved = append(result.Preserved, anchor)
		} else {
			result.MissingStrict = append(result.MissingStrict, anchor)
		}
	}

	// Check soft anchors
	for _, anchor := range anchors.Soft {
		if strings.Contains(optimizedLower, strings.ToLower(anchor)) {
			result.Preserved = append(result.Preserved, anchor)
		} else {
			result.MissingSoft = append(result.MissingSoft, anchor)
		}
	}

	return result
}

// ValidateAnchors checks if all anchors from the original content exist in optimized content.
// Returns (preserved, missing) anchor lists.
// For categorized validation with different strictness, use ValidateAnchorsCategorized.
func ValidateAnchors(original, optimized string) ([]string, []string) {
	result := ValidateAnchorsCategorized(original, optimized)
	return result.Preserved, result.AllMissing()
}

// Common development tools to look for
var knownTools = []string{
	// Python
	"pytest", "ruff", "black", "isort", "mypy", "pyright", "flake8", "pylint",
	"poetry", "pip", "pipenv", "uv", "pydantic", "fastapi", "django", "flask",
	// Go
	"gofmt", "goimports", "golangci-lint", "go test", "go build", "go mod",
	// JavaScript/TypeScript
	"npm", "yarn", "pnpm", "bun", "eslint", "prettier", "biome", "vitest",
	"jest", "webpack", "vite", "rollup", "esbuild", "tsc", "typescript",
	// Rust
	"cargo", "rustfmt", "clippy",
	// General
	"git", "docker", "make", "bash", "zsh", "curl", "wget",
}

// extractToolNames finds known tool names in content.
func extractToolNames(content string) []string {
	var found []string
	contentLower := strings.ToLower(content)

	for _, tool := range knownTools {
		if strings.Contains(contentLower, strings.ToLower(tool)) {
			found = append(found, tool)
		}
	}

	return found
}

// extractFilePaths finds file paths in content.
func extractFilePaths(content string) []string {
	var found []string

	// Match paths like /path/to/file, ./relative/path, ~/home/path
	// Also matches common config files like .gitignore, .env
	pathPatterns := []*regexp.Regexp{
		// Absolute paths: /foo/bar (must start with letter after /)
		regexp.MustCompile(`(?:^|[\s"'\(])(/[a-zA-Z][a-zA-Z0-9_\-./]*)`),
		// Relative paths: ./foo or ../foo
		regexp.MustCompile(`(?:^|[\s"'\(])(\.\./[a-zA-Z0-9_\-./]+|\./[a-zA-Z0-9_\-./]+)`),
		// Home paths: ~/foo
		regexp.MustCompile(`(?:^|[\s"'\(])(~/[a-zA-Z0-9_\-./]+)`),
		// Dotfiles: .gitignore, .env, .eslintrc (must have at least 2 letters)
		regexp.MustCompile(`(?:^|[\s"'\(])(\.[a-zA-Z][a-zA-Z0-9_\-]+(?:\.[a-zA-Z]+)?)`),
		// Common config patterns: must start with letter, have meaningful name
		regexp.MustCompile(`(?:^|[\s"'\(])([a-zA-Z][a-zA-Z0-9_\-]*\.(?:yaml|yml|json|toml|md|txt|py|go|ts|js|rs))`),
	}

	seen := make(map[string]bool)
	for _, pattern := range pathPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := strings.Trim(match[1], `"'`)
				// Filter out common false positives
				if isValidPath(path) && !seen[path] {
					found = append(found, path)
					seen[path] = true
				}
			}
		}
	}

	return found
}

// isValidPath checks if a string looks like a valid path.
func isValidPath(s string) bool {
	// Too short
	if len(s) < 2 {
		return false
	}

	// Filter out version numbers like .1, .2, v1.0.0
	if matched, _ := regexp.MatchString(`^\.\d+$`, s); matched {
		return false
	}
	if matched, _ := regexp.MatchString(`^v?\d+\.\d+`, s); matched {
		return false
	}

	// Filter out common non-path patterns
	invalidPrefixes := []string{"..", ".0", ".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8", ".9"}
	for _, prefix := range invalidPrefixes {
		if s == prefix {
			return false
		}
	}

	// Filter out pure numeric filenames like 123.json
	if matched, _ := regexp.MatchString(`^\d+\.[a-z]+$`, s); matched {
		return false
	}

	return true
}

// extractCommands finds shell commands in content.
func extractCommands(content string) []string {
	var found []string
	seen := make(map[string]bool)

	// Match commands in backticks: `command arg`
	backtickPattern := regexp.MustCompile("`([^`]+)`")
	matches := backtickPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			cmd := strings.TrimSpace(match[1])
			// Filter to likely commands (starts with known command or contains common patterns)
			if looksLikeCommand(cmd) && !seen[cmd] {
				found = append(found, cmd)
				seen[cmd] = true
			}
		}
	}

	return found
}

// looksLikeCommand checks if a string appears to be a shell command.
func looksLikeCommand(s string) bool {
	// Must have some content
	if len(s) < 2 || len(s) > 100 {
		return false
	}

	sLower := strings.ToLower(s)

	// Get first word
	fields := strings.Fields(sLower)
	if len(fields) == 0 {
		return false
	}
	firstWord := fields[0]

	// Check if first word is a known tool
	for _, tool := range knownTools {
		toolLower := strings.ToLower(tool)
		if firstWord == toolLower {
			return true
		}
	}

	// Common command prefixes (for multi-word commands)
	cmdPrefixes := []string{
		"npm ", "yarn ", "pnpm ", "bun ",
		"go ", "cargo ",
		"python ", "pip ", "uv ",
		"git ", "docker ",
		"make", "bash ", "sh ",
		"curl ", "wget ",
	}

	for _, prefix := range cmdPrefixes {
		if strings.HasPrefix(sLower, prefix) {
			return true
		}
	}

	return false
}

// genericIdentifiers are common variable names that appear in code examples
// but are not critical to preserve (they're illustrative, not project-specific).
var genericIdentifiers = map[string]bool{
	// Common variable names
	"config": true, "cfg": true, "conf": true, "settings": true, "options": true, "opts": true,
	"data": true, "result": true, "results": true, "response": true, "res": true, "resp": true,
	"value": true, "val": true, "item": true, "items": true, "list": true, "arr": true,
	"input": true, "output": true, "params": true, "args": true, "props": true,
	"user": true, "users": true, "name": true, "id": true, "key": true, "keys": true,
	"err": true, "error": true, "e": true, "ex": true, "msg": true, "message": true,
	"ctx": true, "context": true, "req": true, "request": true,
	"i": true, "j": true, "k": true, "n": true, "x": true, "y": true, "z": true,
	"a": true, "b": true, "c": true, "s": true, "t": true, "v": true, "w": true,
	"tmp": true, "temp": true, "foo": true, "bar": true, "baz": true,
	// Common test identifiers
	"got": true, "want": true, "expected": true, "actual": true, "tt": true, "tc": true,
	"tests": true, "test": true,
}

// extractCodeBlockAnchors finds important identifiers in fenced code blocks.
// It only extracts function and class names, not variable declarations,
// since variables in code examples are typically generic and illustrative.
func extractCodeBlockAnchors(content string) []string {
	var found []string

	// Match fenced code blocks
	codeBlockPattern := regexp.MustCompile("(?s)```[a-zA-Z]*\n(.*?)```")
	matches := codeBlockPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			codeContent := match[1]

			// Extract function/class definitions only (not variable declarations)
			// Variable names in examples are typically generic and illustrative
			defPatterns := []*regexp.Regexp{
				// Python: def func_name, class ClassName
				regexp.MustCompile(`(?:def|class)\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
				// Go: func FuncName
				regexp.MustCompile(`func\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
				// TypeScript/JavaScript: function funcName (explicit function keyword only)
				regexp.MustCompile(`function\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
			}

			for _, pattern := range defPatterns {
				defMatches := pattern.FindAllStringSubmatch(codeContent, -1)
				for _, m := range defMatches {
					if len(m) > 1 {
						name := m[1]
						// Skip generic identifiers that are common in examples
						if !genericIdentifiers[strings.ToLower(name)] {
							found = append(found, name)
						}
					}
				}
			}
		}
	}

	return found
}
