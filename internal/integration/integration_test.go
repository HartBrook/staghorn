package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestdataDir returns the path to the testdata directory.
func getTestdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "fixtures")
}

// TestIntegration_Fixtures runs all fixture-based integration tests.
func TestIntegration_Fixtures(t *testing.T) {
	fixturesDir := getTestdataDir()

	// Check if fixtures directory exists
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("fixtures directory not found")
	}

	fixtures, err := LoadAllFixtures(fixturesDir)
	require.NoError(t, err, "failed to load fixtures")

	if len(fixtures) == 0 {
		t.Skip("no fixtures found")
	}

	for _, fixture := range fixtures {
		fixture := fixture // capture for parallel execution
		t.Run(fixture.Name, func(t *testing.T) {
			t.Parallel()
			runFixture(t, fixture)
		})
	}
}

// runFixture executes a single fixture test.
func runFixture(t *testing.T, fixture *Fixture) {
	t.Helper()

	// Validate fixture has required fields
	require.NotNil(t, fixture.Setup.Team, "fixture must have team setup")
	require.NotNil(t, fixture.Setup.Config, "fixture must have config setup")

	// Create isolated environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Apply setup
	err := ApplySetup(env, fixture.Setup)
	require.NoError(t, err, "failed to apply setup")

	// Get owner/repo from config
	owner, repo := parseOwnerRepo(fixture.Setup.Team.Source)

	// Get config
	cfg := fixture.Setup.Config.ToConfig()

	// Run sync
	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err, "RunSync failed")

	// Check output exists
	if fixture.Assertions.OutputExists {
		_, err := os.Stat(env.GetOutputPath())
		require.NoError(t, err, "output file should exist")
	}

	// Read output and run assertions
	output, err := env.ReadOutput()
	require.NoError(t, err, "failed to read output")

	asserter := NewAsserter(t, output)
	asserter.RunAssertions(fixture.Assertions)
}

// TestIntegration_BasicSync tests basic team config sync without fixtures.
func TestIntegration_BasicSync(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"
	teamContent := `## Code Style

Follow team standards.

## Testing

Write tests for everything.`

	err := env.SetupTeamConfig(owner, repo, teamContent)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify header
	assert.True(t, asserter.HasManagedHeader(), "should have managed header")
	assert.True(t, asserter.HasSourceRepo(owner+"/"+repo), "should have source repo in header")

	// Verify provenance
	assert.True(t, asserter.HasProvenanceMarker("team"), "should have team marker")

	// Verify content
	assert.True(t, asserter.ContainsText("Follow team standards"), "should contain team content")
	assert.True(t, asserter.ContainsText("Write tests for everything"), "should contain team content")
}

// TestIntegration_TeamPlusPersonal tests merging team and personal configs.
func TestIntegration_TeamPlusPersonal(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"
	teamContent := `## Code Style

Follow team standards.`

	personalContent := `## Code Style

My personal preferences.

## My Preferences

Custom section.`

	err := env.SetupTeamConfig(owner, repo, teamContent)
	require.NoError(t, err)

	err = env.SetupPersonalConfig(personalContent)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify both markers present
	assert.True(t, asserter.HasProvenanceMarker("team"), "should have team marker")
	assert.True(t, asserter.HasProvenanceMarker("personal"), "should have personal marker")

	// Verify provenance order
	order := asserter.ProvenanceOrder()
	assert.Equal(t, []string{"team", "personal"}, order, "team should come before personal")

	// Verify content from both layers
	assert.True(t, asserter.ContainsText("Follow team standards"), "should contain team content")
	assert.True(t, asserter.ContainsText("My personal preferences"), "should contain personal content")
	assert.True(t, asserter.ContainsText("Custom section"), "should contain personal-only section")
}

// TestIntegration_WithLanguages tests language config merging.
func TestIntegration_WithLanguages(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"
	teamContent := `## Code Style

Team rules.`

	teamPython := `## Python Guidelines

Use type hints.`

	personalPython := `## My Python Preferences

Prefer dataclasses.`

	err := env.SetupTeamConfig(owner, repo, teamContent)
	require.NoError(t, err)

	err = env.SetupTeamLanguage(owner, repo, "python", teamPython)
	require.NoError(t, err)

	err = env.SetupPersonalLanguage("python", personalPython)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify language markers
	assert.True(t, strings.Contains(output, "<!-- staghorn:source:team:python -->"), "should have team python marker")
	assert.True(t, strings.Contains(output, "<!-- staghorn:source:personal:python -->"), "should have personal python marker")

	// Verify language content
	assert.True(t, asserter.ContainsText("Use type hints"), "should contain team python content")
	assert.True(t, asserter.ContainsText("Prefer dataclasses"), "should contain personal python content")
}

// TestIntegration_EmptyPersonal tests sync with no personal config.
func TestIntegration_EmptyPersonal(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"
	teamContent := `## Code Style

Team rules only.`

	err := env.SetupTeamConfig(owner, repo, teamContent)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Should have team marker but no personal marker
	assert.True(t, asserter.HasProvenanceMarker("team"), "should have team marker")
	assert.False(t, asserter.HasProvenanceMarker("personal"), "should NOT have personal marker")

	// Verify content
	assert.True(t, asserter.ContainsText("Team rules only"), "should contain team content")
}

