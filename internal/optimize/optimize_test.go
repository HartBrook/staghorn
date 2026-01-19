package optimize

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimizer_CacheModelMatching(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Pre-populate cache with a specific model
	meta := &OptimizationMeta{
		SourceHash:      HashContent("test content"),
		OptimizedAt:     time.Now(),
		OriginalTokens:  100,
		OptimizedTokens: 50,
		Model:           "claude-sonnet-4-20250514",
		Deterministic:   false,
	}
	err := cache.Write("acme", "test", "cached optimized", meta)
	require.NoError(t, err)

	optimizer := NewOptimizer(paths)

	// Test 1: Same model should hit cache
	result, err := optimizer.Optimize(context.Background(), "test content", "acme", "test", Options{
		Model:         "claude-sonnet-4-20250514",
		Deterministic: false,
		NoCache:       false,
	})
	require.NoError(t, err)
	assert.True(t, result.FromCache, "Should use cache when model matches")
	assert.Equal(t, "cached optimized", result.OptimizedContent)

	// Test 2: Empty model (default) should hit cache if default matches
	result2, err := optimizer.Optimize(context.Background(), "test content", "acme", "test", Options{
		Model:         "", // Will use default
		Deterministic: false,
		NoCache:       false,
	})
	require.NoError(t, err)
	assert.True(t, result2.FromCache, "Should use cache when empty model defaults to matching model")

	// Test 3: Different model should NOT hit cache
	result3, err := optimizer.Optimize(context.Background(), "test content", "acme", "test", Options{
		Model:         "claude-opus-4-20250514",
		Deterministic: false,
		NoCache:       false, // Allow cache but expect miss due to model mismatch
	})
	// This will fail because we don't have API key, but that's expected
	// The important thing is it didn't return the cached result
	if err == nil {
		assert.False(t, result3.FromCache, "Should NOT use cache when model differs")
	}
}

func TestOptimizer_CacheDeterministicMatching(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	cache := NewOptimizationCache(paths)

	// Pre-populate cache with deterministic=true
	meta := &OptimizationMeta{
		SourceHash:      HashContent("test content"),
		OptimizedAt:     time.Now(),
		OriginalTokens:  100,
		OptimizedTokens: 50,
		Model:           "",
		Deterministic:   true,
	}
	err := cache.Write("acme", "det-test", "deterministic cached", meta)
	require.NoError(t, err)

	optimizer := NewOptimizer(paths)

	// Test 1: Deterministic=true should hit cache
	result, err := optimizer.Optimize(context.Background(), "test content", "acme", "det-test", Options{
		Deterministic: true,
		NoCache:       false,
	})
	require.NoError(t, err)
	assert.True(t, result.FromCache, "Should use cache when deterministic matches")
	assert.Equal(t, "deterministic cached", result.OptimizedContent)

	// Test 2: Deterministic=false should NOT hit cache (even though hash matches)
	result2, err := optimizer.Optimize(context.Background(), "test content", "acme", "det-test", Options{
		Deterministic: false,
		NoCache:       false,
	})
	// Will fail due to no API key, but should not return cached result
	if err == nil {
		assert.False(t, result2.FromCache, "Should NOT use cache when deterministic differs")
	}
}

func TestOptimizer_DeterministicMode(t *testing.T) {
	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	optimizer := NewOptimizer(paths)

	content := "## Test\n\nYou should use pytest for testing.\n\n\n\nUse ruff for linting."

	// Deterministic mode should work without API key
	result, err := optimizer.Optimize(context.Background(), content, "acme", "det", Options{
		Deterministic: true,
		NoCache:       true,
	})
	require.NoError(t, err)

	// Should have applied preprocessing
	assert.NotContains(t, result.OptimizedContent, "\n\n\n", "Should collapse blank lines")
	assert.Contains(t, result.OptimizedContent, "Use pytest", "Should strip verbose phrases")
	assert.True(t, result.Deterministic)
	assert.False(t, result.FromCache)
}

func TestOptimizer_GetClient_ModelSwitching(t *testing.T) {
	// Set test API key
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	optimizer := NewOptimizer(paths)

	// Get client with model A
	client1, err := optimizer.getClient("model-a")
	require.NoError(t, err)
	assert.NotNil(t, client1)
	assert.Equal(t, "model-a", optimizer.clientModel)

	// Get client with same model should return same client
	client2, err := optimizer.getClient("model-a")
	require.NoError(t, err)
	assert.Same(t, client1, client2, "Same model should return same client")

	// Get client with different model should create new client
	client3, err := optimizer.getClient("model-b")
	require.NoError(t, err)
	assert.NotSame(t, client1, client3, "Different model should create new client")
	assert.Equal(t, "model-b", optimizer.clientModel)
}

func TestOptimizer_GetClient_EmptyModelUsesDefault(t *testing.T) {
	original := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	defer os.Setenv("ANTHROPIC_API_KEY", original)

	tempDir := t.TempDir()
	paths := config.NewPathsWithOverrides(tempDir, tempDir)
	optimizer := NewOptimizer(paths)

	// Get client with empty model
	client1, err := optimizer.getClient("")
	require.NoError(t, err)
	assert.Equal(t, defaultModel, optimizer.clientModel)

	// Get client with empty model again should return same client
	client2, err := optimizer.getClient("")
	require.NoError(t, err)
	assert.Same(t, client1, client2)

	// Get client with explicit default model should also return same client
	client3, err := optimizer.getClient(defaultModel)
	require.NoError(t, err)
	assert.Same(t, client1, client3, "Explicit default model should match empty model")
}
