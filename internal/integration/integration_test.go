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
	require.NotNil(t, fixture.Setup.Config, "fixture must have config setup")
	// Either Team or MultiSource must be present
	require.True(t, fixture.Setup.Team != nil || len(fixture.Setup.MultiSource) > 0,
		"fixture must have team or multi_source setup")

	// Create isolated environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Apply setup
	err := ApplySetup(env, fixture.Setup)
	require.NoError(t, err, "failed to apply setup")

	// Get config
	cfg := fixture.Setup.Config.ToConfig()

	// Run sync - use multi-source sync if configured
	if cfg.Source.IsMultiSource() {
		err = env.RunMultiSourceSync(cfg)
	} else {
		// Get owner/repo from team source
		require.NotNil(t, fixture.Setup.Team, "single-source fixture must have team setup")
		owner, repo := parseOwnerRepo(fixture.Setup.Team.Source)
		err = env.RunSync(owner, repo, cfg)
	}
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

// TestIntegration_MultiSourceLanguages tests multi-source language configuration.
func TestIntegration_MultiSourceLanguages(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup: base config from acme/standards
	baseContent := `## Core Principles

Base principles from acme/standards.`
	err := env.SetupTeamConfig("acme", "standards", baseContent)
	require.NoError(t, err)

	// Setup: python language from community/python-standards
	pythonContent := `## Python Guidelines

Use type hints from community repo.`
	err = env.SetupTeamLanguage("community", "python-standards", "python", pythonContent)
	require.NoError(t, err)

	// Setup: go language from default (acme/standards)
	goContent := `## Go Guidelines

Use gofmt from default repo.`
	err = env.SetupTeamLanguage("acme", "standards", "go", goContent)
	require.NoError(t, err)

	// Create multi-source config
	cfg := &config.Config{
		Version: 1,
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: "acme/standards",
				Languages: map[string]string{
					"python": "community/python-standards",
				},
			},
		},
		Languages: config.LanguageConfig{
			Enabled: []string{"python", "go"},
		},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	// Run multi-source sync
	err = env.RunMultiSourceSync(cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify base content is present
	assert.True(t, asserter.ContainsText("Base principles from acme/standards"),
		"should contain base config content")

	// Verify python content from community repo
	assert.True(t, asserter.ContainsText("Use type hints from community repo"),
		"should contain python content from community repo")

	// Verify go content from default repo
	assert.True(t, asserter.ContainsText("Use gofmt from default repo"),
		"should contain go content from default repo")

	// Verify header
	assert.True(t, asserter.HasManagedHeader(), "should have managed header")
}

// TestIntegration_MultiSourceWithBaseOverride tests base config from different repo.
func TestIntegration_MultiSourceWithBaseOverride(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup: base config from acme/base-config (NOT the default)
	baseContent := `## Base Config

Content from separate base repo.`
	err := env.SetupTeamConfig("acme", "base-config", baseContent)
	require.NoError(t, err)

	// Setup: language from default repo (acme/standards)
	pythonContent := `## Python

Python from default repo.`
	err = env.SetupTeamLanguage("acme", "standards", "python", pythonContent)
	require.NoError(t, err)

	// Create multi-source config with base override
	cfg := &config.Config{
		Version: 1,
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: "acme/standards",
				Base:    "acme/base-config",
			},
		},
		Languages: config.LanguageConfig{
			Enabled: []string{"python"},
		},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunMultiSourceSync(cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify base content is from override repo
	assert.True(t, asserter.ContainsText("Content from separate base repo"),
		"should contain base config from override repo")

	// Verify python content is from default repo
	assert.True(t, asserter.ContainsText("Python from default repo"),
		"should contain python from default repo")
}

