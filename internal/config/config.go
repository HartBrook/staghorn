package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/HartBrook/staghorn/internal/errors"
	"gopkg.in/yaml.v3"
)

// repoPattern matches owner/repo format.
var repoPattern = regexp.MustCompile(`^([a-zA-Z0-9_.-]+)/([a-zA-Z0-9_.-]+)$`)

// LanguageConfig contains language-specific settings.
type LanguageConfig struct {
	AutoDetect bool     `yaml:"auto_detect"`
	Enabled    []string `yaml:"enabled,omitempty"`
	Disabled   []string `yaml:"disabled,omitempty"`
}

// Config represents the staghorn configuration file.
type Config struct {
	Version   int            `yaml:"version"`
	Team      TeamConfig     `yaml:"team"`
	Cache     CacheConfig    `yaml:"cache"`
	Languages LanguageConfig `yaml:"languages,omitempty"`
}

// TeamConfig contains team repository settings.
type TeamConfig struct {
	Repo   string `yaml:"repo"`   // e.g., "github.com/acme/standards" or "acme/standards"
	Branch string `yaml:"branch"` // optional, defaults to repo's default branch
	Path   string `yaml:"path"`   // optional, defaults to "CLAUDE.md"
}

// CacheConfig contains cache settings.
type CacheConfig struct {
	TTL string `yaml:"ttl"` // e.g., "24h"
}

// Default values.
const (
	DefaultVersion  = 1
	DefaultPath     = "CLAUDE.md"
	DefaultCacheTTL = "24h"
)

// Load reads and validates config from the default location.
func Load() (*Config, error) {
	paths := NewPaths()
	return LoadFrom(paths.ConfigFile)
}

// LoadFrom reads and validates config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.ConfigNotFound(path)
		}
		return nil, errors.Wrap(errors.ErrConfigInvalid, "failed to read config", "", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(errors.ErrConfigInvalid, "failed to parse config YAML", "Check config syntax", err)
	}

	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes config to the default location.
func Save(cfg *Config) error {
	paths := NewPaths()
	return SaveTo(cfg, paths.ConfigFile)
}

// SaveTo writes config to a specific path.
func SaveTo(cfg *Config, path string) error {
	cfg.applyDefaults()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(errors.ErrConfigInvalid, "failed to marshal config", "", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(errors.ErrConfigInvalid, "failed to create config directory", "", err)
	}

	return os.WriteFile(path, data, 0644)
}

// Validate checks config for required fields and valid values.
func (c *Config) Validate() error {
	if c.Team.Repo == "" {
		return errors.ConfigInvalid("team.repo is required")
	}

	if _, _, err := c.Team.ParseRepo(); err != nil {
		return err
	}

	if c.Cache.TTL != "" {
		if _, err := time.ParseDuration(c.Cache.TTL); err != nil {
			return errors.ConfigInvalid("invalid cache.ttl format, use Go duration format (e.g., 24h)")
		}
	}

	return nil
}

// applyDefaults sets default values for empty fields.
func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = DefaultVersion
	}
	if c.Team.Path == "" {
		c.Team.Path = DefaultPath
	}
	if c.Cache.TTL == "" {
		c.Cache.TTL = DefaultCacheTTL
	}
	// Language defaults: if no explicit list and auto_detect not set, enable auto_detect
	if len(c.Languages.Enabled) == 0 && !c.Languages.AutoDetect {
		c.Languages.AutoDetect = true
	}
}

// ParseRepo extracts owner and repo name from the repo string.
// Accepts formats:
//   - "https://github.com/owner/repo"
//   - "https://github.com/owner/repo.git"
//   - "https://github.com/owner/repo/tree/main"
//   - "https://github.com/owner/repo/blob/main/file.md"
//   - "github.com/owner/repo"
//   - "owner/repo"
func (t *TeamConfig) ParseRepo() (owner, repo string, err error) {
	repoStr := t.Repo

	// Strip protocol
	repoStr = strings.TrimPrefix(repoStr, "https://")
	repoStr = strings.TrimPrefix(repoStr, "http://")

	// Strip host (github.com for now, extensible for gitlab.com etc.)
	repoStr = strings.TrimPrefix(repoStr, "github.com/")

	// Strip .git suffix and trailing slashes
	repoStr = strings.TrimSuffix(repoStr, ".git")
	repoStr = strings.TrimSuffix(repoStr, "/")

	// Split by / and take only owner/repo (first two parts)
	// This handles URLs like owner/repo/tree/main or owner/repo/blob/main/file.md
	parts := strings.Split(repoStr, "/")
	if len(parts) >= 2 {
		repoStr = parts[0] + "/" + parts[1]
	}

	// Match owner/repo pattern
	matches := repoPattern.FindStringSubmatch(repoStr)
	if matches == nil {
		return "", "", errors.InvalidRepo(t.Repo)
	}

	return matches[1], matches[2], nil
}

// TTLDuration returns the cache TTL as a time.Duration.
func (c *CacheConfig) TTLDuration() time.Duration {
	d, err := time.ParseDuration(c.TTL)
	if err != nil {
		d, _ = time.ParseDuration(DefaultCacheTTL)
	}
	return d
}

// Exists checks if a config file exists at the default location.
func Exists() bool {
	paths := NewPaths()
	_, err := os.Stat(paths.ConfigFile)
	return err == nil
}
