package cli

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncProducesProvenanceComments(t *testing.T) {
	// This test verifies that sync produces output with provenance comments
	// by testing the merge options used in applyConfig

	teamContent := `## Code Style

Follow team standards.`

	personalContent := `## Code Style

My personal preferences.

## My Preferences

Personal section.`

	layers := []merge.Layer{
		{Content: teamContent, Source: "team"},
		{Content: personalContent, Source: "personal"},
	}

	// These are the same options used in applyConfig
	mergeOpts := merge.MergeOptions{
		AnnotateSources: true,
		SourceRepo:      "acme/standards",
	}

	result := merge.MergeWithLanguages(layers, mergeOpts)

	// Verify provenance comments are present
	if !strings.Contains(result, "<!-- staghorn:source:team -->") {
		t.Error("merged output should contain team source marker")
	}
	if !strings.Contains(result, "<!-- staghorn:source:personal -->") {
		t.Error("merged output should contain personal source marker")
	}

	// Verify content is still present
	if !strings.Contains(result, "Follow team standards") {
		t.Error("merged output should contain team content")
	}
	if !strings.Contains(result, "My personal preferences") {
		t.Error("merged output should contain personal content")
	}
}

func TestSyncProvenanceForNewSections(t *testing.T) {
	// Test that new sections added by personal layer have provenance
	teamContent := `## Code Style

Team rules.`

	personalContent := `## My Preferences

Personal only section.`

	layers := []merge.Layer{
		{Content: teamContent, Source: "team"},
		{Content: personalContent, Source: "personal"},
	}

	mergeOpts := merge.MergeOptions{
		AnnotateSources: true,
	}

	result := merge.MergeWithLanguages(layers, mergeOpts)

	// The "My Preferences" section should have personal source markers
	if !strings.Contains(result, "## My Preferences") {
		t.Error("merged output should contain personal section")
	}

	// Find the personal section and verify it has markers
	personalIdx := strings.Index(result, "## My Preferences")
	if personalIdx == -1 {
		t.Fatal("could not find personal section")
	}

	// There should be a personal source marker before the personal section
	beforePersonal := result[:personalIdx]
	if !strings.Contains(beforePersonal, "<!-- staghorn:source:personal -->") {
		t.Error("personal section should be preceded by personal source marker")
	}
}

func TestSyncProvenanceMarkerFormat(t *testing.T) {
	// Verify the exact format of provenance markers for parsing
	teamContent := `## Test

Content.`

	layers := []merge.Layer{
		{Content: teamContent, Source: "team"},
	}

	mergeOpts := merge.MergeOptions{
		AnnotateSources: true,
	}

	result := merge.MergeWithLanguages(layers, mergeOpts)

	// Markers should be machine-parseable
	if !strings.Contains(result, "<!-- staghorn:source:team -->") {
		t.Errorf("source marker format incorrect, got:\n%s", result)
	}
}