// TestIntegration_MultiSourceFallback tests that unconfigured languages use default.
func TestIntegration_MultiSourceFallback(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup: base config
	baseContent := `## Standards

Team standards.`
	err := env.SetupTeamConfig("acme", "standards", baseContent)
	require.NoError(t, err)

	// Setup: python explicitly configured to community repo
	pythonContent := `## Python

Community Python.`
	err = env.SetupTeamLanguage("community", "python-standards", "python", pythonContent)
	require.NoError(t, err)

	// Setup: go NOT explicitly configured - should fallback to default
	goContent := `## Go

Default Go.`
	err = env.SetupTeamLanguage("acme", "standards", "go", goContent)
	require.NoError(t, err)

	// Setup: rust NOT explicitly configured - should fallback to default
	rustContent := `## Rust

Default Rust.`
	err = env.SetupTeamLanguage("acme", "standards", "rust", rustContent)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: "acme/standards",
				Languages: map[string]string{
					"python": "community/python-standards",
					// go and rust not specified - should use default
				},
			},
		},
		Languages: config.LanguageConfig{
			Enabled: []string{"python", "go", "rust"},
		},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	err = env.RunMultiSourceSync(cfg)
	require.NoError(t, err)

	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Python from community repo
	assert.True(t, asserter.ContainsText("Community Python"),
		"python should come from community repo")

	// Go and Rust from default repo
	assert.True(t, asserter.ContainsText("Default Go"),
		"go should fallback to default repo")
	assert.True(t, asserter.ContainsText("Default Rust"),
		"rust should fallback to default repo")
}

// TestIntegration_SkillsBasicSync tests basic skill sync to Claude directory.
func TestIntegration_SkillsBasicSync(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup team skill
	teamSkill := `---
name: code-review
description: Thorough code review
allowed-tools: Read Grep Glob
---

# Code Review

Review the code carefully.`

	err := env.SetupTeamSkill(owner, repo, "code-review", teamSkill)
	require.NoError(t, err)

	// Run sync
	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should sync 1 skill")

	// Read output and verify
	output, err := env.ReadClaudeSkill("code-review")
	require.NoError(t, err)

	assert.Contains(t, output, "Managed by staghorn", "should have managed header")
	assert.Contains(t, output, "Source: team", "should have team source")
	assert.Contains(t, output, "name: code-review", "should contain skill name")
	assert.Contains(t, output, "Review the code carefully", "should contain skill body")
}

// TestIntegration_SkillsPrecedence tests that personal skills override team skills.
func TestIntegration_SkillsPrecedence(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup team skill
	teamSkill := `---
name: code-review
description: Team code review
allowed-tools: Read Grep
---

Team review instructions.`

	// Setup personal skill with same name (should override)
	personalSkill := `---
name: code-review
description: My personal code review
allowed-tools: Read Grep Glob WebSearch
---

My custom review instructions.`

	err := env.SetupTeamSkill(owner, repo, "code-review", teamSkill)
	require.NoError(t, err)

	err = env.SetupPersonalSkill("code-review", personalSkill)
	require.NoError(t, err)

	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have 1 unique skill (personal overrides team)")

	output, err := env.ReadClaudeSkill("code-review")
	require.NoError(t, err)

	// Personal should win
	assert.Contains(t, output, "Source: personal", "should have personal source (higher precedence)")
	assert.Contains(t, output, "My custom review instructions", "should contain personal content")
	assert.NotContains(t, output, "Team review instructions", "should NOT contain team content")
}

// TestIntegration_SkillsWithSupportingFiles tests skills with templates and scripts.
func TestIntegration_SkillsWithSupportingFiles(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	skillMD := `---
name: test-gen
description: Generate tests with templates
allowed-tools: Read Grep Glob Write
---

# Test Generation

Use the templates in templates/ directory.`

	supportingFiles := map[string]string{
		"templates/jest.md":   "# Jest Template\n\nUse describe blocks.",
		"templates/pytest.md": "# Pytest Template\n\nUse fixtures.",
		"scripts/validate.sh": "#!/bin/bash\necho 'Validating...'",
	}

	err := env.SetupTeamSkillWithFiles(owner, repo, "test-gen", skillMD, supportingFiles)
	require.NoError(t, err)

	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should sync 1 skill")

	// Verify SKILL.md
	output, err := env.ReadClaudeSkill("test-gen")
	require.NoError(t, err)
	assert.Contains(t, output, "Use the templates in templates/ directory", "should contain skill body")

	// Verify supporting files were copied
	jestTemplate, err := env.ReadClaudeSkillFile("test-gen", "templates/jest.md")
	require.NoError(t, err)
	assert.Contains(t, jestTemplate, "Use describe blocks", "jest template should be copied")

	pytestTemplate, err := env.ReadClaudeSkillFile("test-gen", "templates/pytest.md")
	require.NoError(t, err)
	assert.Contains(t, pytestTemplate, "Use fixtures", "pytest template should be copied")

	script, err := env.ReadClaudeSkillFile("test-gen", "scripts/validate.sh")
	require.NoError(t, err)
	assert.Contains(t, script, "Validating", "script should be copied")
}

