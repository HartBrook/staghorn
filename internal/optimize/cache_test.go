package optimize

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashContent(t *testing.T) {
	// Same content should produce same hash
	hash1 := HashContent("test content")
	hash2 := HashContent("test content")
	assert.Equal(t, hash1, hash2)

	// Different content should produce different hash
	hash3 := HashContent("different content")
	assert.NotEqual(t, hash1, hash3)

	// Hash should be 64 chars (256 bits in hex)
	assert.Len(t, hash1, 64)
}

func TestOptimizationCache_Write(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	meta := &OptimizationMeta{
		SourceHash:      "abc123",
		OptimizedAt:     time.Now(),
		OriginalTokens:  1000,
		OptimizedTokens: 500,
		Model:           "claude-sonnet-4-20250514",
		Deterministic:   false,
	}

	err := cache.Write("acme", "standards", "optimized content", meta)
	require.NoError(t, err)

	// Verify files exist
	contentPath := paths.OptimizedFile("acme", "standards")
	metaPath := paths.OptimizedMetaFile("acme", "standards")

	assert.FileExists(t, contentPath)
	assert.FileExists(t, metaPath)

	// Verify content
	content, err := os.ReadFile(contentPath)
	require.NoError(t, err)
	assert.Equal(t, "optimized content", string(content))
}

func TestOptimizationCache_Read(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Write test data
	meta := &OptimizationMeta{
		SourceHash:      "abc123",
		OptimizedAt:     time.Now().Truncate(time.Second),
		OriginalTokens:  1000,
		OptimizedTokens: 500,
		Model:           "claude-sonnet-4-20250514",
		Deterministic:   false,
	}
	err := cache.Write("acme", "standards", "optimized content", meta)
	require.NoError(t, err)

	// Read it back
	content, readMeta, err := cache.Read("acme", "standards")
	require.NoError(t, err)
	assert.Equal(t, "optimized content", content)
	assert.Equal(t, meta.SourceHash, readMeta.SourceHash)
	assert.Equal(t, meta.OriginalTokens, readMeta.OriginalTokens)
	assert.Equal(t, meta.OptimizedTokens, readMeta.OptimizedTokens)
}

func TestOptimizationCache_ReadMeta(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Write test data
	meta := &OptimizationMeta{
		SourceHash:      "xyz789",
		OptimizedAt:     time.Now().Truncate(time.Second),
		OriginalTokens:  2000,
		OptimizedTokens: 800,
		Model:           "claude-sonnet-4-20250514",
		Deterministic:   true,
	}
	err := cache.Write("acme", "standards", "content", meta)
	require.NoError(t, err)

	// Read only metadata
	readMeta, err := cache.ReadMeta("acme", "standards")
	require.NoError(t, err)
	assert.Equal(t, "xyz789", readMeta.SourceHash)
	assert.True(t, readMeta.Deterministic)
}

func TestOptimizationCache_IsStale(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Write with specific hash
	meta := &OptimizationMeta{
		SourceHash:      "original_hash",
		OptimizedAt:     time.Now(),
		OriginalTokens:  1000,
		OptimizedTokens: 500,
	}
	err := cache.Write("acme", "standards", "content", meta)
	require.NoError(t, err)

	// Same hash should not be stale
	assert.False(t, cache.IsStale("acme", "standards", "original_hash"))

	// Different hash should be stale
	assert.True(t, cache.IsStale("acme", "standards", "new_hash"))
}

func TestOptimizationCache_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Read non-existent entry
	content, meta, err := cache.Read("nonexistent", "repo")
	require.NoError(t, err)
	assert.Empty(t, content)
	assert.Nil(t, meta)
}

func TestOptimizationCache_Exists(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Should not exist initially
	assert.False(t, cache.Exists("acme", "standards"))

	// Write and check exists
	meta := &OptimizationMeta{SourceHash: "abc"}
	err := cache.Write("acme", "standards", "content", meta)
	require.NoError(t, err)

	assert.True(t, cache.Exists("acme", "standards"))
}

func TestOptimizationCache_Clear(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Write test data
	meta := &OptimizationMeta{SourceHash: "abc"}
	err := cache.Write("acme", "standards", "content", meta)
	require.NoError(t, err)

	assert.True(t, cache.Exists("acme", "standards"))

	// Clear
	err = cache.Clear("acme", "standards")
	require.NoError(t, err)

	assert.False(t, cache.Exists("acme", "standards"))
}

func TestOptimizationCache_Clear_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Clear non-existent entry should not error
	err := cache.Clear("nonexistent", "repo")
	require.NoError(t, err)
}

func TestOptimizationCache_ListCached(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Empty initially
	list, err := cache.ListCached()
	require.NoError(t, err)
	assert.Empty(t, list)

	// Add some entries
	meta := &OptimizationMeta{SourceHash: "abc"}
	err = cache.Write("acme", "standards", "content1", meta)
	require.NoError(t, err)
	err = cache.Write("other", "repo", "content2", meta)
	require.NoError(t, err)

	list, err = cache.ListCached()
	require.NoError(t, err)
	assert.Len(t, list, 2)
	assert.Contains(t, list, "acme-standards")
	assert.Contains(t, list, "other-repo")
}

func TestOptimizationCache_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	// Use a non-existent subdirectory
	configDir := filepath.Join(tempDir, "nested", "config")
	paths := config.NewPathsWithOverrides(configDir, tempDir)
	cache := NewOptimizationCache(paths)

	meta := &OptimizationMeta{SourceHash: "abc"}
	err := cache.Write("acme", "standards", "content", meta)
	require.NoError(t, err)

	// Verify directory was created
	optimizedDir := paths.OptimizedDir()
	assert.DirExists(t, optimizedDir)
}

func TestOptimizationCache_Read_CleansUpOrphanedContent(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Create only content file without metadata (orphaned state)
	contentPath := paths.OptimizedFile("acme", "orphan")
	err := os.MkdirAll(filepath.Dir(contentPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(contentPath, []byte("orphaned content"), 0644)
	require.NoError(t, err)

	// Verify content file exists
	assert.FileExists(t, contentPath)

	// Read should return empty and clean up the orphaned file
	content, meta, err := cache.Read("acme", "orphan")
	require.NoError(t, err)
	assert.Empty(t, content)
	assert.Nil(t, meta)

	// Verify orphaned content file was cleaned up
	_, err = os.Stat(contentPath)
	assert.True(t, os.IsNotExist(err), "Orphaned content file should be removed")
}

func TestOptimizationCache_Read_CleansUpCorruptedMeta(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Create content and corrupted metadata
	contentPath := paths.OptimizedFile("acme", "corrupt")
	metaPath := paths.OptimizedMetaFile("acme", "corrupt")
	err := os.MkdirAll(filepath.Dir(contentPath), 0755)
	require.NoError(t, err)

	err = os.WriteFile(contentPath, []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(metaPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	// Read should return empty and clean up
	content, meta, err := cache.Read("acme", "corrupt")
	require.NoError(t, err)
	assert.Empty(t, content)
	assert.Nil(t, meta)

	// Both files should be cleaned up
	_, err = os.Stat(contentPath)
	assert.True(t, os.IsNotExist(err), "Content file should be removed")
	_, err = os.Stat(metaPath)
	assert.True(t, os.IsNotExist(err), "Corrupted meta file should be removed")
}
