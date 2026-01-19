package optimize

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
)

// OptimizationMeta tracks optimization state for cached results.
type OptimizationMeta struct {
	SourceHash      string    `json:"source_hash"`
	OptimizedAt     time.Time `json:"optimized_at"`
	OriginalTokens  int       `json:"original_tokens"`
	OptimizedTokens int       `json:"optimized_tokens"`
	Model           string    `json:"model"`
	Deterministic   bool      `json:"deterministic"`
}

// OptimizationCache manages cached optimization results.
type OptimizationCache struct {
	paths *config.Paths
}

// NewOptimizationCache creates a new cache.
func NewOptimizationCache(paths *config.Paths) *OptimizationCache {
	return &OptimizationCache{paths: paths}
}

// HashContent generates a SHA256 hash for content.
func HashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Read retrieves cached optimization result and metadata.
// Returns empty string and nil metadata if not found.
// If content exists but metadata is missing/corrupted, cleans up orphaned files.
func (c *OptimizationCache) Read(owner, repo string) (string, *OptimizationMeta, error) {
	contentPath := c.paths.OptimizedFile(owner, repo)

	content, err := os.ReadFile(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, err
	}

	meta, err := c.ReadMeta(owner, repo)
	if err != nil {
		// Content exists but metadata doesn't or is corrupted - clean up orphaned content
		_ = c.Clear(owner, repo)
		return "", nil, nil
	}

	return string(content), meta, nil
}

// ReadMeta retrieves only the metadata for an optimization.
func (c *OptimizationCache) ReadMeta(owner, repo string) (*OptimizationMeta, error) {
	metaPath := c.paths.OptimizedMetaFile(owner, repo)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta OptimizationMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// Write stores an optimization result with metadata.
func (c *OptimizationCache) Write(owner, repo, content string, meta *OptimizationMeta) error {
	// Ensure directory exists
	dir := c.paths.OptimizedDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write content
	contentPath := c.paths.OptimizedFile(owner, repo)
	if err := os.WriteFile(contentPath, []byte(content), 0644); err != nil {
		return err
	}

	// Write metadata
	metaPath := c.paths.OptimizedMetaFile(owner, repo)
	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, metaData, 0644)
}

// IsStale checks if the cached optimization is stale based on source hash.
func (c *OptimizationCache) IsStale(owner, repo, sourceHash string) bool {
	meta, err := c.ReadMeta(owner, repo)
	if err != nil {
		return true
	}
	return meta.SourceHash != sourceHash
}

// Exists checks if an optimization cache entry exists.
func (c *OptimizationCache) Exists(owner, repo string) bool {
	contentPath := c.paths.OptimizedFile(owner, repo)
	metaPath := c.paths.OptimizedMetaFile(owner, repo)

	if _, err := os.Stat(contentPath); err != nil {
		return false
	}
	if _, err := os.Stat(metaPath); err != nil {
		return false
	}
	return true
}

// Clear removes cached optimization for a repo.
func (c *OptimizationCache) Clear(owner, repo string) error {
	contentPath := c.paths.OptimizedFile(owner, repo)
	metaPath := c.paths.OptimizedMetaFile(owner, repo)

	// Ignore errors for non-existent files
	os.Remove(contentPath)
	os.Remove(metaPath)

	return nil
}

// ListCached returns all cached optimizations.
func (c *OptimizationCache) ListCached() ([]string, error) {
	dir := c.paths.OptimizedDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".md" {
			// Strip .md extension to get owner-repo
			name := entry.Name()
			name = name[:len(name)-3]
			result = append(result, name)
		}
	}

	return result, nil
}