func TestStripInstructionalComments(t *testing.T) {
	// The function strips comments in the format <!-- [staghorn] ... -->
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no comments",
			input:    "# Title\n\nContent here.",
			expected: "# Title\n\nContent here.",
		},
		{
			name:     "single line instructional comment",
			input:    "# Title\n<!-- [staghorn] this is a hint -->\n\nContent.",
			expected: "# Title\n\nContent.",
		},
		{
			name:     "preserves non-staghorn comments",
			input:    "# Title\n<!-- regular comment -->\n\nContent.",
			expected: "# Title\n<!-- regular comment -->\n\nContent.",
		},
		{
			name:     "preserves provenance comments",
			input:    "<!-- staghorn:source:team -->\n# Title",
			expected: "<!-- staghorn:source:team -->\n# Title",
		},
		{
			name:     "multiple instructional comments",
			input:    "<!-- [staghorn] first -->\n# Title\n<!-- [staghorn] second -->\nContent.",
			expected: "# Title\nContent.",
		},
		{
			name:     "collapses consecutive blank lines",
			input:    "# Title\n\n\n\nContent.",
			expected: "# Title\n\nContent.",
		},
		{
			name:     "strips leading blank lines",
			input:    "\n\n# Title\nContent.",
			expected: "# Title\nContent.",
		},
		{
			name:     "strips trailing blank lines",
			input:    "# Title\nContent.\n\n",
			expected: "# Title\nContent.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripInstructionalComments(tt.input)
			if result != tt.expected {
				t.Errorf("stripInstructionalComments() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestApplyConfigIntegration(t *testing.T) {
	// Integration test that verifies applyConfig produces provenance markers
	// This requires setting up the full directory structure

	// Create temp home directory
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create config directory structure
	configDir := filepath.Join(tempHome, ".config", "staghorn")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Create cache directory structure
	cacheDir := filepath.Join(tempHome, ".cache", "staghorn")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}

	// Create team cache file (uses owner-repo.md format based on CacheFile method)
	owner, repo := "testorg", "testrepo"
	teamContent := `## Code Style

Team guidelines here.

## Testing

Write tests for everything.`
	teamCacheFile := filepath.Join(cacheDir, owner+"-"+repo+".md")
	if err := os.WriteFile(teamCacheFile, []byte(teamContent), 0644); err != nil {
		t.Fatalf("failed to write team cache: %v", err)
	}

	// Create personal config
	personalContent := `## Code Style

My personal style preferences.

## My Preferences

Custom section.`
	if err := os.WriteFile(filepath.Join(configDir, "personal.md"), []byte(personalContent), 0644); err != nil {
		t.Fatalf("failed to write personal config: %v", err)
	}

	// Create config with Source struct
	cfg := &config.Config{
		Source: config.Source{Simple: owner + "/" + repo},
	}

	// Create paths with overrides
	paths := config.NewPathsWithOverrides(configDir, cacheDir)

	// Run applyConfig
	if err := applyConfig(cfg, paths, owner, repo); err != nil {
		t.Fatalf("applyConfig failed: %v", err)
	}

	// Read the output
	outputPath := filepath.Join(tempHome, ".claude", "CLAUDE.md")
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	outputStr := string(output)

	// Verify header
	if !strings.Contains(outputStr, "Managed by staghorn") {
		t.Error("output should contain staghorn header")
	}

	// Verify provenance markers
	if !strings.Contains(outputStr, "<!-- staghorn:source:team -->") {
		t.Error("output should contain team source marker")
	}
	if !strings.Contains(outputStr, "<!-- staghorn:source:personal -->") {
		t.Error("output should contain personal source marker")
	}

	// Verify content from both layers
	if !strings.Contains(outputStr, "Team guidelines here") {
		t.Error("output should contain team content")
	}
	if !strings.Contains(outputStr, "My personal style preferences") {
		t.Error("output should contain personal content")
	}
	if !strings.Contains(outputStr, "Custom section") {
		t.Error("output should contain personal-only section")
	}
}

// Multi-source tests

func TestMultiSourceConfigDetection(t *testing.T) {
	tests := []struct {
		name          string
		source        config.Source
		isMultiSource bool
	}{
		{
			name:          "simple string source",
			source:        config.Source{Simple: "acme/standards"},
			isMultiSource: false,
		},
		{
			name: "multi-source with languages",
			source: config.Source{
				Multi: &config.SourceConfig{
					Default: "acme/standards",
					Languages: map[string]string{
						"python": "community/python-standards",
					},
				},
			},
			isMultiSource: true,
		},
		{
			name: "multi-source with commands",
			source: config.Source{
				Multi: &config.SourceConfig{
					Default: "acme/standards",
					Commands: map[string]string{
						"security-audit": "security/audits",
					},
				},
			},
			isMultiSource: true,
		},
		{
			name: "multi-source with base override",
			source: config.Source{
				Multi: &config.SourceConfig{
					Default: "acme/standards",
					Base:    "acme/base-config",
				},
			},
			isMultiSource: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.IsMultiSource()
			assert.Equal(t, tt.isMultiSource, result, "IsMultiSource() mismatch")
		})
	}
}

