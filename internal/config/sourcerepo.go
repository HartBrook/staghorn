// Package config handles staghorn configuration.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File permission constants for consistent file creation.
const (
	DefaultFileMode = 0644
	DefaultDirMode  = 0755
)

// SourceRepoConfig represents a .staghorn/source.yaml file that marks
// a repository as a staghorn source repo (team/community standards).
type SourceRepoConfig struct {
	SourceRepo bool `yaml:"source_repo"`
}

// SourceRepoPaths provides paths for source repo operations.
// These are the standard locations for team repo content.
type SourceRepoPaths struct {
	Root         string // Project root
	ConfigFile   string // .staghorn/source.yaml
	ClaudeMD     string // ./CLAUDE.md
	CommandsDir  string // ./commands/
	LanguagesDir string // ./languages/
	TemplatesDir string // ./templates/
	EvalsDir     string // ./evals/
}

// NewSourceRepoPaths creates SourceRepoPaths for a given project root.
func NewSourceRepoPaths(projectRoot string) *SourceRepoPaths {
	return &SourceRepoPaths{
		Root:         projectRoot,
		ConfigFile:   filepath.Join(projectRoot, ".staghorn", "source.yaml"),
		ClaudeMD:     filepath.Join(projectRoot, "CLAUDE.md"),
		CommandsDir:  filepath.Join(projectRoot, "commands"),
		LanguagesDir: filepath.Join(projectRoot, "languages"),
		TemplatesDir: filepath.Join(projectRoot, "templates"),
		EvalsDir:     filepath.Join(projectRoot, "evals"),
	}
}

// LoadSourceRepoConfig loads .staghorn/source.yaml from the given project root.
// Returns nil and an error if the file doesn't exist or can't be parsed.
func LoadSourceRepoConfig(projectRoot string) (*SourceRepoConfig, error) {
	configPath := filepath.Join(projectRoot, ".staghorn", "source.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg SourceRepoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsSourceRepo checks if the given directory is a staghorn source repo.
// A source repo has .staghorn/source.yaml with source_repo: true.
func IsSourceRepo(projectRoot string) bool {
	if projectRoot == "" {
		return false
	}

	cfg, err := LoadSourceRepoConfig(projectRoot)
	if err != nil {
		return false
	}

	return cfg.SourceRepo
}

// WriteSourceRepoConfig writes a source.yaml file to mark a repo as a source repo.
func WriteSourceRepoConfig(projectRoot string) error {
	staghornDir := filepath.Join(projectRoot, ".staghorn")
	if err := os.MkdirAll(staghornDir, DefaultDirMode); err != nil {
		return err
	}

	content := "# This marks this repository as a staghorn source repo\nsource_repo: true\n"
	return os.WriteFile(filepath.Join(staghornDir, "source.yaml"), []byte(content), DefaultFileMode)
}
