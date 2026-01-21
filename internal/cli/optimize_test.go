package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptimizeCmd(t *testing.T) {
	cmd := NewOptimizeCmd()

	assert.Equal(t, "optimize", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewOptimizeCmd_Flags(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check all flags exist
	flags := []string{
		"layer",
		"target",
		"dry-run",
		"diff",
		"output",
		"force",
		"deterministic",
		"verbose",
		"no-cache",
	}

	for _, flag := range flags {
		f := cmd.Flags().Lookup(flag)
		require.NotNil(t, f, "flag %q should exist", flag)
	}
}

func TestNewOptimizeCmd_FlagDefaults(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check default values
	layer, _ := cmd.Flags().GetString("layer")
	assert.Equal(t, "merged", layer)

	target, _ := cmd.Flags().GetInt("target")
	assert.Equal(t, 0, target)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	assert.False(t, dryRun)

	deterministic, _ := cmd.Flags().GetBool("deterministic")
	assert.False(t, deterministic)
}

func TestNewOptimizeCmd_ShortFlags(t *testing.T) {
	cmd := NewOptimizeCmd()

	// Check short flags exist
	shortFlags := map[string]string{
		"o": "output",
		"v": "verbose",
	}

	for short, long := range shortFlags {
		f := cmd.Flags().ShorthandLookup(short)
		require.NotNil(t, f, "short flag %q should exist", short)
		assert.Equal(t, long, f.Name)
	}
}

func TestApplyMergedOptimization(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create paths pointing to temp dir
	paths := &config.Paths{
		ConfigDir:  tmpDir,
		PersonalMD: filepath.Join(tmpDir, "personal.md"),
		CacheDir:   filepath.Join(tmpDir, "cache"),
	}

	// Create cache directory and cache file for team layer
	require.NoError(t, os.MkdirAll(paths.CacheDir, 0755))

	owner, repo := "acme", "standards"

	// Create team cache file (required by applyOptimization for team layer)
	teamCacheFile := paths.CacheFile(owner, repo)
	require.NoError(t, os.WriteFile(teamCacheFile, []byte("old team content"), 0644))

	// Create initial personal.md
	initialPersonal := "## My Preferences\n\nOld personal content."
	require.NoError(t, os.WriteFile(paths.PersonalMD, []byte(initialPersonal), 0644))

	// Create merged content with provenance markers
	mergedContent := `<!-- staghorn:source:team -->
## Code Style

Optimized team content.

## Testing

Write tests.

<!-- staghorn:source:personal -->
### Personal Additions

Optimized personal content.`

	// Apply merged optimization (not in a source repo context)
	err := applyMergedOptimization(paths, owner, repo, mergedContent, false, "", false)
	require.NoError(t, err)

	// Verify personal.md was updated
	personalContent, err := os.ReadFile(paths.PersonalMD)
	require.NoError(t, err)
	assert.Contains(t, string(personalContent), "Optimized personal content")

	// Verify team cache was updated
	teamContent, err := os.ReadFile(teamCacheFile)
	require.NoError(t, err)
	assert.Contains(t, string(teamContent), "Optimized team content")
}

func TestApplyMergedOptimization_NoProvenance(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &config.Paths{
		ConfigDir:  tmpDir,
		PersonalMD: filepath.Join(tmpDir, "personal.md"),
	}

	// Content without provenance markers
	content := `## Code Style

Some content without markers.`

	err := applyMergedOptimization(paths, "acme", "standards", content, false, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no provenance markers")
}

func TestApplyMergedOptimization_PreservesOrder(t *testing.T) {
	// Test that sources are processed in order of appearance
	content := `<!-- staghorn:source:team -->
## First

Team content.

<!-- staghorn:source:personal -->
## Second

Personal content.

<!-- staghorn:source:team -->
## Third

More team content.`

	sources := merge.ListSources(content)

	// Should preserve order: team first, then personal
	require.Len(t, sources, 2)
	assert.Equal(t, "team", sources[0])
	assert.Equal(t, "personal", sources[1])

	// ParseProvenance should aggregate by source
	provenance := merge.ParseProvenance(content)
	assert.Contains(t, provenance["team"], "Team content")
	assert.Contains(t, provenance["team"], "More team content")
	assert.Contains(t, provenance["personal"], "Personal content")
}

func TestApplyOptimization_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &config.Paths{
		ConfigDir:  tmpDir,
		PersonalMD: filepath.Join(tmpDir, "personal.md"),
	}

	// Create initial personal.md with content
	initialContent := "## My Preferences\n\nOriginal content."
	require.NoError(t, os.WriteFile(paths.PersonalMD, []byte(initialContent), 0644))

	// Try to apply empty content - should fail
	err := applyOptimization(paths, "acme", "standards", "personal", "", "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to write empty content")

	// Try to apply whitespace-only content - should also fail
	err = applyOptimization(paths, "acme", "standards", "personal", "   \n\t  ", "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to write empty content")

	// Verify original content was not modified
	content, err := os.ReadFile(paths.PersonalMD)
	require.NoError(t, err)
	assert.Equal(t, initialContent, string(content))
}

func TestApplyOptimization_ProjectLayer(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &config.Paths{
		ConfigDir: tmpDir,
	}

	// Create .staghorn directory for project
	staghornDir := filepath.Join(tmpDir, ".staghorn")
	require.NoError(t, os.MkdirAll(staghornDir, 0755))

	// Create initial project.md
	initialContent := "## Project Config\n\nOriginal project content."
	projectMD := filepath.Join(staghornDir, "project.md")
	require.NoError(t, os.WriteFile(projectMD, []byte(initialContent), 0644))

	// Apply project layer optimization
	newContent := "## Project Config\n\nOptimized project content."
	err := applyOptimization(paths, "acme", "standards", "project", newContent, tmpDir, false)
	require.NoError(t, err)

	// Verify project.md was updated
	content, err := os.ReadFile(projectMD)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Optimized project content")
}

func TestApplyOptimization_ProjectLayerNoProjectRoot(t *testing.T) {
	paths := &config.Paths{
		ConfigDir: t.TempDir(),
	}

	// Try to apply to project layer without a project root - should fail
	err := applyOptimization(paths, "acme", "standards", "project", "some content", "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a project directory")
}