// TestIntegration_SkillsCollisionDetection tests that staghorn won't overwrite user skills.
func TestIntegration_SkillsCollisionDetection(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Create existing user skill (NOT managed by staghorn)
	userSkill := `---
name: my-custom-skill
description: User's own skill
---

My custom workflow that I created manually.`

	err := env.SetupExistingClaudeSkill("my-custom-skill", userSkill)
	require.NoError(t, err)

	// Setup team skill with same name
	teamSkill := `---
name: my-custom-skill
description: Team version trying to overwrite
---

Team content.`

	err = env.SetupTeamSkill(owner, repo, "my-custom-skill", teamSkill)
	require.NoError(t, err)

	// Sync should fail for this skill (collision)
	_, err = env.RunSyncSkills(owner, repo)
	// The sync returns error when collision is detected
	require.Error(t, err, "should error when trying to overwrite non-staghorn skill")

	// Verify user skill was NOT overwritten
	output, err := env.ReadClaudeSkill("my-custom-skill")
	require.NoError(t, err)
	assert.Contains(t, output, "My custom workflow that I created manually",
		"user's skill should be preserved")
	assert.NotContains(t, output, "Managed by staghorn",
		"should NOT have staghorn header")
}

// TestIntegration_SkillsEmptyDirs tests sync with no skills.
func TestIntegration_SkillsEmptyDirs(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// No skills set up - directories don't exist

	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should sync 0 skills")
}

// TestIntegration_SkillsMultiSource tests skills from different source repos.
func TestIntegration_SkillsMultiSource(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Setup: code-review from default repo (acme/standards)
	codeReviewSkill := `---
name: code-review
description: Team code review from acme
allowed-tools: Read Grep Glob
---

Review code using team standards.`
	err := env.SetupTeamSkill("acme", "standards", "code-review", codeReviewSkill)
	require.NoError(t, err)

	// Setup: react skill from vercel-labs/agent-skills
	reactSkill := `---
name: react
description: React development patterns from Vercel
allowed-tools: Read Grep Glob Write
---

# React Patterns

Use React Server Components when possible.`
	err = env.SetupTeamSkill("vercel-labs", "agent-skills", "react", reactSkill)
	require.NoError(t, err)

	// Setup: security-audit from community/security
	securitySkill := `---
name: security-audit
description: Security audit from community
allowed-tools: Read Grep Glob
context: fork
agent: Explore
---

# Security Audit

Check for OWASP Top 10 vulnerabilities.`
	err = env.SetupTeamSkill("community", "security", "security-audit", securitySkill)
	require.NoError(t, err)

	// Create multi-source config
	cfg := &config.Config{
		Version: 1,
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: "acme/standards",
				Skills: map[string]string{
					"react":          "vercel-labs/agent-skills",
					"security-audit": "community/security",
				},
			},
		},
	}

	// Run multi-source sync
	count, err := env.RunSyncSkillsMultiSource(cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should sync 3 skills from different repos")

	// Verify code-review from default repo
	codeReviewOutput, err := env.ReadClaudeSkill("code-review")
	require.NoError(t, err)
	assert.Contains(t, codeReviewOutput, "Review code using team standards",
		"code-review should come from acme/standards")

	// Verify react from vercel-labs
	reactOutput, err := env.ReadClaudeSkill("react")
	require.NoError(t, err)
	assert.Contains(t, reactOutput, "React Server Components",
		"react should come from vercel-labs/agent-skills")

	// Verify security-audit from community
	securityOutput, err := env.ReadClaudeSkill("security-audit")
	require.NoError(t, err)
	assert.Contains(t, securityOutput, "OWASP Top 10",
		"security-audit should come from community/security")
}

// TestIntegration_SkillsMultipleFromSameTeam tests multiple skills from default repo.
func TestIntegration_SkillsMultipleFromSameTeam(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Cleanup()

	owner, repo := "acme", "standards"

	// Setup multiple skills from same repo
	skills := map[string]string{
		"code-review": `---
name: code-review
description: Code review
---

Review code.`,
		"test-gen": `---
name: test-gen
description: Generate tests
---

Generate tests.`,
		"refactor": `---
name: refactor
description: Refactoring helper
---

Refactor code.`,
	}

	for name, content := range skills {
		err := env.SetupTeamSkill(owner, repo, name, content)
		require.NoError(t, err)
	}

	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should sync 3 skills")

	// Verify all skills exist
	for name := range skills {
		_, err := env.ReadClaudeSkill(name)
		require.NoError(t, err, "%s should exist", name)
	}
}
