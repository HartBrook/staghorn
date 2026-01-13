package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HartBrook/staghorn/internal/config"
	"github.com/HartBrook/staghorn/internal/errors"
)

// Cache manages local cached team configs.
type Cache struct {
	paths *config.Paths
}

// New creates a cache manager.
func New(paths *config.Paths) *Cache {
	return &Cache{paths: paths}
}

// Read returns cached content and metadata, or error if not cached.
func (c *Cache) Read(owner, repo string) (content string, meta *Metadata, err error) {
	contentPath := c.paths.CacheFile(owner, repo)
	metaPath := c.paths.CacheMetadataFile(owner, repo)

	// Read content
	contentBytes, err := os.ReadFile(contentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, errors.CacheNotFound(owner + "/" + repo)
		}
		return "", nil, err
	}

	// Read metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		// Content exists but metadata doesn't - create minimal metadata
		meta = &Metadata{
			Owner:       owner,
			Repo:        repo,
			LastFetched: time.Now(),
		}
	} else {
		meta = &Metadata{}
		if err := json.Unmarshal(metaBytes, meta); err != nil {
			// Invalid metadata - create minimal metadata
			meta = &Metadata{
				Owner:       owner,
				Repo:        repo,
				LastFetched: time.Now(),
			}
		}
	}

	return string(contentBytes), meta, nil
}

// Write stores content and metadata.
func (c *Cache) Write(owner, repo, content string, meta *Metadata) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(c.paths.CacheDir, 0755); err != nil {
		return err
	}

	contentPath := c.paths.CacheFile(owner, repo)
	metaPath := c.paths.CacheMetadataFile(owner, repo)

	// Update metadata timestamp if not set
	if meta.LastFetched.IsZero() {
		meta.LastFetched = time.Now()
	}
	meta.Owner = owner
	meta.Repo = repo

	// Write content
	if err := os.WriteFile(contentPath, []byte(content), 0644); err != nil {
		return err
	}

	// Write metadata
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, metaBytes, 0644)
}

// Exists checks if cache exists for a repo.
func (c *Cache) Exists(owner, repo string) bool {
	contentPath := c.paths.CacheFile(owner, repo)
	_, err := os.Stat(contentPath)
	return err == nil
}

// Clear removes cached content for a repo.
// Returns nil even if files don't exist (idempotent operation).
func (c *Cache) Clear(owner, repo string) error {
	contentPath := c.paths.CacheFile(owner, repo)
	metaPath := c.paths.CacheMetadataFile(owner, repo)

	// Remove both files, ignore "not exist" errors (idempotent)
	if err := os.Remove(contentPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cached content: %w", err)
	}
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache metadata: %w", err)
	}
	return nil
}

// GetMetadata returns only the metadata without reading content.
func (c *Cache) GetMetadata(owner, repo string) (*Metadata, error) {
	metaPath := c.paths.CacheMetadataFile(owner, repo)

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.CacheNotFound(owner + "/" + repo)
		}
		return nil, err
	}

	meta := &Metadata{}
	if err := json.Unmarshal(metaBytes, meta); err != nil {
		return nil, err
	}

	return meta, nil
}

// CacheDir returns the cache directory path.
func (c *Cache) CacheDir() string {
	return c.paths.CacheDir
}

// ListCached returns all cached repo identifiers.
func (c *Cache) ListCached() ([]string, error) {
	entries, err := os.ReadDir(c.paths.CacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var repos []string
	for _, entry := range entries {
		name := entry.Name()
		// Look for .md files (not .meta.json)
		if filepath.Ext(name) == ".md" {
			// Strip .md extension to get owner-repo
			repos = append(repos, strings.TrimSuffix(name, ".md"))
		}
	}
	return repos, nil
}