// TestIntegration_RulesBasicSync tests basic rule sync to Claude directory.
func TestIntegration_RulesBasicSync(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup team rule
	teamRule := `# Security Guidelines

Never commit secrets.`

	err := env.SetupTeamRule(owner, repo, "security.md", teamRule)
	require.NoError(t, err)

	// Run sync
	count, err := env.RunSyncRules(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should sync 1 rule")

	// Read output and verify
	output, err := env.ReadClaudeRule("security.md")
	require.NoError(t, err)

	assert.Contains(t, output, "Managed by staghorn", "should have managed header")
	assert.Contains(t, output, "Source: team", "should have team source")
	assert.Contains(t, output, "Never commit secrets", "should contain rule content")
}

// TestIntegration_RulesWithPaths tests rules with path-scoping frontmatter.
func TestIntegration_RulesWithPaths(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup rule with paths frontmatter
	teamRule := `---
paths:
  - "src/api/**/*.ts"
  - "src/routes/**/*.ts"
---
# REST API Standards

Use proper HTTP methods.`

	err := env.SetupTeamRule(owner, repo, "api/rest.md", teamRule)
	require.NoError(t, err)

	count, err := env.RunSyncRules(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	output, err := env.ReadClaudeRule("api/rest.md")
	require.NoError(t, err)

	// Verify frontmatter is preserved (required for Claude's path-scoping)
	assert.Contains(t, output, "---", "should have frontmatter delimiters")
	assert.Contains(t, output, "paths:", "should have paths in frontmatter")
	assert.Contains(t, output, "src/api/**/*.ts", "should preserve path patterns")
	assert.Contains(t, output, "Use proper HTTP methods", "should contain rule body")
}

// TestIntegration_RulesPrecedence tests that personal rules override team rules.
func TestIntegration_RulesPrecedence(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup team rule
	teamRule := `# Security

Team security guidelines.`

	// Setup personal rule with same path (should override)
	personalRule := `# Security

My personal security preferences.`

	err := env.SetupTeamRule(owner, repo, "security.md", teamRule)
	require.NoError(t, err)

	err = env.SetupPersonalRule("security.md", personalRule)
	require.NoError(t, err)

	count, err := env.RunSyncRules(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have 1 unique rule (personal overrides team)")

	output, err := env.ReadClaudeRule("security.md")
	require.NoError(t, err)

	// Personal should win
	assert.Contains(t, output, "Source: personal", "should have personal source (higher precedence)")
	assert.Contains(t, output, "My personal security preferences", "should contain personal content")
	assert.NotContains(t, output, "Team security guidelines", "should NOT contain team content")
}

// TestIntegration_RulesSubdirectories tests rules in subdirectories.
func TestIntegration_RulesSubdirectories(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup rules in different subdirectories
	err := env.SetupTeamRule(owner, repo, "security.md", "# Security")
	require.NoError(t, err)

	err = env.SetupTeamRule(owner, repo, "api/rest.md", "# REST API")
	require.NoError(t, err)

	err = env.SetupTeamRule(owner, repo, "frontend/react.md", "# React Guidelines")
	require.NoError(t, err)

	count, err := env.RunSyncRules(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should sync 3 rules")

	// Verify each rule exists in correct location
	_, err = env.ReadClaudeRule("security.md")
	require.NoError(t, err, "security.md should exist")

	_, err = env.ReadClaudeRule("api/rest.md")
	require.NoError(t, err, "api/rest.md should exist")

	_, err = env.ReadClaudeRule("frontend/react.md")
	require.NoError(t, err, "frontend/react.md should exist")
}

// TestIntegration_RulesEmptyDirs tests sync with no rules.
func TestIntegration_RulesEmptyDirs(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// No rules set up - directories don't exist

	count, err := env.RunSyncRules(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should sync 0 rules")
}

// TestIntegration_ProvenanceOrder tests that team content appears before personal.
func TestIntegration_ProvenanceOrder(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Team has Section A
	teamContent := `## Section A

Team content for A.`

	// Personal has Section A additions and new Section B
	personalContent := `## Section A

Personal additions for A.

## Section B

Personal-only section.`

	err := env.SetupTeamConfig(owner, repo, teamContent)
	require.NoError(t, err)

	err = env.SetupPersonalConfig(personalContent)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	// Team marker should appear before personal marker
	teamIdx := strings.Index(output, "<!-- staghorn:source:team -->")
	personalIdx := strings.Index(output, "<!-- staghorn:source:personal -->")

	assert.Greater(t, personalIdx, teamIdx, "team marker should appear before personal marker")

	// Team content should appear before personal content in same section
	teamContentIdx := strings.Index(output, "Team content for A")
	personalContentIdx := strings.Index(output, "Personal additions for A")

	assert.Greater(t, personalContentIdx, teamContentIdx, "team content should appear before personal additions")
}
