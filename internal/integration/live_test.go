//go:build live

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/github"
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
