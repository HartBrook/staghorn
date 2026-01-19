// Package config handles staghorn configuration.
package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/HartBrook/staghorn/internal/errors"
	"gopkg.in/yaml.v3"
)

// LanguageConfig contains language-specific settings.
type LanguageConfig struct {
	AutoDetect bool     `yaml:"auto_detect"`
	Enabled    []string `yaml:"enabled,omitempty"`
	Disabled   []string `yaml:"disabled,omitempty"`
}

// CacheConfig contains cache settings.
type CacheConfig struct {
	TTL string `yaml:"ttl"` // e.g., "24h"
}

// OptimizeConfig contains optimization settings.
type OptimizeConfig struct {
	WarnThreshold     int    `yaml:"warn_threshold,omitempty"`     // Token threshold for warning (default: 3000)
	TargetTokens      int    `yaml:"target_tokens,omitempty"`      // Default target token count
	Model             string `yaml:"model,omitempty"`              // Model for optimization
	DeterministicOnly bool   `yaml:"deterministic_only,omitempty"` // Skip LLM, only do deterministic cleanup
}

// Config represents the staghorn configuration file.
type Config struct {
	Version int `yaml:"version"`

	// Source defines where to fetch configs from.
	// Can be a simple string ("owner/repo") or a structured object for multi-source.
	Source Source `yaml:"source"`

	// Trusted is a list of repos/orgs that don't require confirmation.
	// Examples: "acme-corp" (trusts all repos from org), "user/repo" (specific repo)
	Trusted []string `yaml:"trusted,omitempty"`

	Cache     CacheConfig    `yaml:"cache"`
	Languages LanguageConfig `yaml:"languages,omitempty"`
	Optimize  OptimizeConfig `yaml:"optimize,omitempty"`
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
	if err := c.Source.Validate(); err != nil {
		return errors.ConfigInvalid(err.Error())
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
	if c.Cache.TTL == "" {
		c.Cache.TTL = DefaultCacheTTL
	}
	// Language defaults: if no explicit list and auto_detect not set, enable auto_detect
	if len(c.Languages.Enabled) == 0 && !c.Languages.AutoDetect {
		c.Languages.AutoDetect = true
	}
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

// IsTrustedSource checks if a repo is trusted according to this config.
// It checks both the user's trusted list and the default trusted sources.
func (c *Config) IsTrustedSource(repo string) bool {
	// Check user's trusted list
	if IsTrusted(repo, c.Trusted) {
		return true
	}
	// Check default trusted sources
	return IsTrusted(repo, DefaultTrustedSources)
}

// NewSimpleConfig creates a config with a single source.
func NewSimpleConfig(repo string) *Config {
	return &Config{
		Version: DefaultVersion,
		Source: Source{
			Simple: repo,
		},
		Cache: CacheConfig{
			TTL: DefaultCacheTTL,
		},
	}
}

// DefaultOwnerRepo returns the owner and repo for the default source.
// This is a convenience method for the common single-source case.
func (c *Config) DefaultOwnerRepo() (owner, repo string, err error) {
	return ParseRepo(c.Source.DefaultRepo())
}

// SourceRepo returns the full repo string for display purposes.
func (c *Config) SourceRepo() string {
	return c.Source.DefaultRepo()
}