func TestMultiSourceRepoResolution(t *testing.T) {
	source := config.Source{
		Multi: &config.SourceConfig{
			Default: "acme/standards",
			Base:    "acme/base-config",
			Languages: map[string]string{
				"python": "community/python-standards",
				"go":     "acme/go-standards",
			},
			Commands: map[string]string{
				"security-audit": "security/audits",
			},
		},
	}

	tests := []struct {
		name     string
		method   string
		arg      string
		expected string
	}{
		{"default repo", "default", "", "acme/standards"},
		{"base repo", "base", "", "acme/base-config"},
		{"python language", "language", "python", "community/python-standards"},
		{"go language", "language", "go", "acme/go-standards"},
		{"rust language (fallback)", "language", "rust", "acme/standards"},
		{"security-audit command", "command", "security-audit", "security/audits"},
		{"code-review command (fallback)", "command", "code-review", "acme/standards"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			switch tt.method {
			case "default":
				result = source.DefaultRepo()
			case "base":
				result = source.RepoForBase()
			case "language":
				result = source.RepoForLanguage(tt.arg)
			case "command":
				result = source.RepoForCommand(tt.arg)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMultiSourceAllRepos(t *testing.T) {
	source := config.Source{
		Multi: &config.SourceConfig{
			Default: "acme/standards",
			Base:    "acme/base-config",
			Languages: map[string]string{
				"python": "community/python-standards",
				"go":     "acme/standards", // duplicate of default
			},
			Commands: map[string]string{
				"security-audit": "security/audits",
			},
		},
	}

	repos := source.AllRepos()

	// Should have 4 unique repos (acme/standards appears twice but deduplicated)
	require.Len(t, repos, 4, "AllRepos() should return 4 unique repos")

	// Sort for consistent comparison
	sort.Strings(repos)
	expected := []string{"acme/base-config", "acme/standards", "community/python-standards", "security/audits"}
	sort.Strings(expected)

	assert.Equal(t, expected, repos)
}

func TestApplyMultiSourceConfigIntegration(t *testing.T) {
	// Integration test for multi-source config merging
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	err := os.Setenv("HOME", tempHome)
	require.NoError(t, err, "failed to set HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create directory structure
	configDir := filepath.Join(tempHome, ".config", "staghorn")
	cacheDir := filepath.Join(tempHome, ".cache", "staghorn")
	for _, dir := range []string{configDir, cacheDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "failed to create dir")
	}

	// Setup: base config from acme/base-config
	baseContent := `## Core Principles

Base principles here.`
	baseCacheFile := filepath.Join(cacheDir, "acme-base-config.md")
	err = os.WriteFile(baseCacheFile, []byte(baseContent), 0644)
	require.NoError(t, err, "failed to write base cache")

	// Setup: python language from community/python-standards
	pythonLangDir := filepath.Join(cacheDir, "community-python-standards-languages")
	err = os.MkdirAll(pythonLangDir, 0755)
	require.NoError(t, err, "failed to create python lang dir")
	pythonContent := `## Python Guidelines

Use type hints.`
	err = os.WriteFile(filepath.Join(pythonLangDir, "python.md"), []byte(pythonContent), 0644)
	require.NoError(t, err, "failed to write python config")

	// Setup: go language from default (acme/standards)
	goLangDir := filepath.Join(cacheDir, "acme-standards-languages")
	err = os.MkdirAll(goLangDir, 0755)
	require.NoError(t, err, "failed to create go lang dir")
	goContent := `## Go Guidelines

Use gofmt.`
	err = os.WriteFile(filepath.Join(goLangDir, "go.md"), []byte(goContent), 0644)
	require.NoError(t, err, "failed to write go config")

	// Create multi-source config
	cfg := &config.Config{
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: "acme/standards",
				Base:    "acme/base-config",
				Languages: map[string]string{
					"python": "community/python-standards",
				},
			},
		},
		Languages: config.LanguageConfig{
			Enabled: []string{"python", "go"},
		},
	}

	paths := config.NewPathsWithOverrides(configDir, cacheDir)

	// Create repo contexts
	repoContexts := map[string]*repoContext{
		"acme/standards":             {owner: "acme", repo: "standards", branch: "main"},
		"acme/base-config":           {owner: "acme", repo: "base-config", branch: "main"},
		"community/python-standards": {owner: "community", repo: "python-standards", branch: "main"},
	}

	// Run applyConfigFromMultiSource
	err = applyConfigFromMultiSource(cfg, paths, repoContexts)
	require.NoError(t, err, "applyConfigFromMultiSource failed")

	// Read output
	outputPath := filepath.Join(tempHome, ".claude", "CLAUDE.md")
	output, err := os.ReadFile(outputPath)
	require.NoError(t, err, "failed to read output")

	outputStr := string(output)

	// Verify base content is present
	assert.Contains(t, outputStr, "Base principles here", "output should contain base config content")

	// Verify python content from community repo
	assert.Contains(t, outputStr, "Use type hints", "output should contain python content from community repo")

	// Verify go content from default repo
	assert.Contains(t, outputStr, "Use gofmt", "output should contain go content from default repo")

	// Verify header
	assert.Contains(t, outputStr, "Managed by staghorn", "output should contain staghorn header")
}
