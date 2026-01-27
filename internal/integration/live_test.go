//go:build live

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/github"
	"github.com/HartBrook/staghorn/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLive_CommunityRepo tests against the real staghorn-community repo.
// Run with: go test -tags=live ./internal/integration/...
func TestLive_CommunityRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}

	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		t.Skip("GitHub auth not available, skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	owner, repo := "HartBrook", "staghorn-community"

	// Fetch real content
	result, err := client.FetchFile(ctx, owner, repo, "CLAUDE.md", "")
	if err != nil {
		t.Skipf("Failed to fetch from GitHub (may be rate limited): %v", err)
	}

	// Setup test environment with real content
	err = env.SetupTeamConfig(owner, repo, result.Content)
	require.NoError(t, err)

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	// Run sync
	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	// Read and verify output
	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify basic structure
	assert.True(t, asserter.HasManagedHeader(), "should have managed header")
	assert.True(t, asserter.HasSourceRepo(owner+"/"+repo), "should have correct source repo")
	assert.True(t, asserter.HasProvenanceMarker("team"), "should have team marker")

	// Verify some expected content from community standards
	assert.True(t, asserter.ContainsSection("## Core Principles") || asserter.ContainsSection("## Code"), "should have main section headers")
}

// TestLive_WithLanguages tests fetching language configs from the community repo.
// Run with: go test -tags=live ./internal/integration/...
func TestLive_WithLanguages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}

	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		t.Skip("GitHub auth not available, skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	owner, repo := "HartBrook", "staghorn-community"

	// Fetch main CLAUDE.md
	result, err := client.FetchFile(ctx, owner, repo, "CLAUDE.md", "")
	if err != nil {
		t.Skipf("Failed to fetch from GitHub: %v", err)
	}

	err = env.SetupTeamConfig(owner, repo, result.Content)
	require.NoError(t, err)

	// Try to fetch python language config
	pythonResult, err := client.FetchFile(ctx, owner, repo, "languages/python.md", "")
	if err == nil {
		err = env.SetupTeamLanguage(owner, repo, "python", pythonResult.Content)
		require.NoError(t, err)
	}

	cfg := &config.Config{
		Version: 1,
		Source:  config.Source{Simple: owner + "/" + repo},
	}
	err = env.SetupConfig(cfg)
	require.NoError(t, err)

	// Run sync
	err = env.RunSync(owner, repo, cfg)
	require.NoError(t, err)

	// Read output
	output, err := env.ReadOutput()
	require.NoError(t, err)

	asserter := NewAsserter(t, output)

	// Verify output has expected structure
	assert.True(t, asserter.HasManagedHeader(), "should have managed header")
	assert.True(t, asserter.HasProvenanceMarker("team"), "should have team marker")
}

// TestLive_VercelSkills tests fetching skills from vercel-labs/agent-skills.
// This validates that staghorn can parse real-world Agent Skills format from external repos.
// Run with: go test -tags=live ./internal/integration/...
func TestLive_VercelSkills(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}

	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		t.Skip("GitHub auth not available, skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	owner, repo := "vercel-labs", "agent-skills"

	// List skills directory to discover available skills
	entries, err := client.ListDirectory(ctx, owner, repo, "skills", "")
	if err != nil {
		t.Skipf("Failed to list skills directory from GitHub: %v", err)
	}

	if len(entries) == 0 {
		t.Skip("No skills found in vercel-labs/agent-skills")
	}

	// Fetch and parse each skill
	var fetchedSkills []*skills.Skill
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		skillPath := "skills/" + entry.Name + "/SKILL.md"
		result, err := client.FetchFile(ctx, owner, repo, skillPath, "")
		if err != nil {
			t.Logf("Skipping skill %s: %v", entry.Name, err)
			continue
		}

		// Parse the skill to validate format compatibility
		skill, err := skills.Parse(result.Content, skills.SourceTeam, "")
		if err != nil {
			t.Errorf("Failed to parse skill %s: %v", entry.Name, err)
			continue
		}

		fetchedSkills = append(fetchedSkills, skill)
		t.Logf("Successfully parsed skill: %s (%s)", skill.Name, skill.Description)
	}

	require.NotEmpty(t, fetchedSkills, "should have fetched at least one skill")

	// Setup skills in test environment and sync to Claude
	for _, skill := range fetchedSkills {
		// Recreate SKILL.md content for setup
		content := buildSkillMD(skill)
		err := env.SetupTeamSkill(owner, repo, skill.Name, content)
		require.NoError(t, err, "failed to setup skill %s", skill.Name)
	}

	// Run sync
	count, err := env.RunSyncSkills(owner, repo)
	require.NoError(t, err)
	assert.Equal(t, len(fetchedSkills), count, "should sync all fetched skills")

	// Verify skills were synced correctly
	for _, skill := range fetchedSkills {
		output, err := env.ReadClaudeSkill(skill.Name)
		require.NoError(t, err, "should be able to read synced skill %s", skill.Name)

		assert.Contains(t, output, "Managed by staghorn", "skill %s should have staghorn header", skill.Name)
		assert.Contains(t, output, "name: "+skill.Name, "skill %s should have name in frontmatter", skill.Name)
	}

	t.Logf("Successfully synced %d skills from vercel-labs/agent-skills", count)
}

