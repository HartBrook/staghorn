package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Go aliases
		{"golang", "go"},
		{"Golang", "go"},
		{"GOLANG", "go"},
		{"go", "go"},

		// Python aliases
		{"python3", "python"},
		{"python2", "python"},
		{"py", "python"},
		{"py3", "python"},
		{"Python", "python"},

		// JavaScript aliases
		{"js", "javascript"},
		{"node", "javascript"},
		{"nodejs", "javascript"},
		{"ecmascript", "javascript"},

		// TypeScript aliases
		{"ts", "typescript"},
		{"TypeScript", "typescript"},

		// Ruby aliases
		{"rb", "ruby"},

		// Rust aliases
		{"rs", "rust"},

		// C++ aliases
		{"cpp", "c++"},
		{"cplusplus", "c++"},

		// C# aliases
		{"csharp", "c#"},
		{"cs", "c#"},
		{"dotnet", "c#"},

		// Kotlin aliases
		{"kt", "kotlin"},

		// Shell/Bash aliases
		{"shell", "bash"},
		{"sh", "bash"},
		{"zsh", "bash"},

		// YAML aliases
		{"yml", "yaml"},

		// No alias - returns as-is (lowercased)
		{"java", "java"},
		{"swift", "swift"},
		{"RUST", "rust"},
		{"bash", "bash"},
		{"yaml", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeLanguage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterByLanguage(t *testing.T) {
	results := []SearchResult{
		{Owner: "acme", Repo: "python-config", Topics: []string{"staghorn-config", "python"}},
		{Owner: "acme", Repo: "go-config", Topics: []string{"staghorn-config", "go", "golang"}},
		{Owner: "acme", Repo: "js-config", Topics: []string{"staghorn-config", "javascript", "typescript"}},
		{Owner: "acme", Repo: "multi-config", Topics: []string{"staghorn-config", "python", "go", "rust"}},
	}

	tests := []struct {
		name     string
		lang     string
		expected []string // expected repo names
	}{
		{
			name:     "filter by python",
			lang:     "python",
			expected: []string{"python-config", "multi-config"},
		},
		{
			name:     "filter by py alias",
			lang:     "py",
			expected: []string{"python-config", "multi-config"},
		},
		{
			name:     "filter by python3 alias",
			lang:     "python3",
			expected: []string{"python-config", "multi-config"},
		},
		{
			name:     "filter by go",
			lang:     "go",
			expected: []string{"go-config", "multi-config"},
		},
		{
			name:     "filter by golang alias",
			lang:     "golang",
			expected: []string{"go-config", "multi-config"},
		},
		{
			name:     "filter by javascript",
			lang:     "javascript",
			expected: []string{"js-config"},
		},
		{
			name:     "filter by js alias",
			lang:     "js",
			expected: []string{"js-config"},
		},
		{
			name:     "filter by typescript",
			lang:     "typescript",
			expected: []string{"js-config"},
		},
		{
			name:     "filter by ts alias",
			lang:     "ts",
			expected: []string{"js-config"},
		},
		{
			name:     "empty filter returns all",
			lang:     "",
			expected: []string{"python-config", "go-config", "js-config", "multi-config"},
		},
		{
			name:     "no matches",
			lang:     "haskell",
			expected: []string{},
		},
		{
			name:     "case insensitive",
			lang:     "PYTHON",
			expected: []string{"python-config", "multi-config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterByLanguage(results, tt.lang)
			repos := make([]string, len(filtered))
			for i, r := range filtered {
				repos[i] = r.Repo
			}
			assert.Equal(t, tt.expected, repos)
		})
	}
}

func TestFilterByTag(t *testing.T) {
	results := []SearchResult{
		{Owner: "acme", Repo: "security-config", Topics: []string{"staghorn-config", "security", "audit"}},
		{Owner: "acme", Repo: "web-config", Topics: []string{"staghorn-config", "web", "frontend"}},
		{Owner: "acme", Repo: "general-config", Topics: []string{"staghorn-config"}},
	}

	tests := []struct {
		name     string
		tag      string
		expected []string
	}{
		{
			name:     "filter by security",
			tag:      "security",
			expected: []string{"security-config"},
		},
		{
			name:     "filter by web",
			tag:      "web",
			expected: []string{"web-config"},
		},
		{
			name:     "empty filter returns all",
			tag:      "",
			expected: []string{"security-config", "web-config", "general-config"},
		},
		{
			name:     "case insensitive",
			tag:      "SECURITY",
			expected: []string{"security-config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterByTag(results, tt.tag)
			repos := make([]string, len(filtered))
			for i, r := range filtered {
				repos[i] = r.Repo
			}
			assert.Equal(t, tt.expected, repos)
		})
	}
}

func TestSortByStars(t *testing.T) {
	results := []SearchResult{
		{Repo: "low", Stars: 10},
		{Repo: "high", Stars: 1000},
		{Repo: "medium", Stars: 100},
	}

	SortByStars(results)

	assert.Equal(t, "high", results[0].Repo)
	assert.Equal(t, "medium", results[1].Repo)
	assert.Equal(t, "low", results[2].Repo)
}

func TestSearchResult_FullName(t *testing.T) {
	r := SearchResult{Owner: "acme", Repo: "config"}
	assert.Equal(t, "acme/config", r.FullName())
}