// TestLive_VercelSkillsMultiSource tests a multi-source config with Vercel skills.
// Run with: go test -tags=live ./internal/integration/...
func TestLive_VercelSkillsMultiSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}

	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create GitHub client
	client, err := github.NewClient()
	if err != nil {
		t.Skip("GitHub auth not available, skipping live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Fetch one skill from Vercel
	vercelOwner, vercelRepo := "vercel-labs", "agent-skills"

	// List and pick the first available skill
	entries, err := client.ListDirectory(ctx, vercelOwner, vercelRepo, "skills", "")
	if err != nil {
		t.Skipf("Failed to list Vercel skills: %v", err)
	}

	// Find first valid skill (directory with SKILL.md)
	var vercelSkill *skills.Skill
	var vercelSkillContent string
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}

		skillPath := "skills/" + entry.Name + "/SKILL.md"
		result, fetchErr := client.FetchFile(ctx, vercelOwner, vercelRepo, skillPath, "")
		if fetchErr != nil {
			t.Logf("Skipping %s: no SKILL.md", entry.Name)
			continue
		}

		skill, parseErr := skills.Parse(result.Content, skills.SourceTeam, "")
		if parseErr != nil {
			t.Logf("Skipping %s: failed to parse: %v", entry.Name, parseErr)
			continue
		}

		vercelSkill = skill
		vercelSkillContent = result.Content
		break
	}

	if vercelSkill == nil {
		t.Skip("No valid skills found in vercel-labs/agent-skills")
	}

	// Setup Vercel skill
	err = env.SetupTeamSkill(vercelOwner, vercelRepo, vercelSkill.Name, vercelSkillContent)
	require.NoError(t, err)

	// Also fetch from staghorn-community if available
	communityOwner, communityRepo := "HartBrook", "staghorn-community"
	communityEntries, err := client.ListDirectory(ctx, communityOwner, communityRepo, "skills", "")

	var communitySkillName string
	if err == nil && len(communityEntries) > 0 {
		for _, entry := range communityEntries {
			if entry.Type == "dir" {
				communitySkillName = entry.Name

				communityResult, fetchErr := client.FetchFile(ctx, communityOwner, communityRepo, "skills/"+entry.Name+"/SKILL.md", "")
				if fetchErr == nil {
					err = env.SetupTeamSkill(communityOwner, communityRepo, entry.Name, communityResult.Content)
					if err != nil {
						t.Logf("Warning: failed to setup community skill %s: %v", entry.Name, err)
						communitySkillName = ""
					}
				}
				break
			}
		}
	}

	// Create multi-source config
	cfg := &config.Config{
		Version: 1,
		Source: config.Source{
			Multi: &config.SourceConfig{
				Default: communityOwner + "/" + communityRepo,
				Skills: map[string]string{
					vercelSkill.Name: vercelOwner + "/" + vercelRepo,
				},
			},
		},
	}

	// Run multi-source sync
	count, err := env.RunSyncSkillsMultiSource(cfg)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "should sync at least the Vercel skill")

	// Verify Vercel skill was synced
	output, err := env.ReadClaudeSkill(vercelSkill.Name)
	require.NoError(t, err)
	assert.Contains(t, output, "Managed by staghorn", "Vercel skill should have staghorn header")

	t.Logf("Successfully synced %d skills in multi-source config (Vercel: %s)", count, vercelSkill.Name)
	if communitySkillName != "" {
		t.Logf("Also included community skill: %s", communitySkillName)
	}
}

// buildSkillMD reconstructs a SKILL.md from a parsed skill.
// This is a simplified version for testing - the real content comes from GitHub.
func buildSkillMD(skill *skills.Skill) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: " + skill.Name + "\n")
	sb.WriteString("description: " + skill.Description + "\n")
	if skill.AllowedTools != "" {
		sb.WriteString("allowed-tools: " + skill.AllowedTools + "\n")
	}
	if skill.Context != "" {
		sb.WriteString("context: " + skill.Context + "\n")
	}
	if skill.Agent != "" {
		sb.WriteString("agent: " + skill.Agent + "\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(skill.Body)
	return sb.String()
}
